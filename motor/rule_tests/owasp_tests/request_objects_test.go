package tests

import (
	"fmt"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestRuleSet_OWASPRequestObjects(t *testing.T) {
	for _, version := range []string{"3.0.3", "3.1.0", "3.2.0"} {
		for _, tc := range []struct {
			name, schema    string
			open, unbounded int
		}{
			{"omitted", "{type: object, properties: {name: {type: string}}}", 1, 0},
			{"untyped", "{properties: {name: {type: string}}}", 1, 0},
			{"untyped open", "{additionalProperties: true}", 1, 1},
			{"finite explicit open", "{type: object, additionalProperties: true, enum: [{name: fixed}]}", 0, 0},
			{"explicit open", "{type: object, additionalProperties: true}", 1, 1},
			{"count does not close object", "{type: object, additionalProperties: true, maxProperties: 10}", 1, 0},
			{"closed", "{type: object, additionalProperties: false}", 0, 0},
			{"bounded map", "{type: object, additionalProperties: {type: string}, maxProperties: 10}", 0, 0},
			{"unbounded map", "{type: object, additionalProperties: {type: string}}", 0, 1},
			{"untyped map", "{additionalProperties: {type: string}}", 0, 1},
			{"finite object", "{type: object, enum: [{name: fixed}]}", 0, 0},
			{"referenced composition", "{allOf: [{$ref: '#/components/schemas/Base'}], unevaluatedProperties: false}", 0, 0},
			{"composed and direct", "{type: object, additionalProperties: false, properties: {closed: {allOf: [{$ref: '#/components/schemas/Base'}], unevaluatedProperties: false}, open: {$ref: '#/components/schemas/Base'}}}", 1, 0},
			{"inline composition", "{allOf: [{type: object, properties: {name: {type: string}}}, {type: object, additionalProperties: false}]}", 0, 0},
		} {
			t.Run(version+"/"+tc.name, func(t *testing.T) {
				spec := fmt.Sprintf(`openapi: %s
info: {title: Test, version: '1'}
paths:
  /items:
    post:
      requestBody:
        content:
          application/json:
            schema: {$ref: '#/components/schemas/Input'}
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema: {type: object, additionalProperties: true}
components:
  schemas:
    Input: %s
    Unused: {type: object, additionalProperties: true}
    Base: {type: object, properties: {name: {type: string}}}
`, version, tc.schema)
				a, b := rulesets.GetOWASPNoAdditionalPropertiesRule(), rulesets.GetOWASPConstrainedAdditionalPropertiesRule()
				result := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{Spec: []byte(spec), RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{a.Id: a, b.Id: b}}})
				require.Empty(t, result.Errors)
				counts := map[string]int{}
				for _, finding := range result.Results {
					counts[finding.Rule.Id]++
					assert.Positive(t, finding.StartNode.Line)
					assert.NotEmpty(t, finding.Path)
				}
				assert.Equal(t, tc.open, counts[a.Id])
				assert.Equal(t, tc.unbounded, counts[b.Id])
			})
		}
	}
}

func TestRuleSet_OWASPRequestObjectUnevaluatedProperties(t *testing.T) {
	for _, version := range []string{"3.0.3", "3.1.0", "3.2.0"} {
		for _, schema := range []string{"{type: object, unevaluatedProperties: false}", "{type: object, unevaluatedProperties: {type: string}, maxProperties: 10}"} {
			t.Run(version+schema, func(t *testing.T) {
				spec := fmt.Sprintf("openapi: %s\ninfo: {title: Test, version: '1'}\npaths:\n  /items:\n    post:\n      requestBody:\n        content:\n          application/json:\n            schema: %s\n", version, schema)
				a, b := rulesets.GetOWASPNoAdditionalPropertiesRule(), rulesets.GetOWASPConstrainedAdditionalPropertiesRule()
				result := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{Spec: []byte(spec), RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{a.Id: a, b.Id: b}}})
				require.Empty(t, result.Errors)
				want := 0
				if version == "3.0.3" {
					want = 1
				}
				assert.Len(t, result.Results, want)
			})
		}
	}
}
