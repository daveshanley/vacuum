// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package motor

import (
	"testing"

	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
)

func TestIssue1032_FieldDoesNotApplyToScalarTargets(t *testing.T) {
	spec := []byte(`openapi: 3.0.3
info: {title: Repro, version: 1.0.0}
paths:
  /things:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
                  country: {type: string, example: DE}
                  address:
                    type: object
                    properties:
                      country: {type: string}
              example:
                country: DE
                address:
                  country: DE
`)

	rule := func(id, function string) *model.Rule {
		return &model.Rule{
			Id:           id,
			Description:  "country properties must have an example",
			Given:        "$..country",
			Severity:     model.SeverityWarn,
			RuleCategory: model.RuleCategories[model.CategoryValidation],
			Type:         rulesets.Validation,
			Then:         model.RuleAction{Field: "example", Function: function},
		}
	}
	rs := &rulesets.RuleSet{
		Rules: map[string]*model.Rule{
			"country-has-example-truthy":  rule("country-has-example-truthy", "truthy"),
			"country-has-example-defined": rule("country-has-example-defined", "defined"),
		},
	}

	results := ApplyRulesToRuleSet(&RuleSetExecution{
		RuleSet:     rs,
		Spec:        spec,
		SilenceLogs: true,
	})
	require.Empty(t, results.Errors)

	// Only the nested schema property lacks an example; the example payload scalars are values, not objects.
	for _, id := range []string{"country-has-example-truthy", "country-has-example-defined"} {
		ruleResults := filterResultsByRuleId(results.Results, id)
		require.Len(t, ruleResults, 1, id)
		assert.Contains(t, ruleResults[0].Path, ".properties['address'].properties['country']", id)
	}
}
