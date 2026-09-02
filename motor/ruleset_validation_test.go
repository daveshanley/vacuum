// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package motor

import (
	"errors"
	"testing"

	"github.com/daveshanley/vacuum/functions"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

type rulesetValidationFunction struct{}

func (rulesetValidationFunction) RunRule(_ []*yaml.Node, _ model.RuleFunctionContext) []model.RuleFunctionResult {
	return nil
}

func (rulesetValidationFunction) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "customFunction"}
}

func (rulesetValidationFunction) GetCategory() string {
	return model.CategoryValidation
}

func TestValidateRuleSetFunctionsReportsUnknownFunctionsDeterministically(t *testing.T) {
	ruleSet := &rulesets.RuleSet{Rules: map[string]*model.Rule{
		"nil-rule": nil,
		"z-rule": {
			Then: []model.RuleAction{
				{Function: "missing-z"},
				{Function: "missing-z"},
			},
		},
		"valid-custom": {
			Then: model.RuleAction{Function: "customFunction"},
		},
		"valid-built-in": {
			Then: map[string]interface{}{"function": "pattern"},
		},
		"valid-action-maps": {
			Then: []map[string]interface{}{{"function": "truthy"}},
		},
		"a-rule": {
			Id: "explicit-rule-id",
			Then: []interface{}{
				map[string]interface{}{"function": "missing-a"},
			},
		},
	}}

	validationErrors := ValidateRuleSetFunctions(ruleSet, map[string]model.RuleFunction{
		"customFunction": rulesetValidationFunction{},
	})

	require.Len(t, validationErrors, 2)
	var first *UnknownRuleFunctionError
	require.ErrorAs(t, validationErrors[0], &first)
	assert.Equal(t, "explicit-rule-id", first.RuleID)
	assert.Equal(t, "missing-a", first.Function)
	assert.Equal(t, "unknown function 'missing-a' in rule 'explicit-rule-id'", first.Error())

	var second *UnknownRuleFunctionError
	require.ErrorAs(t, validationErrors[1], &second)
	assert.Equal(t, "z-rule", second.RuleID)
	assert.Equal(t, "missing-z", second.Function)
}

func TestBuildResultsReportsLateUnknownFunctionAsExecutionError(t *testing.T) {
	var ruleResults []model.RuleFunctionResult
	var executionErrors []error
	ctx := ruleContext{
		rule: &model.Rule{
			Id:    "late-invalid-rule",
			Given: "$.info",
		},
		builtinFunctions: functions.MapBuiltinFunctions(),
		ruleResults:      &ruleResults,
		errors:           &executionErrors,
		silenceLogs:      true,
	}

	results := buildResults(ctx, model.RuleAction{Function: "missing-late"}, nil)

	assert.Empty(t, *results)
	require.Len(t, executionErrors, 1)
	var functionErr *UnknownRuleFunctionError
	require.ErrorAs(t, executionErrors[0], &functionErr)
	assert.Equal(t, "late-invalid-rule", functionErr.RuleID)
	assert.Equal(t, "missing-late", functionErr.Function)
}

func TestValidateRuleSetFunctionsAcceptsEmptyRuleSet(t *testing.T) {
	assert.Empty(t, ValidateRuleSetFunctions(nil, nil))
	assert.Empty(t, ValidateRuleSetFunctions(&rulesets.RuleSet{}, nil))
}

func TestUnknownRuleFunctionErrorNilReceiver(t *testing.T) {
	var functionErr *UnknownRuleFunctionError
	assert.Equal(t, "unknown rule function", functionErr.Error())
}

func TestApplyRulesToRuleSetReportsUnknownFunctionsAsExecutionErrors(t *testing.T) {
	ruleSet := &rulesets.RuleSet{Rules: map[string]*model.Rule{
		"check-title-is-exactly-this": {
			Id:       "check-title-is-exactly-this",
			Severity: model.SeverityInfo,
			Given:    "$.info",
			Then: model.RuleAction{
				Field:    "title",
				Function: "patternd",
			},
		},
	}}
	execution := &RuleSetExecution{
		RuleSet: ruleSet,
		Spec:    []byte("not: [valid"),
	}

	result := ApplyRulesToRuleSet(execution)

	assert.Same(t, execution, result.RuleSetExecution)
	assert.Empty(t, result.Results)
	require.Len(t, result.Errors, 1)
	var functionErr *UnknownRuleFunctionError
	require.True(t, errors.As(result.Errors[0], &functionErr))
	assert.Equal(t, "check-title-is-exactly-this", functionErr.RuleID)
	assert.Equal(t, "patternd", functionErr.Function)
	assert.Nil(t, result.Index)
}
