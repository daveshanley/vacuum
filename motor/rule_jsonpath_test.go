package motor

import (
	"sync"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

func TestRuleJSONPathQuotedProperties(t *testing.T) {
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("{\"$ref\": \"target\", \"application/json\": 1, \"single'quote\": 2, \"back`tick\": 3}"), &node))
	var cache sync.Map
	for _, tc := range []struct{ path, value string }{
		{`$['$ref']`, "target"}, {`$['application/json']`, "1"}, {`$['single\'quote']`, "2"}, {"$['back`tick']", "3"},
		{"$.`$ref", "target"}, {"$.`application/json", "1"}, {"$[`$ref]", "target"},
	} {
		for range 2 {
			nodes, err := findRuleGivenNodes(&node, tc.path, 0, &cache)
			require.NoError(t, err)
			require.Len(t, nodes, 1)
			require.Equal(t, tc.value, nodes[0].Value)
		}
	}
}

func TestRuleJSONPathInvalidBacktickContext(t *testing.T) {
	var cache sync.Map
	for range 2 {
		_, err := compileRuleGivenPath("$`$ref", &cache)
		require.ErrorContains(t, err, "unexpected token when parsing segment")
	}
}

func TestIssue1004RequestBodyRefSelection(t *testing.T) {
	spec := []byte(`openapi: 3.0.3
info:
  title: Example
  version: 1.0.0
paths:
  /pets:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Pet'
      responses:
        '200':
          description: OK
components:
  schemas:
    Pet:
      type: object
`)
	for _, given := range []string{
		`$.paths.*.*.requestBody.content['application/json'].schema['$ref']`,
		"$.paths.*.*.requestBody.content.*.schema.`$ref",
		"$.paths.*.*.requestBody.content.`application/json.schema.`$ref",
	} {
		for _, resolved := range []bool{false, true} {
			rule := &model.Rule{Id: "request-ref", Given: given, Resolved: resolved, Severity: model.SeverityWarn, RuleCategory: model.RuleCategories[model.CategorySchemas], Then: model.RuleAction{Function: "falsy"}}
			result := ApplyRulesToRuleSet(&RuleSetExecution{Spec: spec, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{rule.Id: rule}}, SilenceLogs: true})
			require.Empty(t, result.Errors)
			found := filterResultsByRuleId(result.Results, rule.Id)
			if resolved {
				require.Empty(t, found)
			} else {
				require.Len(t, found, 1)
				require.Equal(t, 12, found[0].StartNode.Line)
			}
			result.ReleaseOwnedResources()
		}
	}
}
