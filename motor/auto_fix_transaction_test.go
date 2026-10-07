// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package motor

import (
	"fmt"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

type repeatedFindingFunction struct{}

func (repeatedFindingFunction) GetCategory() string { return model.CategoryValidation }
func (repeatedFindingFunction) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "repeatedFinding"}
}
func (repeatedFindingFunction) RunRule(nodes []*yaml.Node, _ model.RuleFunctionContext) []model.RuleFunctionResult {
	return []model.RuleFunctionResult{
		{StartNode: nodes[0], Path: "$.info.description", Message: "first"},
		{StartNode: nodes[0], Path: "$.info.description", Message: "second"},
	}
}

func TestAutoFixTransactionPreservesDocumentAndRepeatedTargetEdits(t *testing.T) {
	for _, failure := range []string{"", "error", "panic"} {
		t.Run("failure="+failure, func(t *testing.T) {
			calls := 0
			var recovered any
			rule := &model.Rule{
				Id:              "test",
				Given:           "$.info.description",
				Severity:        model.SeverityWarn,
				AutoFixFunction: "fix",
				Then:            &model.RuleAction{Function: "repeatedFinding"},
			}
			execution := &RuleSetExecution{
				Spec: []byte(`openapi: 3.0.0
info:
  title: Test
  version: 1.0.0
  description: original
paths: {}
`),
				RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"test": rule}},
				CustomFunctions: map[string]model.RuleFunction{
					"repeatedFinding": repeatedFindingFunction{},
				},
				ApplyAutoFixes: true,
				SilenceLogs:    true,
				PanicFunction:  func(p any) { recovered = p },
				AutoFixFunctions: map[string]model.AutoFixFunction{"fix": func(node, document *yaml.Node, _ *model.RuleFunctionContext) (*yaml.Node, error) {
					calls++
					node.Value += "X"
					root := document
					if root.Kind == yaml.DocumentNode {
						root = root.Content[0]
					}
					if calls == 1 {
						root.Content = append(root.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "x-fixed"}, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "yes"})
					}
					if calls == 2 {
						switch failure {
						case "error":
							return nil, fmt.Errorf("cannot fix")
						case "panic":
							panic("cannot fix")
						}
					}
					return node, nil
				}},
			}
			result := ApplyRulesToRuleSet(execution)
			defer result.ReleaseOwnedResources()
			require.Equal(t, 2, calls)
			if failure == "" {
				require.Empty(t, result.Errors)
				require.Len(t, result.FixedResults, 2)
				require.Contains(t, string(result.ModifiedSpec), "originalXX")
				require.Contains(t, string(result.ModifiedSpec), "x-fixed: yes")
			} else {
				require.Empty(t, result.FixedResults)
				rendered, err := yaml.Marshal(execution.CanonicalDocument)
				require.NoError(t, err)
				require.Contains(t, string(rendered), "description: original")
				require.NotContains(t, string(rendered), "originalX")
				require.NotContains(t, string(rendered), "x-fixed")
				if failure == "error" {
					require.Len(t, result.Results, 2)
				} else {
					require.Equal(t, "cannot fix", recovered)
					// A panicked rule never completes, so neither its staged fixes
					// nor its incomplete findings are published.
					require.Empty(t, result.Results)
				}
			}
		})
	}
}
