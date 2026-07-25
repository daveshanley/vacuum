// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package motor

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/functions"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

type atomicCounter struct {
	value atomic.Int32
}

func (c *atomicCounter) Add(delta int32) int32 {
	return c.value.Add(delta)
}

func (c *atomicCounter) Load() int32 {
	return c.value.Load()
}

type channelRuleFunction struct {
	releases map[string]<-chan struct{}
	started  chan string
	finished chan string
	active   atomic.Int32
	max      atomic.Int32
	withNode bool
}

func (c *channelRuleFunction) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "controlled"}
}

func (c *channelRuleFunction) GetCategory() string {
	return model.CategoryValidation
}

func (c *channelRuleFunction) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	id := context.Rule.Id
	active := c.active.Add(1)
	for {
		maximum := c.max.Load()
		if active <= maximum || c.max.CompareAndSwap(maximum, active) {
			break
		}
	}
	if c.started != nil {
		c.started <- id
	}
	if release := c.releases[id]; release != nil {
		<-release
	}
	c.active.Add(-1)
	if c.finished != nil {
		c.finished <- id
	}
	result := model.RuleFunctionResult{Message: id, Path: "$"}
	if c.withNode && len(nodes) > 0 {
		result.StartNode = nodes[0]
	}
	return []model.RuleFunctionResult{result}
}

func TestRuleCancellationReturnsOnlyCompletedResults(t *testing.T) {
	blockB := make(chan struct{})
	blockC := make(chan struct{})
	function := &channelRuleFunction{
		releases: map[string]<-chan struct{}{
			"b": blockB,
			"c": blockC,
		},
		started:  make(chan string, 3),
		finished: make(chan string, 3),
	}
	rules := controlledRules("a", "b", "c")
	caller, cancel := context.WithCancel(context.Background())
	control, err := newExecutionControl(&ExecutionOptions{
		Context:            caller,
		MaxRuleConcurrency: 2,
	})
	require.NoError(t, err)
	defer control.Close()

	resultCh := make(chan []model.RuleFunctionResult, 1)
	go func() {
		results, _, _, _ := runControlledRules(control, rules, function, time.Second)
		resultCh <- results
	}()

	started := receiveRuleIDs(t, function.started, 3)
	assert.ElementsMatch(t, []string{"a", "b", "c"}, started)
	cancel()

	var results []model.RuleFunctionResult
	select {
	case results = <-resultCh:
	case <-time.After(time.Second):
		t.Fatal("cancelled scheduler did not return promptly")
	}
	require.Len(t, results, 1)
	assert.Equal(t, "a", results[0].RuleId)
	assert.Equal(t, "a", results[0].Message)

	returnedMessage := results[0].Message
	close(blockB)
	close(blockC)
	receiveRuleIDs(t, function.finished, 3)
	assert.Len(t, results, 1)
	assert.Equal(t, returnedMessage, results[0].Message)
}

func TestRuleCancellationDoesNotStartQueuedRules(t *testing.T) {
	blockA := make(chan struct{})
	function := &channelRuleFunction{
		releases: map[string]<-chan struct{}{"a": blockA},
		started:  make(chan string, 2),
		finished: make(chan string, 1),
	}
	caller, cancel := context.WithCancel(context.Background())
	control, err := newExecutionControl(&ExecutionOptions{
		Context:            caller,
		MaxRuleConcurrency: 1,
	})
	require.NoError(t, err)
	defer control.Close()

	resultCh := make(chan struct{}, 1)
	go func() {
		runControlledRules(control, controlledRules("a", "b"), function, time.Second)
		resultCh <- struct{}{}
	}()

	assert.Equal(t, "a", <-function.started)
	cancel()
	select {
	case <-resultCh:
	case <-time.After(time.Second):
		t.Fatal("cancelled scheduler did not return promptly")
	}
	select {
	case unexpected := <-function.started:
		t.Fatalf("queued rule %q started after cancellation", unexpected)
	default:
	}
	close(blockA)
	assert.Equal(t, "a", receiveRuleIDs(t, function.finished, 1)[0])
}

func TestRuleCancellationBeforeAutoFixCommitSkipsSharedMutation(t *testing.T) {
	blockRule := make(chan struct{})
	function := &channelRuleFunction{
		releases: map[string]<-chan struct{}{"a": blockRule},
		started:  make(chan string, 1),
		finished: make(chan string, 1),
		withNode: true,
	}
	var autoFixCalls atomic.Int32
	root := &yaml.Node{Kind: yaml.MappingNode}
	execution := &RuleSetExecution{
		Timeout:        time.Second,
		SilenceLogs:    true,
		ApplyAutoFixes: true,
		AutoFixFunctions: map[string]model.AutoFixFunction{
			"fix": func(node, _ *yaml.Node, _ *model.RuleFunctionContext) (*yaml.Node, error) {
				autoFixCalls.Add(1)
				node.Value = "mutated"
				return node, nil
			},
		},
	}
	rule := controlledRules("a")[0]
	rule.AutoFixFunction = "fix"
	caller, cancel := context.WithCancel(context.Background())
	control, err := newExecutionControl(&ExecutionOptions{
		Context:            caller,
		MaxRuleConcurrency: 1,
	})
	require.NoError(t, err)
	defer control.Close()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	resultCh := make(chan []model.RuleFunctionResult, 1)
	go func() {
		results, _, _, _ := runRuleContexts(
			control,
			execution,
			[]*model.Rule{rule},
			logger,
			func(rule *model.Rule) ruleContext {
				return ruleContext{
					rule:               rule,
					specNode:           root,
					specNodeUnresolved: root,
					builtinFunctions:   functions.MapBuiltinFunctions(),
					customFunctions: map[string]model.RuleFunction{
						"controlled": function,
					},
					autoFixFunctions:  execution.AutoFixFunctions,
					applyAutoFixes:    true,
					skipDocumentCheck: true,
					logger:            logger,
				}
			},
		)
		resultCh <- results
	}()

	assert.Equal(t, "a", <-function.started)
	cancel()
	select {
	case results := <-resultCh:
		assert.Empty(t, results)
	case <-time.After(time.Second):
		t.Fatal("cancelled scheduler did not return")
	}
	close(blockRule)
	assert.Equal(t, "a", receiveRuleIDs(t, function.finished, 1)[0])
	select {
	case <-control.ruleRunsDone():
	case <-time.After(time.Second):
		t.Fatal("abandoned rule did not exit")
	}
	assert.Equal(t, int32(0), autoFixCalls.Load())
	assert.Empty(t, root.Value)
}

func TestRuleRunnerHonorsConfiguredConcurrency(t *testing.T) {
	for _, maximum := range []int{1, 2, 8} {
		t.Run(strconv.Itoa(maximum), func(t *testing.T) {
			release := make(chan struct{})
			ruleCount := 5
			function := &channelRuleFunction{
				releases: make(map[string]<-chan struct{}, ruleCount),
				started:  make(chan string, ruleCount),
			}
			ids := make([]string, 0, ruleCount)
			for i := 0; i < ruleCount; i++ {
				id := fmt.Sprintf("rule-%02d", i)
				ids = append(ids, id)
				function.releases[id] = release
			}
			control, err := newExecutionControl(&ExecutionOptions{MaxRuleConcurrency: maximum})
			require.NoError(t, err)
			defer control.Close()

			resultCh := make(chan []model.RuleFunctionResult, 1)
			go func() {
				results, _, _, _ := runControlledRules(
					control, controlledRules(ids...), function, time.Second,
				)
				resultCh <- results
			}()

			wantActive := min(maximum, ruleCount)
			receiveRuleIDs(t, function.started, wantActive)
			assert.Equal(t, int32(wantActive), function.max.Load())
			assert.Equal(t, int32(wantActive), function.active.Load())
			close(release)

			select {
			case results := <-resultCh:
				assert.Len(t, results, ruleCount)
			case <-time.After(time.Second):
				t.Fatal("scheduler did not finish")
			}
			assert.LessOrEqual(t, function.max.Load(), int32(maximum))
		})
	}
}

func TestRuleRunnerDefaultConcurrencyCeiling(t *testing.T) {
	const ruleCount = defaultMaxRuleConcurrency + 8
	release := make(chan struct{})
	function := &channelRuleFunction{
		releases: make(map[string]<-chan struct{}, ruleCount),
		started:  make(chan string, ruleCount),
	}
	ids := make([]string, 0, ruleCount)
	for i := 0; i < ruleCount; i++ {
		id := fmt.Sprintf("rule-%02d", i)
		ids = append(ids, id)
		function.releases[id] = release
	}
	control, err := newExecutionControl(nil)
	require.NoError(t, err)
	defer control.Close()

	resultCh := make(chan []model.RuleFunctionResult, 1)
	go func() {
		results, _, _, _ := runControlledRules(
			control, controlledRules(ids...), function, time.Second,
		)
		resultCh <- results
	}()

	receiveRuleIDs(t, function.started, defaultMaxRuleConcurrency)
	assert.Equal(t, int32(defaultMaxRuleConcurrency), function.active.Load())
	assert.Equal(t, int32(defaultMaxRuleConcurrency), function.max.Load())
	close(release)

	select {
	case results := <-resultCh:
		assert.Len(t, results, ruleCount)
	case <-time.After(time.Second):
		t.Fatal("scheduler did not finish")
	}
	assert.Equal(t, int32(defaultMaxRuleConcurrency), function.max.Load())
}

func TestRuleRunnerPreservesRuleOrder(t *testing.T) {
	releaseA := make(chan struct{})
	releaseB := make(chan struct{})
	releaseC := make(chan struct{})
	function := &channelRuleFunction{
		releases: map[string]<-chan struct{}{
			"a": releaseA,
			"b": releaseB,
			"c": releaseC,
		},
		started: make(chan string, 3),
	}
	control, err := newExecutionControl(&ExecutionOptions{MaxRuleConcurrency: 3})
	require.NoError(t, err)
	defer control.Close()

	resultCh := make(chan []model.RuleFunctionResult, 1)
	go func() {
		results, _, _, _ := runControlledRules(
			control, controlledRules("a", "b", "c"), function, time.Second,
		)
		resultCh <- results
	}()

	receiveRuleIDs(t, function.started, 3)
	close(releaseC)
	close(releaseB)
	close(releaseA)
	var results []model.RuleFunctionResult
	select {
	case results = <-resultCh:
	case <-time.After(time.Second):
		t.Fatal("scheduler did not finish")
	}
	require.Len(t, results, 3)
	assert.Equal(t, []string{"a", "b", "c"}, resultRuleIDs(results))
}

func TestPerRuleTimeoutDoesNotCancelRun(t *testing.T) {
	blockA := make(chan struct{})
	function := &channelRuleFunction{
		releases: map[string]<-chan struct{}{"a": blockA},
		started:  make(chan string, 2),
		finished: make(chan string, 2),
	}
	control, err := newExecutionControl(&ExecutionOptions{MaxRuleConcurrency: 1})
	require.NoError(t, err)
	defer control.Close()

	results, _, _, errs := runControlledRules(
		control, controlledRules("a", "b"), function, 25*time.Millisecond,
	)
	close(blockA)
	receiveRuleIDs(t, function.finished, 2)

	assert.NoError(t, control.Err())
	assert.Empty(t, errs)
	require.Len(t, results, 1)
	assert.Equal(t, "b", results[0].RuleId)
}

func controlledRules(ids ...string) []*model.Rule {
	rules := make([]*model.Rule, 0, len(ids))
	for _, id := range ids {
		rules = append(rules, &model.Rule{
			Id:           id,
			Given:        "$",
			RuleCategory: model.RuleCategories[model.CategoryValidation],
			Type:         rulesets.Validation,
			Severity:     model.SeverityError,
			Then: model.RuleAction{
				Function: "controlled",
			},
		})
	}
	return rules
}

func runControlledRules(
	control *executionControl,
	rules []*model.Rule,
	function model.RuleFunction,
	timeout time.Duration,
) ([]model.RuleFunctionResult, []model.RuleFunctionResult, []model.RuleFunctionResult, []error) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	root := &yaml.Node{Kind: yaml.MappingNode}
	execution := &RuleSetExecution{
		Timeout:     timeout,
		SilenceLogs: true,
	}
	return runRuleContexts(control, execution, rules, logger, func(rule *model.Rule) ruleContext {
		return ruleContext{
			rule:               rule,
			specNode:           root,
			specNodeUnresolved: root,
			builtinFunctions:   functions.MapBuiltinFunctions(),
			customFunctions: map[string]model.RuleFunction{
				"controlled": function,
			},
			skipDocumentCheck: true,
			logger:            logger,
		}
	})
}

func receiveRuleIDs(t *testing.T, channel <-chan string, count int) []string {
	t.Helper()
	ids := make([]string, 0, count)
	deadline := time.After(time.Second)
	for len(ids) < count {
		select {
		case id := <-channel:
			ids = append(ids, id)
		case <-deadline:
			t.Fatalf("received %d of %d expected rule signals", len(ids), count)
		}
	}
	return ids
}

func resultRuleIDs(results []model.RuleFunctionResult) []string {
	ids := make([]string, 0, len(results))
	for _, result := range results {
		ids = append(ids, result.RuleId)
	}
	return ids
}
