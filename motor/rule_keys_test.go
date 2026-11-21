// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package motor

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
)

func TestNativeKeyFieldMediaTypes(t *testing.T) {
	const spec = `openapi: 3.1.0
info: {title: Example, version: "1"}
paths:
  /things:
    get:
      responses:
        "200":
          description: OK
          content:
            text/plain: {schema: {type: string}}
            application/json: {schema: {type: object}}
`
	const rules = `rules:
  pattern:
    given: $.paths[*][*].responses[*].content
    then: {field: '@key', function: pattern, functionOptions: {match: '^application/json$'}}
  enumeration:
    given: $.paths[*][*].responses[*].content
    then: {field: '@key', function: enumeration, functionOptions: {values: [text/plain]}}
  casing:
    given: $.paths[*][*].responses[*].content
    then: {field: '@key', function: casing, functionOptions: {type: macro}}
`
	raw, err := rulesets.CreateRuleSetFromData([]byte(rules))
	require.NoError(t, err)
	rs := rulesets.BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(raw)
	result := ApplyRulesToRuleSet(&RuleSetExecution{RuleSet: rs, Spec: []byte(spec), SilenceLogs: true})
	defer result.ReleaseOwnedResources()
	require.Empty(t, result.Errors)
	type finding struct {
		rule, path, key string
		line            int
	}
	var got []finding
	for _, r := range result.Results {
		got = append(got, finding{r.RuleId, r.Path, r.StartNode.Value, r.StartNode.Line})
	}
	const path = "$.paths['/things'].get.responses['200'].content"
	require.ElementsMatch(t, []finding{
		{"pattern", path + "['text/plain']", "text/plain", 10},
		{"enumeration", path + "['application/json']", "application/json", 11},
		{"casing", path + "['text/plain']", "text/plain", 10},
		{"casing", path + "['application/json']", "application/json", 11},
	}, got)
}

func TestNativeKeyFieldTargets(t *testing.T) {
	const fixture = `data:
  - {a/b: 12, 42: xyz}
  - [hello, world]
  - scalar
`
	for _, batch := range []bool{false, true} {
		rule := &model.Rule{Id: "r", Given: "$.data[*]", Severity: "warn", Message: "{{value}} at {{path}}", Then: model.RuleAction{Field: "@key", Function: "capture", FunctionOptions: map[string]any{"batch": batch}}}
		result := ApplyRulesToRuleSet(&RuleSetExecution{Spec: []byte(fixture), SkipDocumentCheck: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"r": rule}}, CustomFunctions: map[string]model.RuleFunction{"capture": messageTestFunction{}}})
		defer result.ReleaseOwnedResources()
		require.Empty(t, result.Errors)
		var got []string
		for _, r := range result.Results {
			got = append(got, r.Message)
			require.Equal(t, "!!str", r.StartNode.Tag)
		}
		require.ElementsMatch(t, []string{"a/b at /data/0/a~1b", "42 at /data/0/42", "0 at /data/1/0", "1 at /data/1/1", "scalar at /data/2"}, got)
	}
}

func TestNativeKeyFieldPresenceAndLocations(t *testing.T) {
	const spec = `openapi: 3.1.0
info: {title: Test, version: "1"}
components:
  responses:
    R:
      description: Response
      content:
        text/plain: {}
        text/html: {}
paths:
  /a:
    get:
      responses: {'200': {$ref: '#/components/responses/R'}}
  /b:
    get:
      responses: {'200': {$ref: '#/components/responses/R'}}
x-empty: {}
x-media: &media {good: true, bad: true}
x-copy: *media
`
	const rules = `rules:
  shared:
    resolved: true
    given: $.paths[*][*].responses[*].content
    then: {field: '@key', function: pattern, functionOptions: {match: '^application/json$'}}
  empty:
    given: $['x-empty']
    then: {field: '@key', function: defined}
  present:
    given: $['x-copy']
    then: {field: '@key', function: undefined}
`
	raw, err := rulesets.CreateRuleSetFromData([]byte(rules))
	require.NoError(t, err)
	result := ApplyRulesToRuleSet(&RuleSetExecution{RuleSet: rulesets.BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(raw), Spec: []byte(spec), SilenceLogs: true})
	defer result.ReleaseOwnedResources()
	require.Empty(t, result.Errors)
	var paths []string
	for _, r := range result.Results {
		paths = append(paths, r.Path)
	}
	require.ElementsMatch(t, []string{"$.components.responses.R.content['text/plain']", "$.components.responses.R.content['text/html']", "$['x-empty']", "$['x-copy'].good", "$['x-copy'].bad"}, paths)
}
