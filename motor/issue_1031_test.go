// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package motor

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

// casing with then.field checks the field's value and reports its location, as Spectral does (issue #1031)
func TestIssue1031_CasingChecksFieldValue(t *testing.T) {
	spec := []byte(`openapi: 3.0.3
info: {title: Repro, version: 1.0.0}
paths:
  /things:
    get:
      operationId: Get things
      x-internal_note: a
      responses:
        '200': {description: OK}
`)

	rs := &rulesets.RuleSet{
		Rules: map[string]*model.Rule{
			"operation-id-camel-case": {
				Id:           "operation-id-camel-case",
				Description:  "operationId must be camelCase",
				Given:        "$.paths.*.get",
				Severity:     model.SeverityWarn,
				RuleCategory: model.RuleCategories[model.CategoryValidation],
				Type:         rulesets.Validation,
				Then: model.RuleAction{
					Field:           "operationId",
					Function:        "casing",
					FunctionOptions: map[string]any{"type": "camel"},
				},
			},
		},
	}

	results := ApplyRulesToRuleSet(&RuleSetExecution{
		RuleSet:     rs,
		Spec:        spec,
		SilenceLogs: true,
	})
	require.Empty(t, results.Errors)

	ruleResults := filterResultsByRuleId(results.Results, "operation-id-camel-case")
	require.Len(t, ruleResults, 1, "only operationId should be checked, not the operation keys")
	assert.Equal(t, "$.paths['/things'].get.operationId", ruleResults[0].Path)
	assert.Equal(t, 6, ruleResults[0].StartNode.Line)
	assert.Equal(t, 20, ruleResults[0].StartNode.Column)
	assert.Contains(t, ruleResults[0].Message, "`Get things` is not `camel` case")
}
