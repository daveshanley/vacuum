// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package motor

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
)

func TestOASSchema_Issue1034ResultsReachDoctor(t *testing.T) {
	const spec = `openapi: 3.0.3
info: {title: Repro, version: 1.0.0}
paths: {}
components:
  schemas:
    Order:
      type: object
      properties:
        token:
          oneOf:
            - type: string
            - type: 'null'
`
	for _, defaults := range []bool{false, true} {
		name := "isolated rule"
		ruleSet := &rulesets.RuleSet{Rules: map[string]*model.Rule{"oas3-schema": rulesets.GetOAS3SchemaRule()}}
		if defaults {
			name = "default rules"
			ruleSet = rulesets.BuildDefaultRuleSets().GenerateOpenAPIRecommendedRuleSet()
		}
		t.Run(name, func(t *testing.T) {
			result := ApplyRulesToRuleSet(&RuleSetExecution{Spec: []byte(spec), RuleSet: ruleSet})
			defer result.Release()
			require.Empty(t, result.Errors)
			var findings []model.RuleFunctionResult
			for _, finding := range result.Results {
				if finding.RuleId == "oas3-schema" {
					findings = append(findings, finding)
				}
			}
			require.Len(t, findings, 1)
			finding := findings[0]
			require.Equal(t, "$.components.schemas.Order.properties.token.oneOf[1].type", finding.Path)
			require.Equal(t, []string{finding.Path}, finding.Paths)
			require.Equal(t, 12, finding.StartNode.Line)
			require.Equal(t, 21, finding.StartNode.Column)
			stored := result.RuleSetExecution.DrDocument.V3Document.GetRuleFunctionResults()
			count := 0
			for _, diagnostic := range stored {
				if diagnostic.RuleId == "oas3-schema" {
					count++
					require.Equal(t, finding.Path, diagnostic.Path)
					require.Equal(t, finding.Message, diagnostic.Message)
				}
			}
			require.Equal(t, 1, count, "Doctor must receive the same single diagnostic")
		})
	}
}

func TestOASSchema_AliasFindingsReachDoctor(t *testing.T) {
	for _, schema := range []string{"*bad", "\n      <<: *bad"} {
		t.Run(schema, func(t *testing.T) {
			spec := "openapi: 3.0.3\ninfo: {title: Test, version: '1'}\npaths: {}\nx-bad: &bad\n  type: 'null'\ncomponents:\n  schemas:\n    Order: " + schema + "\n"
			result := ApplyRulesToRuleSet(&RuleSetExecution{Spec: []byte(spec), RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"oas3-schema": rulesets.GetOAS3SchemaRule()}}})
			defer result.Release()
			require.Empty(t, result.Errors)
			require.Len(t, result.Results, 1)
			require.Equal(t, 5, result.Results[0].StartNode.Line)
			stored := result.RuleSetExecution.DrDocument.V3Document.GetRuleFunctionResults()
			require.Len(t, stored, 1)
			require.Equal(t, result.Results[0].Path, stored[0].Path)
			require.Equal(t, result.Results[0].Message, stored[0].Message)
		})
	}
}
