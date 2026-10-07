package motor

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/daveshanley/vacuum/functions"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

type configurationProbe struct{ calls atomic.Int32 }

func (p *configurationProbe) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "probe"}
}
func (p *configurationProbe) GetCategory() string { return model.FunctionCategoryCore }
func (p *configurationProbe) RunRule(_ []*yaml.Node, _ model.RuleFunctionContext) []model.RuleFunctionResult {
	p.calls.Add(1)
	return nil
}

func TestUnknownFunction_IsConfigurationErrorBeforeExecution(t *testing.T) {
	for _, severity := range []string{"error", "warn", "info", "hint"} {
		for _, given := range []string{"$", "$.missing"} {
			t.Run(severity+given, func(t *testing.T) {
				probe := &configurationProbe{}
				rs := &rulesets.RuleSet{Rules: map[string]*model.Rule{
					"a-valid":   {Id: "a-valid", Given: "$", Then: model.RuleAction{Function: "probe"}},
					"b-invalid": {Id: "b-invalid", Severity: severity, Given: given, Then: []model.RuleAction{{Function: "truthy"}, {Function: "patternd"}}},
				}}
				result := ApplyRulesToRuleSet(&RuleSetExecution{
					RuleSet: rs, Spec: []byte("openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}"),
					CustomFunctions: map[string]model.RuleFunction{"probe": probe},
				})
				defer result.Release()
				require.Empty(t, result.Results)
				require.Len(t, result.Errors, 1)
				var unknown *UnknownFunctionError
				require.ErrorAs(t, result.Errors[0], &unknown)
				require.Equal(t, "b-invalid", unknown.RuleID)
				require.Equal(t, "patternd", unknown.Function)
				require.Zero(t, probe.calls.Load(), "no rule should run with invalid configuration")
			})
		}
	}
}

func TestUnknownFunction_ActionRepresentations(t *testing.T) {
	action := model.RuleAction{Function: "missing"}
	for i, then := range []any{
		action, &action, []model.RuleAction{action}, []*model.RuleAction{&action},
		map[string]any{"function": "missing"}, []any{map[string]any{"function": "missing"}},
		map[string]any{"Function": "missing"}, map[string]any{"FUNCTION": "missing"},
		map[any]any{"function": "missing"}, []map[any]any{{"function": "missing"}},
		map[string]string{"function": "missing"}, []map[string]string{{"function": "missing"}},
	} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			rs := &rulesets.RuleSet{Rules: map[string]*model.Rule{"check": {Given: "$.missing", Then: then}}}
			errs := validateRuleFunctions(rs, functions.MapBuiltinFunctions(), nil)
			require.Len(t, errs, 1)
			require.EqualError(t, errs[0], `rule "check" uses unknown function "missing"`)
			result := ApplyRulesToRuleSet(&RuleSetExecution{RuleSet: rs})
			defer result.Release()
			require.Len(t, result.Errors, 1)
			var unknown *UnknownFunctionError
			require.ErrorAs(t, result.Errors[0], &unknown)
		})
	}
}

func TestRuleFunctions_RegisteredAndDisabled(t *testing.T) {
	probe := &configurationProbe{}
	rs := &rulesets.RuleSet{Rules: map[string]*model.Rule{
		"custom": {Id: "custom", Severity: "error", Given: "$", Then: model.RuleAction{Function: "probe"}},
	}}
	result := ApplyRulesToRuleSet(&RuleSetExecution{
		RuleSet: rs, Spec: []byte("openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}"),
		CustomFunctions: map[string]model.RuleFunction{"probe": probe},
	})
	defer result.Release()
	require.Empty(t, result.Errors)
	require.Equal(t, int32(1), probe.calls.Load())
	supplied, err := rulesets.CreateRuleSetFromData([]byte("rules:\n  disabled: false\n"))
	require.NoError(t, err)
	disabled := rulesets.BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(supplied)
	require.Empty(t, validateRuleFunctions(disabled, functions.MapBuiltinFunctions(), nil))
}

func TestInvalidRuleset_AsyncAPIAndJSONSchema(t *testing.T) {
	for _, tc := range []struct{ format, spec string }{
		{"asyncapi3", "asyncapi: 3.0.0\ninfo: {title: Test, version: '1'}\nchannels: {}"},
		{"json-schema-2020-12", "type: object"},
	} {
		t.Run(tc.format, func(t *testing.T) {
			rs := &rulesets.RuleSet{Rules: map[string]*model.Rule{"invalid": {Id: "invalid", Given: "$.missing", Then: model.RuleAction{Function: "missing"}}}}
			result := ApplyRulesToRuleSet(&RuleSetExecution{RuleSet: rs, Spec: []byte(tc.spec), SpecFormat: tc.format})
			defer result.Release()
			require.Len(t, result.Errors, 1)
			var unknown *UnknownFunctionError
			require.ErrorAs(t, result.Errors[0], &unknown)
			require.Empty(t, result.Results)
			if tc.format == "asyncapi3" {
				direct, handled := ApplyAsyncAPIRulesToRuleSet(&RuleSetExecution{RuleSet: rs, Spec: []byte(tc.spec)}, nil, functions.MapBuiltinFunctions())
				require.True(t, handled)
				defer direct.Release()
				require.Len(t, direct.Errors, 1)
				require.ErrorAs(t, direct.Errors[0], &unknown)
			}
		})
	}
}

func TestIncompleteRuleset_IsRejectedByEngine(t *testing.T) {
	supplied, err := rulesets.CreateRuleSetFromData([]byte("extends: [./missing-ruleset-945.yaml]"))
	require.NoError(t, err)
	rs := rulesets.BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(supplied)
	result := ApplyRulesToRuleSet(&RuleSetExecution{RuleSet: rs})
	defer result.Release()
	require.Len(t, result.Errors, 1)
	require.Contains(t, result.Errors[0].Error(), "missing-ruleset-945.yaml")
	require.Empty(t, result.Results)
}

func TestRuleFunctions_CaseInsensitiveActionMaps(t *testing.T) {
	for _, key := range []string{"function", "Function", "FUNCTION"} {
		for _, given := range []string{"$", "$.missing"} {
			t.Run(key+given, func(t *testing.T) {
				probe := &configurationProbe{}
				rs := &rulesets.RuleSet{Rules: map[string]*model.Rule{
					"probe": {Id: "probe", Given: given, Then: map[string]any{key: "probe"}},
				}}
				result := ApplyRulesToRuleSet(&RuleSetExecution{RuleSet: rs,
					Spec:            []byte("openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}"),
					CustomFunctions: map[string]model.RuleFunction{"probe": probe},
				})
				defer result.Release()
				require.Empty(t, result.Errors)
				if given == "$" {
					require.Equal(t, int32(1), probe.calls.Load())
				} else {
					require.Zero(t, probe.calls.Load())
				}
			})
		}
	}
}
