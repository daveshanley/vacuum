// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

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
			nodes, err := findRuleGivenNodes(&node, tc.path, 0, &cache, model.NativeContract)
			require.NoError(t, err)
			require.Len(t, nodes, 1)
			require.Equal(t, tc.value, nodes[0].Value)
		}
	}
}

func TestRuleJSONPathInvalidBacktickContext(t *testing.T) {
	var cache sync.Map
	for range 2 {
		_, err := compileRuleGivenPath("$`$ref", &cache, model.NativeContract)
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

func TestOpenLintSlashMemberSelection(t *testing.T) {
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(`paths:
  /pets/{id}:
    /v2: found
  './literal': quoted
`), &node))
	for _, tc := range []struct{ path, want string }{
		{"$.paths./pets/{id}./v2", "found"},
		{"$../pets/{id}./v2", "found"},
		{"$.paths['./literal']", "quoted"},
	} {
		nodes, err := findRuleGivenNodes(&node, tc.path, 0, nil, model.OpenLintV1Draft)
		require.NoError(t, err)
		require.Len(t, nodes, 1)
		require.Equal(t, tc.want, nodes[0].Value)
	}
	_, err := compileRuleGivenPath("$.paths./pets", nil, model.NativeContract)
	require.Error(t, err)
	for _, selector := range []string{`$[?(@.name == './literal')]`, `$[?(@.name.match(/x./))]`, `$[?(@./invalid)]`} {
		require.Equal(t, selector, openLintLiteralMembers(selector))
	}
}

func TestOpenLintHyphenatedMemberSelection(t *testing.T) {
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(`number-then-numeric-string: [1.5, "1"]
not-numeric-words: [[1, nope], [2, NaN]]
parent-name:
  child-2: found
  accented-é: unicode
items:
  - child-name: inside
    name: a-b
`), &node))
	for _, tc := range []struct {
		path, want string
		count      int
	}{
		{"$.number-then-numeric-string[0]", "1.5", 1},
		{"$.not-numeric-words[*][1]", "nope", 2},
		{"$.parent-name.child-2", "found", 1},
		{"$.parent-name.accented-é", "unicode", 1},
		{"$..child-name", "inside", 1},
		{"$.items[*].child-name", "inside", 1},
		{`$.items[?(@.name == 'a-b')].child-name`, "inside", 1},
		{"$['parent-name']['child-2']", "found", 1},
	} {
		t.Run(tc.path, func(t *testing.T) {
			nodes, err := findRuleGivenNodes(&node, tc.path, 0, nil, model.OpenLintV1Draft)
			require.NoError(t, err)
			require.Len(t, nodes, tc.count)
			require.Equal(t, tc.want, nodes[0].Value)
		})
	}
	for _, selector := range []string{
		`$['quoted-name']`, `$[?(@.name == 'a-b')]`, `$[?(@.name.match(/a-b/))]`,
		`$[?(@.score - 1 > 0)]`, "$[?(@['child-name'])]", "$.`already-escaped",
	} {
		require.Equal(t, selector, openLintLiteralMembers(selector))
	}
}

func TestOpenLintSelectorCacheKeepsNativeContract(t *testing.T) {
	for _, contracts := range [][]model.Contract{
		{model.NativeContract, model.OpenLintV1Draft},
		{model.OpenLintV1Draft, model.NativeContract},
	} {
		var cache sync.Map
		for range 2 {
			for _, contract := range contracts {
				_, err := compileRuleGivenPath("$.hyphenated-name", &cache, contract)
				if contract == model.OpenLintV1Draft {
					require.NoError(t, err)
				} else {
					require.Error(t, err)
				}
			}
		}
	}
}

func TestOpenLintHyphenatedGivenAndField(t *testing.T) {
	rs, err := rulesets.CreateRuleSetFromDataWithContract([]byte(`rules:
  hyphenated:
    given: $.parent-name
    then:
      field: $.child-name
      function: falsy
`), model.OpenLintV1Draft)
	require.NoError(t, err)
	result := ApplyRulesToRuleSetWithOptions(&RuleSetExecution{
		RuleSet: rs, Spec: []byte("parent-name:\n  child-name: present\n"), SilenceLogs: true,
	}, &ExecutionOptions{Contract: model.OpenLintV1Draft})
	defer result.ReleaseOwnedResources()
	require.Empty(t, result.Errors)
	require.Len(t, result.Results, 1)
	require.Equal(t, "$['parent-name']['child-name']", result.Results[0].Path)
	require.Equal(t, 2, result.Results[0].StartNode.Line)
	require.Equal(t, 15, result.Results[0].StartNode.Column)
}
