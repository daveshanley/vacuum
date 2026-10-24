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
