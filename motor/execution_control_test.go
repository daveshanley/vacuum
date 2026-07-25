// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package motor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestExecutionControlDefaultsAndValidation(t *testing.T) {
	t.Run("nil options", func(t *testing.T) {
		control, err := newExecutionControl(nil)
		require.NoError(t, err)
		defer control.Close()

		_, hasDeadline := control.Context().Deadline()
		assert.False(t, hasDeadline)
		assert.Equal(t, defaultMaxRuleConcurrency, control.MaxRuleConcurrency())
		assert.NoError(t, control.Err())
	})

	t.Run("reference options preserve defaults", func(t *testing.T) {
		control, err := newExecutionControl(&ExecutionOptions{
			ResolveAllRefs:       true,
			NestedRefsDocContext: true,
		})
		require.NoError(t, err)
		defer control.Close()

		_, hasDeadline := control.Context().Deadline()
		assert.False(t, hasDeadline)
		assert.Equal(t, defaultMaxRuleConcurrency, control.MaxRuleConcurrency())
	})

	t.Run("positive concurrency is exact", func(t *testing.T) {
		control, err := newExecutionControl(&ExecutionOptions{MaxRuleConcurrency: 7})
		require.NoError(t, err)
		defer control.Close()
		assert.Equal(t, 7, control.MaxRuleConcurrency())
	})

	t.Run("negative concurrency is rejected", func(t *testing.T) {
		control, err := newExecutionControl(&ExecutionOptions{MaxRuleConcurrency: -1})
		assert.Nil(t, control)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "max rule concurrency")
	})
}

func TestExecutionControlDeadlinePrecedence(t *testing.T) {
	t.Run("caller deadline wins", func(t *testing.T) {
		caller, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		control, err := newExecutionControl(&ExecutionOptions{
			Context:    caller,
			RunTimeout: time.Hour,
		})
		require.NoError(t, err)
		defer control.Close()

		callerDeadline, _ := caller.Deadline()
		effectiveDeadline, ok := control.Context().Deadline()
		require.True(t, ok)
		assert.Equal(t, callerDeadline, effectiveDeadline)
	})

	t.Run("run timeout wins", func(t *testing.T) {
		caller, cancel := context.WithTimeout(context.Background(), time.Hour)
		defer cancel()

		before := time.Now().Add(500 * time.Millisecond)
		control, err := newExecutionControl(&ExecutionOptions{
			Context:    caller,
			RunTimeout: 500 * time.Millisecond,
		})
		require.NoError(t, err)
		defer control.Close()

		effectiveDeadline, ok := control.Context().Deadline()
		require.True(t, ok)
		assert.WithinDuration(t, before, effectiveDeadline, 100*time.Millisecond)
	})
}

func TestAppendContextErrorOnce(t *testing.T) {
	workErr := errors.New("work failed")
	wrappedCancellation := errors.Join(errors.New("worker stopped"), context.Canceled)
	result := &RuleSetExecutionResult{Errors: []error{workErr, wrappedCancellation}}

	appendContextError(result, context.Canceled)
	require.Len(t, result.Errors, 2)
	assert.Equal(t, workErr, result.Errors[0])
	assert.ErrorIs(t, result.Errors[1], context.Canceled)

	appendContextError(result, context.DeadlineExceeded)
	require.Len(t, result.Errors, 3)
	assert.ErrorIs(t, result.Errors[2], context.DeadlineExceeded)
}

func TestExecutionErrorResultPreservesWorkErrorBeforeCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	control, err := newExecutionControl(&ExecutionOptions{Context: ctx})
	require.NoError(t, err)
	defer control.Close()

	workErr := errors.New("document build failed")
	result := executionErrorResult(&RuleSetExecution{}, control, workErr)
	require.Len(t, result.Errors, 2)
	assert.Equal(t, workErr, result.Errors[0])
	assert.ErrorIs(t, result.Errors[1], context.Canceled)
}

func TestRuleConcurrencyLimit(t *testing.T) {
	tests := []struct {
		name       string
		rules      int
		configured int
		want       int
	}{
		{name: "no rules", rules: 0, configured: 1, want: 0},
		{name: "single default", rules: 1, configured: defaultMaxRuleConcurrency, want: 1},
		{name: "below default", rules: 16, configured: defaultMaxRuleConcurrency, want: 16},
		{name: "above default", rules: 64, configured: defaultMaxRuleConcurrency, want: 32},
		{name: "serial", rules: 16, configured: 1, want: 1},
		{name: "two", rules: 16, configured: 2, want: 2},
		{name: "clamped", rules: 2, configured: 64, want: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ruleConcurrencyLimit(tt.rules, tt.configured))
		})
	}
}

func TestApplyRulesExecutionOptionsRejectNegativeConcurrencyBeforeRuleWork(t *testing.T) {
	counter := &countingRuleFunction{}
	result := ApplyRulesToRuleSetWithOptions(
		newExecutionOptionsTestExecution(counter),
		&ExecutionOptions{MaxRuleConcurrency: -1},
	)

	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0].Error(), "max rule concurrency")
	assert.Equal(t, int32(0), counter.calls.Load())
}

func TestApplyRulesPreCancelledContextAcrossFormats(t *testing.T) {
	tests := []struct {
		name   string
		format string
		spec   string
	}{
		{
			name: "OpenAPI",
			spec: executionOptionsOpenAPISpec,
		},
		{
			name:   "AsyncAPI",
			format: model.AsyncAPI30,
			spec: `asyncapi: 3.0.0
info:
  title: Test
  version: 1.0.0
channels: {}
operations: {}
`,
		},
		{
			name:   "JSON Schema",
			format: model.JSONSchemaDraft2020,
			spec:   `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			counter := &countingRuleFunction{}
			execution := newExecutionOptionsTestExecution(counter)
			execution.Spec = []byte(tt.spec)
			execution.SpecFormat = tt.format

			result := ApplyRulesToRuleSetWithOptions(execution, &ExecutionOptions{Context: ctx})
			require.Len(t, result.Errors, 1)
			assert.ErrorIs(t, result.Errors[0], context.Canceled)
			assert.Equal(t, int32(0), counter.calls.Load())
			result.ReleaseOwnedResources()
			result.ReleaseOwnedResources()
		})
	}
}

func TestApplyRulesCancellationDuringRuleExecutionAcrossFormats(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		formats []string
		spec    string
	}{
		{
			name: "OpenAPI",
			spec: executionOptionsOpenAPISpec,
		},
		{
			name:    "AsyncAPI",
			format:  model.AsyncAPI30,
			formats: model.AsyncAPI3AllFormats,
			spec: `asyncapi: 3.0.0
info:
  title: Test
  version: 1.0.0
channels: {}
operations: {}
`,
		},
		{
			name:    "JSON Schema",
			format:  model.JSONSchemaDraft2020,
			formats: model.JSONSchemaAllFormats,
			spec:    `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocked := make(chan struct{})
			function := &channelRuleFunction{
				releases: map[string]<-chan struct{}{"controlled": blocked},
				started:  make(chan string, 1),
				finished: make(chan string, 1),
			}
			execution := newExecutionOptionsTestExecution(function)
			execution.Spec = []byte(tt.spec)
			execution.SpecFormat = tt.format
			execution.SkipDocumentCheck = isJSONSchemaFormat(tt.format)
			execution.RuleSet.Rules["controlled"].Formats = tt.formats

			ctx, cancel := context.WithCancel(context.Background())
			resultCh := make(chan *RuleSetExecutionResult, 1)
			go func() {
				resultCh <- ApplyRulesToRuleSetWithOptions(execution, &ExecutionOptions{
					Context:            ctx,
					MaxRuleConcurrency: 1,
				})
			}()

			select {
			case <-function.started:
			case earlyResult := <-resultCh:
				t.Fatalf("execution returned before rule start: %v", earlyResult.Errors)
			case <-time.After(3 * time.Second):
				t.Fatal("rule did not start")
			}
			cancel()

			var result *RuleSetExecutionResult
			select {
			case result = <-resultCh:
			case <-time.After(time.Second):
				t.Fatal("cancelled execution did not return promptly")
			}
			assert.Empty(t, result.Results)
			require.Len(t, result.Errors, 1)
			assert.ErrorIs(t, result.Errors[0], context.Canceled)
			assert.NotNil(t, result.Index)
			assert.NotNil(t, result.SpecInfo)
			result.ReleaseOwnedResources()
			result.ReleaseOwnedResources()

			close(blocked)
			select {
			case <-function.finished:
			case <-time.After(time.Second):
				t.Fatal("detached rule did not finish after cleanup")
			}
			assert.Empty(t, result.Results)
		})
	}
}

func TestCancelledResultDefersResourceReleaseUntilRuleExit(t *testing.T) {
	ruleStarted := make(chan struct{})
	releaseRule := make(chan struct{})
	rootObserved := make(chan bool, 1)
	function := &resourceTouchingRuleFunction{
		started:      ruleStarted,
		release:      releaseRule,
		rootObserved: rootObserved,
	}
	rule := controlledRule("resource-touching", "$")
	rule.Then = model.RuleAction{Function: "resourceTouching"}
	execution := &RuleSetExecution{
		RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{rule.Id: rule}},
		Spec:    []byte(executionOptionsOpenAPISpec),
		CustomFunctions: map[string]model.RuleFunction{
			"resourceTouching": function,
		},
		SilenceLogs: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	resultCh := make(chan *RuleSetExecutionResult, 1)
	go func() {
		resultCh <- ApplyRulesToRuleSetWithOptions(execution, &ExecutionOptions{
			Context:            ctx,
			MaxRuleConcurrency: 1,
		})
	}()
	select {
	case <-ruleStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("resource-touching rule did not start")
	}
	cancel()

	var result *RuleSetExecutionResult
	select {
	case result = <-resultCh:
	case <-time.After(time.Second):
		t.Fatal("cancelled execution did not return promptly")
	}
	result.ReleaseOwnedResources()
	result.ReleaseOwnedResources()
	close(releaseRule)

	select {
	case rootWasAvailable := <-rootObserved:
		assert.True(t, rootWasAvailable, "index was released before detached rule exited")
	case <-time.After(time.Second):
		t.Fatal("detached rule did not finish")
	}
}

func TestApplyRulesRunTimeoutReturnsStandardError(t *testing.T) {
	blocked := make(chan struct{})
	function := &channelRuleFunction{
		releases: map[string]<-chan struct{}{"controlled": blocked},
		started:  make(chan string, 1),
		finished: make(chan string, 1),
	}
	execution := newExecutionOptionsTestExecution(function)

	startedAt := time.Now()
	resultCh := make(chan *RuleSetExecutionResult, 1)
	go func() {
		resultCh <- ApplyRulesToRuleSetWithOptions(execution, &ExecutionOptions{
			RunTimeout:         250 * time.Millisecond,
			MaxRuleConcurrency: 1,
		})
	}()
	select {
	case <-function.started:
	case result := <-resultCh:
		t.Fatalf("run timeout expired before the controlled rule started: %v", result.Errors)
	case <-time.After(3 * time.Second):
		t.Fatal("controlled rule did not start")
	}

	var result *RuleSetExecutionResult
	select {
	case result = <-resultCh:
	case <-time.After(time.Second):
		t.Fatal("run timeout did not return promptly")
	}
	elapsed := time.Since(startedAt)
	close(blocked)
	select {
	case finished := <-function.finished:
		assert.Equal(t, "controlled", finished)
	case <-time.After(time.Second):
		t.Fatal("detached rule did not finish after cleanup")
	}

	assert.Less(t, elapsed, time.Second)
	assert.Empty(t, result.Results)
	require.Len(t, result.Errors, 1)
	assert.ErrorIs(t, result.Errors[0], context.DeadlineExceeded)
}

func TestApplyRulesCancellationPreservesCompletedIgnoredResults(t *testing.T) {
	blockB := make(chan struct{})
	blockC := make(chan struct{})
	function := &channelRuleFunction{
		releases: map[string]<-chan struct{}{
			"b-blocked": blockB,
			"c-blocked": blockC,
		},
		started:  make(chan string, 3),
		finished: make(chan string, 3),
	}
	rules := map[string]*model.Rule{
		"a-ignored": controlledRule("a-ignored", "$"),
		"b-blocked": controlledRule("b-blocked", "$"),
		"c-blocked": controlledRule("c-blocked", "$"),
	}
	execution := controlledExecution(
		`openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
paths: {}
x-lint-ignore: a-ignored
`,
		rules,
		function,
	)
	ctx, cancel := context.WithCancel(context.Background())
	resultCh := make(chan *RuleSetExecutionResult, 1)
	go func() {
		resultCh <- ApplyRulesToRuleSetWithOptions(execution, &ExecutionOptions{
			Context:            ctx,
			MaxRuleConcurrency: 2,
		})
	}()

	assert.ElementsMatch(
		t,
		[]string{"a-ignored", "b-blocked", "c-blocked"},
		receiveRuleIDs(t, function.started, 3),
	)
	cancel()
	result := receiveExecutionResult(t, resultCh)

	require.Len(t, result.IgnoredResults, 1)
	assert.Equal(t, "a-ignored", result.IgnoredResults[0].RuleId)
	require.Len(t, result.Errors, 1)
	assert.ErrorIs(t, result.Errors[0], context.Canceled)

	close(blockB)
	close(blockC)
	receiveRuleIDs(t, function.finished, 3)
}

func TestApplyRulesCancellationPreservesCompletedFixes(t *testing.T) {
	blockB := make(chan struct{})
	blockC := make(chan struct{})
	function := &channelRuleFunction{
		releases: map[string]<-chan struct{}{
			"b-blocked": blockB,
			"c-blocked": blockC,
		},
		started:  make(chan string, 2),
		finished: make(chan string, 2),
	}
	fixRule := &model.Rule{
		Id:              "a-fix",
		Given:           "$.info.description",
		RuleCategory:    model.RuleCategories[model.CategoryValidation],
		Type:            rulesets.Validation,
		Severity:        model.SeverityError,
		AutoFixFunction: "fixDescription",
		Then: model.RuleAction{
			Function: "truthy",
		},
	}
	rules := map[string]*model.Rule{
		"a-fix":     fixRule,
		"b-blocked": controlledRule("b-blocked", "$"),
		"c-blocked": controlledRule("c-blocked", "$"),
	}
	execution := controlledExecution(
		`openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
  description: ""
paths: {}
`,
		rules,
		function,
	)
	execution.ApplyAutoFixes = true
	execution.AutoFixFunctions = map[string]model.AutoFixFunction{
		"fixDescription": func(
			node *yaml.Node,
			_ *yaml.Node,
			_ *model.RuleFunctionContext,
		) (*yaml.Node, error) {
			node.Value = "fixed before cancellation"
			return node, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	resultCh := make(chan *RuleSetExecutionResult, 1)
	go func() {
		resultCh <- ApplyRulesToRuleSetWithOptions(execution, &ExecutionOptions{
			Context:            ctx,
			MaxRuleConcurrency: 2,
		})
	}()

	assert.ElementsMatch(t, []string{"b-blocked", "c-blocked"}, receiveRuleIDs(t, function.started, 2))
	cancel()
	result := receiveExecutionResult(t, resultCh)

	require.Len(t, result.FixedResults, 1)
	assert.Equal(t, "a-fix", result.FixedResults[0].RuleId)
	assert.Contains(t, string(result.ModifiedSpec), "fixed before cancellation")
	require.Len(t, result.Errors, 1)
	assert.ErrorIs(t, result.Errors[0], context.Canceled)

	close(blockB)
	close(blockC)
	receiveRuleIDs(t, function.finished, 2)
}

func TestApplyRulesCancellationDoesNotWaitForStartedAutoFixCallback(t *testing.T) {
	autoFixStarted := make(chan struct{})
	releaseAutoFix := make(chan struct{})
	autoFixFinished := make(chan struct{})
	fixRule := &model.Rule{
		Id:              "fix-description",
		Given:           "$.info.description",
		RuleCategory:    model.RuleCategories[model.CategoryValidation],
		Type:            rulesets.Validation,
		Severity:        model.SeverityError,
		AutoFixFunction: "fixDescription",
		Then: model.RuleAction{
			Function: "truthy",
		},
	}
	execution := controlledExecution(
		`openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
  description: ""
paths: {}
`,
		map[string]*model.Rule{fixRule.Id: fixRule},
		&countingRuleFunction{},
	)
	execution.ApplyAutoFixes = true
	execution.AutoFixFunctions = map[string]model.AutoFixFunction{
		"fixDescription": func(
			node *yaml.Node,
			document *yaml.Node,
			_ *model.RuleFunctionContext,
		) (*yaml.Node, error) {
			close(autoFixStarted)
			node.Value = "private mutation"
			if document != nil {
				document.Value = "private document mutation"
			}
			<-releaseAutoFix
			close(autoFixFinished)
			return node, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	resultCh := make(chan *RuleSetExecutionResult, 1)
	go func() {
		resultCh <- ApplyRulesToRuleSetWithOptions(execution, &ExecutionOptions{
			Context:            ctx,
			MaxRuleConcurrency: 1,
		})
	}()

	select {
	case <-autoFixStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("auto-fix did not start")
	}
	cancel()

	var result *RuleSetExecutionResult
	select {
	case result = <-resultCh:
	case <-time.After(time.Second):
		t.Fatal("cancelled execution waited for a private auto-fix callback")
	}
	assert.Empty(t, result.FixedResults)
	assert.Empty(t, result.ModifiedSpec)
	require.Len(t, result.Errors, 1)
	assert.ErrorIs(t, result.Errors[0], context.Canceled)

	close(releaseAutoFix)
	select {
	case <-autoFixFinished:
	case <-time.After(time.Second):
		t.Fatal("detached auto-fix callback did not finish")
	}
	select {
	case <-result.ruleRunsDone:
	case <-time.After(time.Second):
		t.Fatal("detached auto-fix rule did not exit")
	}
	assert.Empty(t, result.FixedResults)
	assert.Empty(t, result.ModifiedSpec)
	assert.NotContains(t, string(execution.Spec), "private mutation")
	if execution.CanonicalDocument != nil {
		rendered, err := yaml.Marshal(execution.CanonicalDocument)
		require.NoError(t, err)
		assert.NotContains(t, string(rendered), "private mutation")
	}
}

func TestApplyRulesCancellationPreservesCompletedErrors(t *testing.T) {
	blockB := make(chan struct{})
	blockC := make(chan struct{})
	function := &channelRuleFunction{
		releases: map[string]<-chan struct{}{
			"b-blocked": blockB,
			"c-blocked": blockC,
		},
		started:  make(chan string, 2),
		finished: make(chan string, 2),
	}
	rules := map[string]*model.Rule{
		"a-lookup-error": controlledRule("a-lookup-error", "$..paths"),
		"b-blocked":      controlledRule("b-blocked", "$"),
		"c-blocked":      controlledRule("c-blocked", "$"),
	}
	execution := controlledExecution(executionOptionsOpenAPISpec, rules, function)
	execution.NodeLookupTimeout = time.Nanosecond

	ctx, cancel := context.WithCancel(context.Background())
	resultCh := make(chan *RuleSetExecutionResult, 1)
	go func() {
		resultCh <- ApplyRulesToRuleSetWithOptions(execution, &ExecutionOptions{
			Context:            ctx,
			MaxRuleConcurrency: 2,
		})
	}()

	assert.ElementsMatch(t, []string{"b-blocked", "c-blocked"}, receiveRuleIDs(t, function.started, 2))
	cancel()
	result := receiveExecutionResult(t, resultCh)

	require.Len(t, result.Errors, 2)
	var lookupErr *RuleLookupError
	assert.ErrorAs(t, result.Errors[0], &lookupErr)
	assert.ErrorIs(t, result.Errors[1], context.Canceled)

	close(blockB)
	close(blockC)
	receiveRuleIDs(t, function.finished, 2)
}

const executionOptionsOpenAPISpec = `openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
paths: {}
`

func newExecutionOptionsTestExecution(function model.RuleFunction) *RuleSetExecution {
	rule := controlledRule("controlled", "$")
	return controlledExecution(executionOptionsOpenAPISpec, map[string]*model.Rule{rule.Id: rule}, function)
}

func controlledRule(id, given string) *model.Rule {
	return &model.Rule{
		Id:           id,
		Given:        given,
		RuleCategory: model.RuleCategories[model.CategoryValidation],
		Type:         rulesets.Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "controlled",
		},
	}
}

func controlledExecution(
	spec string,
	rules map[string]*model.Rule,
	function model.RuleFunction,
) *RuleSetExecution {
	return &RuleSetExecution{
		RuleSet: &rulesets.RuleSet{Rules: rules},
		Spec:    []byte(spec),
		CustomFunctions: map[string]model.RuleFunction{
			"controlled": function,
		},
		SilenceLogs: true,
	}
}

type countingRuleFunction struct {
	calls atomicCounter
}

func (c *countingRuleFunction) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "controlled"}
}

func (c *countingRuleFunction) GetCategory() string {
	return model.CategoryValidation
}

func (c *countingRuleFunction) RunRule(_ []*yaml.Node, _ model.RuleFunctionContext) []model.RuleFunctionResult {
	c.calls.Add(1)
	return nil
}

type resourceTouchingRuleFunction struct {
	started      chan struct{}
	release      chan struct{}
	rootObserved chan bool
}

func (r *resourceTouchingRuleFunction) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "resourceTouching"}
}

func (r *resourceTouchingRuleFunction) GetCategory() string {
	return model.CategoryValidation
}

func (r *resourceTouchingRuleFunction) RunRule(
	_ []*yaml.Node,
	context model.RuleFunctionContext,
) []model.RuleFunctionResult {
	close(r.started)
	<-r.release
	r.rootObserved <- context.Index != nil && context.Index.GetRootNode() != nil
	return nil
}

func receiveExecutionResult(
	t *testing.T,
	results <-chan *RuleSetExecutionResult,
) *RuleSetExecutionResult {
	t.Helper()
	select {
	case result := <-results:
		return result
	case <-time.After(time.Second):
		t.Fatal("execution did not return")
		return nil
	}
}
