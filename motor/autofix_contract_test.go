package motor

import (
	"encoding/json"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

func TestAutoFixTransactionOutcome(t *testing.T) {
	for _, mode := range []string{"noop", "replacement", "sibling"} {
		t.Run(mode, func(t *testing.T) {
			rule := &model.Rule{Id: "repair", Given: "$.info.description", Severity: "warn", AutoFixFunction: "fix", Then: &model.RuleAction{Function: "truthy"}}
			e := &RuleSetExecution{Spec: []byte(`{"openapi":"3.0.0","info":{"title":"Test","version":"1","description":""},"paths":{},"x-big":9007199254740993.0,"x-fraction":0.123456789012345678901,"x-exponent":1.234567890123456789e40}`), SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"repair": rule}}, ApplyAutoFixes: true, AutoFixFunctions: map[string]model.AutoFixFunction{"fix": func(node, doc *yaml.Node, _ *model.RuleFunctionContext) (*yaml.Node, error) {
				if mode == "replacement" {
					node.Value = "staged"
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "replacement"}, nil
				}
				if mode == "sibling" {
					doc.Content[0].Content = append(doc.Content[0].Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "x-note"}, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "added"})
				}
				return node, nil
			}}}
			r := ApplyRulesToRuleSet(e)
			defer r.ReleaseOwnedResources()
			if mode == "sibling" {
				require.Len(t, r.FixedResults, 1)
				require.True(t, json.Valid(r.ModifiedSpec))
				require.Contains(t, string(r.ModifiedSpec), "9007199254740993.0")
				require.Contains(t, string(r.ModifiedSpec), "0.123456789012345678901")
				require.Contains(t, string(r.ModifiedSpec), "1.234567890123456789e40")
				require.Contains(t, string(r.ModifiedSpec), `"x-note":"added"`)
			} else {
				if mode == "replacement" {
					require.Len(t, r.Errors, 1)
					var fixError *RuleExecutionError
					require.ErrorAs(t, r.Errors[0], &fixError)
					require.Equal(t, RuleErrorAutoFix, fixError.Kind)
				}
				require.Empty(t, r.FixedResults)
				require.Nil(t, r.ModifiedSpec)
				require.Len(t, r.Results, 1)
				raw, err := yaml.Marshal(e.CanonicalDocument)
				require.NoError(t, err)
				require.NotContains(t, string(raw), "staged")
			}
		})
	}
}
