// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package javascript

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/model/reports"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v3"
)

type CoreFunction func(input any, context model.RuleFunctionContext) []model.RuleFunctionResult

type JSEnabledRuleFunction interface {
	RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult
	GetSchema() model.RuleFunctionSchema
	CheckScript() error
	RunScript() error
	RegisterCoreFunction(name string, function CoreFunction)
}

type JSRuleFunction struct {
	ruleName      string
	script        string
	scriptParsed  bool
	runtime       *goja.Runtime
	coreFunctions map[string]interface{}
}

func NewJSRuleFunction(ruleName, script string) JSEnabledRuleFunction {
	rt := goja.New()
	rt.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
	reg := new(require.Registry)
	reg.Enable(rt)
	console.Enable(rt)

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
	_, err := j.runtime.RunString(j.script)
	if err != nil {
		return err
	}
	j.scriptParsed = true
	return nil
}

func (j *JSRuleFunction) GetSchema() model.RuleFunctionSchema {
	var ok bool
	var schemaFunc goja.Callable
	basic := model.RuleFunctionSchema{
		Name: j.ruleName,
	}

	if !j.scriptParsed {
		err := j.RunScript()
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
	if !j.scriptParsed {
		err := j.RunScript()
		if err != nil {
			return err
		}
	}
	if _, ok := goja.AssertFunction(j.runtime.Get("runRule")); !ok {
		return fmt.Errorf("runRule function not found")
	}
	return nil
}

func (j *JSRuleFunction) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	for _, node := range nodes {

		var enc interface{}
		_ = node.Decode(&enc)

		runtimeErr := j.runtime.Set("context", context)
		if runtimeErr != nil {
			return []model.RuleFunctionResult{
				{
					Message: fmt.Sprintf("Unable to set context in JavaScript function: '%s': %s ",
						j.ruleName, runtimeErr.Error()),
					StartNode: node,
					EndNode:   node,
					Path:      fmt.Sprint(context.Given),
					Rule:      context.Rule,
				},
			}
		}

		// register core functions
		for name, function := range j.coreFunctions {
			coreErr := j.runtime.Set(fmt.Sprintf("vacuum_%s", name), function)
			if coreErr != nil {
				return []model.RuleFunctionResult{
					{
						Message: fmt.Sprintf("Unable to set core vacuum function '%s': '%s': %s ",
							name, j.ruleName, coreErr.Error()),
						StartNode: node,
						EndNode:   node,
						Path:      fmt.Sprint(context.Given),
						Rule:      context.Rule,
					},
				}
			}
		}

		runRule, ok := goja.AssertFunction(j.runtime.Get("runRule"))
		if !ok {
			return []model.RuleFunctionResult{
				{
					Message: fmt.Sprintf("'runRule' is not defined as a JavaScript function: '%s': %s ",
						j.ruleName, runtimeErr.Error()),
					StartNode: node,
					EndNode:   node,
					Path:      fmt.Sprint(context.Given),
					Rule:      context.Rule,
				},
			}
		}
		var functionResults []model.RuleFunctionResult

		// run JS rule!
		runtimeValue := j.runtime.ToValue(enc)
		ruleOutput, rErr := runRule(goja.Undefined(), runtimeValue)
		if rErr != nil {
			if jserr, okE := rErr.(*goja.Exception); okE {
				return []model.RuleFunctionResult{
					{
						Message: fmt.Sprintf("Unable to execute JavaScript function: '%s': %s",
							j.ruleName, jserr.Value().String()),
						StartNode: node,
						EndNode:   node,
						Path:      fmt.Sprint(context.Given),
						Rule:      context.Rule,
					},
				}
			}
			panic(rErr) // not an exception
		}
		op := ruleOutput.Export()
		rErr = mapstructure.Decode(op, &functionResults)
		if rErr != nil {
			return []model.RuleFunctionResult{
				{
					Message: fmt.Sprintf("Unable to decode results from JavaScript function: '%s': %s ",
						j.ruleName, runtimeErr.Error()),
					StartNode: node,
					EndNode:   node,
					Path:      fmt.Sprint(context.Given),
					Rule:      context.Rule,
				},
			}
		}

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
			functionResults[i].Path = fmt.Sprint(context.Given)
			functionResults[i].Rule = context.Rule
		}
		results = append(results, functionResults...)
	}
	return results
}
