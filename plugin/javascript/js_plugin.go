// Copyright 2023-2025 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package javascript

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/model/reports"
	"github.com/daveshanley/vacuum/plugin/javascript/eventloop"
	"github.com/daveshanley/vacuum/plugin/javascript/eventloop/fetch"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
	"github.com/mitchellh/mapstructure"
	"github.com/pb33f/libopenapi/index"
	"go.yaml.in/yaml/v4"
)

// DefaultRuleTimeout is the default timeout for JavaScript rule execution
const DefaultRuleTimeout = 5 * time.Second

type CoreFunction func(input any, context model.RuleFunctionContext) []model.RuleFunctionResult

type JSEnabledRuleFunction interface {
	RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult
	GetSchema() model.RuleFunctionSchema
	CheckScript() error
	RunScript() error
	RegisterCoreFunction(name string, function CoreFunction)
	GetCategory() string
}

type JSRuleFunction struct {
	ruleName      string
	script        string
	scriptParsed  bool
	runtime       *goja.Runtime
	coreFunctions map[string]interface{}
	l             sync.Mutex
	ruleTimeout   time.Duration // timeout for async rule execution
}

// SetTimeout sets the timeout for async rule execution.
// If not set, DefaultRuleTimeout (5 seconds) is used.
func (j *JSRuleFunction) SetTimeout(timeout time.Duration) {
	j.ruleTimeout = timeout
}

// getTimeout returns the configured timeout or the default
func (j *JSRuleFunction) getTimeout() time.Duration {
	if j.ruleTimeout > 0 {
		return j.ruleTimeout
	}
	return DefaultRuleTimeout
}

// createErrorResult creates a single error result with the given message
func (j *JSRuleFunction) createErrorResult(message string, node *yaml.Node, ruleContext model.RuleFunctionContext) []model.RuleFunctionResult {
	return []model.RuleFunctionResult{
		{
			Message:   message,
			StartNode: node,
			EndNode:   node,
			Path:      fmt.Sprint(ruleContext.Given),
			Rule:      ruleContext.Rule,
		},
	}
}

// batchNodeInfo stores node info for batch result mapping (rolodex-aware)
type batchNodeInfo struct {
	node   *yaml.Node
	origin *index.NodeOrigin
}

func NewJSRuleFunction(ruleName, script string) JSEnabledRuleFunction {
	rt := BuildVM()

	return &JSRuleFunction{
		ruleName: ruleName,
		script:   script,
		runtime:  rt,
	}
}

func (j *JSRuleFunction) RegisterCoreFunction(name string, function CoreFunction) {
	if j.coreFunctions == nil {
		j.coreFunctions = make(map[string]interface{})
	}
	j.coreFunctions[name] = function
}

func (j *JSRuleFunction) RunScript() error {
	j.l.Lock()
	defer j.l.Unlock()
	return j.runScriptUnsafe()
}

func (j *JSRuleFunction) runScriptUnsafe() error {
	_, err := j.runtime.RunString(j.script)
	if err != nil {
		return err
	}
	j.scriptParsed = true
	return nil
}

func (j *JSRuleFunction) GetCategory() string {
	return model.FunctionCategoryCustomJS
}

func (j *JSRuleFunction) GetSchema() model.RuleFunctionSchema {
	j.l.Lock()
	defer j.l.Unlock()

	var ok bool
	var schemaFunc goja.Callable
	basic := model.RuleFunctionSchema{
		Name: j.ruleName,
	}

	if !j.scriptParsed {
		err := j.runScriptUnsafe()
		if err != nil {

			return basic
		}
	}

	if schemaFunc, ok = goja.AssertFunction(j.runtime.Get("getSchema")); !ok {
		return basic
	}
	schema, sErr := schemaFunc(goja.Undefined())
	if sErr != nil {
		return basic
	}

	var decoded model.RuleFunctionSchema
	err := mapstructure.Decode(schema.Export(), &decoded)
	if err != nil {
		return basic
	}
	return decoded
}

func (j *JSRuleFunction) CheckScript() error {
	j.l.Lock()
	defer j.l.Unlock()

	if !j.scriptParsed {
		err := j.runScriptUnsafe()
		if err != nil {
			return err
		}
	}
	if _, ok := goja.AssertFunction(j.runtime.Get("runRule")); !ok {
		return fmt.Errorf("runRule function not found")
	}
	return nil
}

func (j *JSRuleFunction) RunRule(nodes []*yaml.Node, ruleContext model.RuleFunctionContext) []model.RuleFunctionResult {
	// check if batch mode is enabled - if so, pass all nodes at once
	if vacuumUtils.IsBatchMode(ruleContext.Options) {
		return j.runBatch(nodes, ruleContext)
	}

	// per-node invocation (default behavior)
	// each rule needs its own runtime because these functions may run concurrently.
	// the same runtime would become polluted with a shared state.
	rt := BuildVM()

	loop := eventloop.New(rt)

	// register fetch() function with configuration from rule context (or secure defaults)
	fetchModule, fetchErr := fetch.NewFetchModuleFromConfig(loop, ruleContext.FetchConfig)
	if fetchErr != nil {
		var firstNode *yaml.Node
		if len(nodes) > 0 {
			firstNode = nodes[0]
		}
		return j.createErrorResult(
			fmt.Sprintf("Failed to configure fetch() for JavaScript function '%s': %s", j.ruleName, fetchErr.Error()),
			firstNode, ruleContext)
	}
	fetchModule.Register()

	ctx, cancel := context.WithTimeout(context.Background(), j.getTimeout())
	defer cancel()

	var results []model.RuleFunctionResult
	var runtimeErr error

	for _, node := range nodes {
		var enc interface{}
		_ = node.Decode(&enc)

		runtimeErr = rt.Set("context", ruleContext)
		if runtimeErr != nil {
			return j.createErrorResult(
				fmt.Sprintf("Unable to set context in JavaScript function: '%s': %s", j.ruleName, runtimeErr.Error()),
				node, ruleContext)
		}

		_, runtimeErr = rt.RunString(j.script)

		for name, function := range j.coreFunctions {
			if coreErr := rt.Set(fmt.Sprintf("vacuum_%s", name), function); coreErr != nil {
				return j.createErrorResult(
					fmt.Sprintf("Unable to set core vacuum function '%s': '%s': %s", name, j.ruleName, coreErr.Error()),
					node, ruleContext)
			}
		}

		runRuleFn, ok := goja.AssertFunction(rt.Get("runRule"))
		if !ok {
			return j.createErrorResult(
				fmt.Sprintf("'runRule' is not defined as a JavaScript function: '%s'", j.ruleName),
				node, ruleContext)
		}

		runtimeValue := rt.ToValue(enc)
		ruleOutput, rErr := loop.Run(ctx, func(vm *goja.Runtime) (goja.Value, error) {
			return runRuleFn(goja.Undefined(), runtimeValue)
		})

		if rErr != nil {
			if errors.Is(rErr, context.DeadlineExceeded) {
				return j.createErrorResult(
					fmt.Sprintf("JavaScript function '%s' timed out after %v", j.ruleName, j.getTimeout()),
					node, ruleContext)
			}
			var jsErr *goja.Exception
			if errors.As(rErr, &jsErr) {
				return j.createErrorResult(
					fmt.Sprintf("Unable to execute JavaScript function: '%s': %s", j.ruleName, jsErr.Value().String()),
					node, ruleContext)
			}
			return j.createErrorResult(
				fmt.Sprintf("JavaScript function '%s' failed: %s", j.ruleName, rErr.Error()),
				node, ruleContext)
		}

		functionResults, extractErr := j.extractResults(ruleOutput, node, ruleContext)
		if extractErr != nil {
			return j.createErrorResult(
				fmt.Sprintf("Unable to extract results from JavaScript function: '%s': %s", j.ruleName, extractErr.Error()),
				node, ruleContext)
		}

		results = append(results, functionResults...)
	}
	return results
}

// extractResults handles both synchronous values and Promise results
func (j *JSRuleFunction) extractResults(value goja.Value, node *yaml.Node, ruleContext model.RuleFunctionContext) ([]model.RuleFunctionResult, error) {
	exported, err := eventloop.ExtractPromiseValue(value)
	if err != nil {
		return nil, err
	}

	var functionResults []model.RuleFunctionResult
	if err := mapstructure.Decode(exported, &functionResults); err != nil {
		return nil, fmt.Errorf("unable to decode results: %w", err)
	}

	// populate node information for each result
	for i := range functionResults {
		functionResults[i].StartNode = node
		functionResults[i].EndNode = node
		functionResults[i].Range = reports.Range{
			Start: reports.RangeItem{
				Line: node.Line,
				Char: node.Column,
			},
			End: reports.RangeItem{
				Line: node.Line,
				Char: node.Column,
			},
		}
		functionResults[i].Path = fmt.Sprint(ruleContext.Given)
		functionResults[i].Rule = ruleContext.Rule
	}

	return functionResults, nil
}

// runBatch executes the JS function with all nodes at once (batch mode)
func (j *JSRuleFunction) runBatch(nodes []*yaml.Node, ruleContext model.RuleFunctionContext) []model.RuleFunctionResult {
	if len(nodes) == 0 {
		return nil
	}

	rt := BuildVM()
	loop := eventloop.New(rt)

	// register fetch
	fetchModule, fetchErr := fetch.NewFetchModuleFromConfig(loop, ruleContext.FetchConfig)
	if fetchErr != nil {
		return j.createErrorResult(
			fmt.Sprintf("Failed to configure fetch() for batch: %s", fetchErr.Error()),
			nodes[0], ruleContext)
	}
	fetchModule.Register()

	ctx, cancel := context.WithTimeout(context.Background(), j.getTimeout())
	defer cancel()

	// build batch input with tracking for result mapping
	nodeInfos := make([]batchNodeInfo, len(nodes))
	batchInputs := make([]map[string]interface{}, 0, len(nodes))

	for i, node := range nodes {
		var decoded interface{}
		_ = node.Decode(&decoded)

		// get origin from index (handles multi-file specs via rolodex)
		var origin *index.NodeOrigin
		if ruleContext.Index != nil {
			origin = ruleContext.Index.FindNodeOrigin(node)
		}

		nodeInfos[i] = batchNodeInfo{node: node, origin: origin}
		batchInputs = append(batchInputs, map[string]interface{}{
			"value": decoded,
			"index": i,
		})
	}

	// setup runtime and get runRule function
	if err := rt.Set("context", ruleContext); err != nil {
		return j.createErrorResult(
			fmt.Sprintf("Unable to set context in batch: %s", err.Error()),
			nodes[0], ruleContext)
	}

	if _, err := rt.RunString(j.script); err != nil {
		return j.createErrorResult(
			fmt.Sprintf("Unable to run script in batch: %s", err.Error()),
			nodes[0], ruleContext)
	}

	for name, function := range j.coreFunctions {
		if coreErr := rt.Set(fmt.Sprintf("vacuum_%s", name), function); coreErr != nil {
			return j.createErrorResult(
				fmt.Sprintf("Unable to set core vacuum function '%s': '%s': %s", name, j.ruleName, coreErr.Error()),
				nodes[0], ruleContext)
		}
	}

	runRuleFn, ok := goja.AssertFunction(rt.Get("runRule"))
	if !ok {
		return j.createErrorResult(
			"'runRule' is not defined as a JavaScript function",
			nodes[0], ruleContext)
	}

	// single invocation with all inputs
	runtimeValue := rt.ToValue(batchInputs)
	ruleOutput, rErr := loop.Run(ctx, func(vm *goja.Runtime) (goja.Value, error) {
		return runRuleFn(goja.Undefined(), runtimeValue)
	})

	if rErr != nil {
		if errors.Is(rErr, context.DeadlineExceeded) {
			return j.createErrorResult(
				fmt.Sprintf("Batch function timed out after %v", j.getTimeout()),
				nodes[0], ruleContext)
		}
		return j.createErrorResult(
			fmt.Sprintf("Batch function failed: %s", rErr.Error()),
			nodes[0], ruleContext)
	}

	return j.extractBatchResults(ruleOutput, nodeInfos, ruleContext)
}

// extractBatchResults handles batch mode results with deterministic `input.index` matching.
// Batch mode requires each result to include the original input object for node mapping.
func (j *JSRuleFunction) extractBatchResults(
	value goja.Value,
	nodeInfos []batchNodeInfo,
	ruleContext model.RuleFunctionContext,
) []model.RuleFunctionResult {
	if len(nodeInfos) == 0 {
		return nil
	}

	// extract the value, handling Promises
	exported, err := eventloop.ExtractPromiseValue(value)
	if err != nil {
		return j.createErrorResult(
			fmt.Sprintf("Failed to extract batch results: %s", err.Error()),
			nodeInfos[0].node, ruleContext)
	}

	// results should be an array of objects with {message, input, ...}
	exportedArr, ok := exported.([]interface{})
	if !ok {
		return j.createErrorResult(
			"Batch function must return an array of results",
			nodeInfos[0].node, ruleContext)
	}

	var functionResults []model.RuleFunctionResult
	for itemIdx, item := range exportedArr {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			functionResults = append(functionResults, model.RuleFunctionResult{
				Message:   fmt.Sprintf("Batch result at position %d is not an object", itemIdx),
				StartNode: nodeInfos[0].node,
				EndNode:   nodeInfos[0].node,
				Path:      fmt.Sprint(ruleContext.Given),
				Rule:      ruleContext.Rule,
			})
			continue
		}

		// extract index from input object, batch mode requires returning the original input for deterministic node mapping
		nodeIdx := -1 // Invalid by default
		if inputObj, hasInput := itemMap["input"]; hasInput {
			if inputMap, ok := inputObj.(map[string]interface{}); ok {
				switch idx := inputMap["index"].(type) {
				case float64:
					nodeIdx = int(idx)
				case int64:
					nodeIdx = int(idx)
				case int:
					nodeIdx = idx
				}
			}
			delete(itemMap, "input") // Remove before mapstructure decode
		}

		// validate index
		if nodeIdx < 0 || nodeIdx >= len(nodeInfos) {
			functionResults = append(functionResults, model.RuleFunctionResult{
				Message:   fmt.Sprintf("Batch result at position %d missing valid 'input' object with 'index' field (batch mode requires returning the input)", itemIdx),
				StartNode: nodeInfos[0].node,
				EndNode:   nodeInfos[0].node,
				Path:      fmt.Sprint(ruleContext.Given),
				Rule:      ruleContext.Rule,
			})
			continue
		}

		var result model.RuleFunctionResult
		if err := mapstructure.Decode(itemMap, &result); err != nil {
			functionResults = append(functionResults, model.RuleFunctionResult{
				Message:   fmt.Sprintf("Failed to decode batch result at position %d: %s", itemIdx, err.Error()),
				StartNode: nodeInfos[0].node,
				EndNode:   nodeInfos[0].node,
				Path:      fmt.Sprint(ruleContext.Given),
				Rule:      ruleContext.Rule,
			})
			continue
		}

		// populate node info from matched nodeInfo (includes origin from rolodex)
		info := nodeInfos[nodeIdx]
		result.StartNode = info.node
		result.EndNode = info.node
		result.Origin = info.origin
		result.Range = reports.Range{
			Start: reports.RangeItem{Line: info.node.Line, Char: info.node.Column},
			End:   reports.RangeItem{Line: info.node.Line, Char: info.node.Column},
		}
		result.Path = fmt.Sprint(ruleContext.Given)
		result.Rule = ruleContext.Rule

		functionResults = append(functionResults, result)
	}

	return functionResults
}

// BuildVM returns a new goja runtime, a VM if you will.
func BuildVM() *goja.Runtime {
	rt := goja.New()
	rt.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
	reg := new(require.Registry)
	reg.Enable(rt)
	console.Enable(rt)
	return rt
}
