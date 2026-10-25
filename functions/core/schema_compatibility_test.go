package core

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/utils"
	"github.com/pb33f/testify/require"
)

func TestSchemaValidatesSelectedValuesWithoutField(t *testing.T) {
	for _, tc := range []struct {
		name, document string
		count          int
	}{
		{"invalid scalar", "value: 42", 1},
		{"valid scalar", "value: hello", 0},
		{"all selected values", "value: 42\nother: false", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			nodes, err := utils.FindNodes([]byte(tc.document), "$.*")
			require.NoError(t, err)
			opts := map[string]any{"schema": map[string]any{"type": "string"}}
			rule := model.Rule{Given: "$.*", Then: &model.RuleAction{Function: "schema", FunctionOptions: opts}}
			ctx := model.RuleFunctionContext{Rule: &rule, RuleAction: model.CastToRuleAction(rule.Then), Options: opts, Given: rule.Given}
			results := (Schema{}).RunRule(nodes, ctx)
			require.Len(t, results, tc.count)
			for _, result := range results {
				require.Contains(t, result.Message, "want string")
			}
		})
	}
}

func TestSchemaResolvesOptionReferencesAgainstOwnDefinitions(t *testing.T) {
	for _, tc := range []struct {
		value string
		count int
	}{{"text", 1}, {"42", 0}} {
		t.Run(tc.value, func(t *testing.T) {
			document := "value: " + tc.value + "\n$defs: {x: {type: string}}"
			nodes, err := utils.FindNodes([]byte(document), "$.value")
			require.NoError(t, err)
			_, idx := testBuildSpecIndex(document)
			opts := map[string]any{"schema": map[string]any{"$ref": "#/$defs/x", "$defs": map[string]any{"x": map[string]any{"type": "integer"}}}}
			rule := model.Rule{Given: "$.value", Then: &model.RuleAction{Function: "schema", FunctionOptions: opts}}
			ctx := model.RuleFunctionContext{Rule: &rule, RuleAction: model.CastToRuleAction(rule.Then), Options: opts, Given: rule.Given, Index: idx}
			results := (Schema{}).RunRule(nodes, ctx)
			require.Len(t, results, tc.count)
			if tc.count > 0 {
				require.Contains(t, results[0].Message, "want integer")
			}
		})
	}
}

func TestSchemaPreservesInputDocumentReferences(t *testing.T) {
	for _, tc := range []struct {
		value string
		count int
	}{{"text", 1}, {"42", 0}} {
		t.Run(tc.value, func(t *testing.T) {
			document := "value: " + tc.value + "\ncomponents: {schemas: {Number: {type: integer}}}"
			nodes, err := utils.FindNodes([]byte(document), "$.value")
			require.NoError(t, err)
			_, idx := testBuildSpecIndex(document)
			opts := map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/Number"}, "forceValidationOnCurrentNode": true}
			rule := model.Rule{Given: "$.value", Then: &model.RuleAction{Function: "schema", FunctionOptions: opts}}
			ctx := model.RuleFunctionContext{Rule: &rule, RuleAction: model.CastToRuleAction(rule.Then), Options: opts, Given: rule.Given, Index: idx}
			results := (Schema{}).RunRule(nodes, ctx)
			require.Len(t, results, tc.count)
			if tc.count > 0 {
				require.Contains(t, results[0].Message, "want integer")
			}
		})
	}
}

func TestSchemaValidationOptions(t *testing.T) {
	for _, tc := range []struct {
		name, document string
		schema         map[string]any
		options        map[string]any
		count          int
	}{
		{"null rejects string", "value: null", map[string]any{"type": "string"}, nil, 1},
		{"null accepts null", "value: null", map[string]any{"type": "null"}, nil, 0},
		{"all errors by default", "value: abc", map[string]any{"type": "string", "minLength": 5, "pattern": "^z"}, nil, 2},
		{"explicit first error", "value: abc", map[string]any{"type": "string", "minLength": 5, "pattern": "^z"}, map[string]any{"allErrors": false}, 1},
		{"all errors", "value: abc", map[string]any{"type": "string", "minLength": 5, "pattern": "^z"}, map[string]any{"allErrors": true}, 2},
		{"invalid email", "value: not-an-email", map[string]any{"type": "string", "format": "email"}, nil, 1},
		{"valid email", "value: person@example.com", map[string]any{"type": "string", "format": "email"}, nil, 0},
		{"default dialect checks prefix items", "value: [42]", map[string]any{"type": "array", "prefixItems": []any{map[string]any{"type": "string"}}}, nil, 1},
		{"explicit draft seven", "value: [42]", map[string]any{"type": "array", "prefixItems": []any{map[string]any{"type": "string"}}}, map[string]any{"dialect": "draft7"}, 0},
		{"explicit draft 2020", "value: [42]", map[string]any{"type": "array", "prefixItems": []any{map[string]any{"type": "string"}}}, map[string]any{"dialect": "draft2020-12"}, 1},
		{"schema declaration", "value: [42]", map[string]any{"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "prefixItems": []any{map[string]any{"type": "string"}}}, nil, 1},
		{"invalid pattern", "value: abc", map[string]any{"type": "string", "pattern": "["}, nil, 1},
		{"unknown dialect", "value: abc", map[string]any{"type": "string"}, map[string]any{"dialect": "typo"}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			nodes, err := utils.FindNodes([]byte(tc.document), "$.value")
			require.NoError(t, err)
			opts := map[string]any{"schema": tc.schema}
			for k, v := range tc.options {
				opts[k] = v
			}
			rule := model.Rule{Given: "$.value", Then: &model.RuleAction{Function: "schema", FunctionOptions: opts}}
			ctx := model.RuleFunctionContext{Rule: &rule, RuleAction: model.CastToRuleAction(rule.Then), Options: opts, Given: rule.Given}
			require.Len(t, (Schema{}).RunRule(nodes, ctx), tc.count)
		})
	}
}

func TestSchemaMissingField(t *testing.T) {
	for _, tc := range []struct {
		name    string
		options map[string]any
		count   int
	}{
		{"default", nil, 1},
		{"explicit strict", map[string]any{"forceValidation": true}, 1},
		{"optional extension", map[string]any{"forceValidation": false}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			nodes, err := utils.FindNodes([]byte("value: {}"), "$.value")
			require.NoError(t, err)
			opts := map[string]any{"schema": map[string]any{"type": "string"}}
			for k, v := range tc.options {
				opts[k] = v
			}
			rule := model.Rule{Given: "$.value", Then: &model.RuleAction{Function: "schema", Field: "absent", FunctionOptions: opts}}
			ctx := model.RuleFunctionContext{Rule: &rule, RuleAction: model.CastToRuleAction(rule.Then), Options: opts, Given: rule.Given}
			require.Len(t, (Schema{}).RunRule(nodes, ctx), tc.count)
		})
	}
}

func TestSchemaRejectsInvalidOptions(t *testing.T) {
	for _, options := range []map[string]any{
		{"dialect": "unknown"}, {"dialect": 42}, {"dialect": ""}, {"allErrors": "true"},
	} {
		require.Error(t, (Schema{}).ValidateOptions(options))
	}
	require.NoError(t, (Schema{}).ValidateOptions(map[string]any{"dialect": "draft7", "allErrors": true}))
}

func TestSchemaReportsInvalidRuntimeOptions(t *testing.T) {
	nodes, err := utils.FindNodes([]byte("value: hello"), "$.value")
	require.NoError(t, err)
	var reported error
	results := (Schema{}).RunRule(nodes, model.RuleFunctionContext{
		Options:     map[string]any{"allErrors": "true"},
		ReportError: func(err error) { reported = err },
	})
	require.Error(t, reported)
	require.Len(t, results, 1)
}
