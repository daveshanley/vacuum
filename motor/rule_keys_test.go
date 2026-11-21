// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package motor

import (
	"fmt"
	"testing"

	"github.com/daveshanley/vacuum/model"
	jsplugin "github.com/daveshanley/vacuum/plugin/javascript"
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

func TestNativeKeyFieldNamesAreStrings(t *testing.T) {
	const spec = `openapi: 3.1.0
info: {title: Test, version: "1"}
paths: {}
x-list: [a, b]
x-names: {"0": a, false: b, "": c, ok: d}
`
	const rules = `rules:
  truthy-list:
    given: $['x-list']
    then: {field: '@key', function: truthy}
  truthy-names:
    given: $['x-names']
    then: {field: '@key', function: truthy}
  falsy-names:
    given: $['x-names']
    then: {field: '@key', function: falsy}
`
	raw, err := rulesets.CreateRuleSetFromData([]byte(rules))
	require.NoError(t, err)
	result := ApplyRulesToRuleSet(&RuleSetExecution{RuleSet: rulesets.BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(raw), Spec: []byte(spec), SilenceLogs: true})
	defer result.ReleaseOwnedResources()
	require.Empty(t, result.Errors)
	var got []string
	for _, r := range result.Results {
		got = append(got, r.RuleId+" "+r.Path)
	}
	require.ElementsMatch(t, []string{
		"truthy-names $['x-names']['']",
		"falsy-names $['x-names']['0']",
		"falsy-names $['x-names'].false",
		"falsy-names $['x-names'].ok",
	}, got)
}

func TestNativeKeyFieldMergeKeys(t *testing.T) {
	const spec = `openapi: 3.1.0
info: {title: Test, version: "1"}
paths: {}
x-first: &first {shared: 1, first: 1}
x-second: &second {shared: 2, second: 2, <<: *first}
x-merged:
  <<: [*second, *first]
  own: 3
  shared: 4
`
	const rules = `rules:
  keys:
    given: $['x-merged']
    then: {field: '@key', function: pattern, functionOptions: {match: '^$'}}
`
	raw, err := rulesets.CreateRuleSetFromData([]byte(rules))
	require.NoError(t, err)
	result := ApplyRulesToRuleSet(&RuleSetExecution{RuleSet: rulesets.BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(raw), Spec: []byte(spec), SilenceLogs: true})
	defer result.ReleaseOwnedResources()
	require.Empty(t, result.Errors)
	type finding struct {
		path string
		line int
	}
	var got []finding
	for _, r := range result.Results {
		got = append(got, finding{r.Path, r.StartNode.Line})
	}
	// Local keys win; inherited keys keep their authored lines but belong to x-merged.
	require.ElementsMatch(t, []finding{
		{"$['x-merged'].second", 5},
		{"$['x-merged'].first", 4},
		{"$['x-merged'].own", 8},
		{"$['x-merged'].shared", 9},
	}, got)
}

func TestNativeKeyFieldJavaScriptErrorPath(t *testing.T) {
	const fixture = `data: {good: 1, bad: 2}
`
	fn := jsplugin.NewJSRuleFunction("failOnBad", `function runRule(input) {
  if (input === "bad") { throw new Error("cannot check " + input); }
  return [];
}`)
	require.NoError(t, fn.CheckScript())
	rule := &model.Rule{Id: "r", Given: "$.data", Severity: "warn", Then: model.RuleAction{Field: "@key", Function: "failOnBad"}}
	result := ApplyRulesToRuleSet(&RuleSetExecution{Spec: []byte(fixture), SkipDocumentCheck: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"r": rule}}, CustomFunctions: map[string]model.RuleFunction{"failOnBad": fn}})
	defer result.ReleaseOwnedResources()
	require.Len(t, result.Results, 1)
	require.Contains(t, result.Results[0].Message, "cannot check bad")
	require.Equal(t, "$.data.bad", result.Results[0].Path)
	require.Equal(t, "bad", result.Results[0].StartNode.Value)
}

func TestNativeKeyFieldDefinedMessageAndArrayCasing(t *testing.T) {
	const spec = `openapi: 3.1.0
info: {title: Test, version: "1"}
paths: {}
x-empty: {}
x-tags: [alpha, beta]
`
	const rules = `rules:
  empty:
    description: keys required
    given: $['x-empty']
    then: {field: '@key', function: defined}
  # Like Spectral, @key on an array selects its indices, not its elements.
  tags:
    description: tags
    given: $['x-tags']
    then: {field: '@key', function: casing, functionOptions: {type: camel}}
`
	raw, err := rulesets.CreateRuleSetFromData([]byte(rules))
	require.NoError(t, err)
	result := ApplyRulesToRuleSet(&RuleSetExecution{RuleSet: rulesets.BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(raw), Spec: []byte(spec), SilenceLogs: true})
	defer result.ReleaseOwnedResources()
	require.Empty(t, result.Errors)
	var got []string
	for _, r := range result.Results {
		got = append(got, r.RuleId+" "+r.Path+" "+r.Message)
	}
	require.ElementsMatch(t, []string{
		"empty $['x-empty'] keys required: value must be defined",
		"tags $['x-tags'][0] tags: `0` is not `camel` case",
		"tags $['x-tags'][1] tags: `1` is not `camel` case",
	}, got)
}

func TestNativeKeyFieldEmptyMerge(t *testing.T) {
	const spec = `openapi: 3.1.0
info: {title: Test, version: "1"}
paths: {}
x-base: &empty {}
x-full: &full {a: 1}
x-empty:
  <<: *empty
x-filled:
  <<: *full
`
	const rules = `rules:
  defined:
    description: keys required
    given:
      - $['x-empty']
      - $['x-filled']
    then: {field: '@key', function: defined}
  truthy:
    description: keys required
    given:
      - $['x-empty']
      - $['x-filled']
    then: {field: '@key', function: truthy}
`
	raw, err := rulesets.CreateRuleSetFromData([]byte(rules))
	require.NoError(t, err)
	result := ApplyRulesToRuleSet(&RuleSetExecution{RuleSet: rulesets.BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(raw), Spec: []byte(spec), SilenceLogs: true})
	defer result.ReleaseOwnedResources()
	require.Empty(t, result.Errors)
	var got []string
	for _, r := range result.Results {
		got = append(got, fmt.Sprintf("%s %s %d", r.RuleId, r.Path, r.StartNode.Line))
	}
	// Merging an empty anchor inherits no keys, so x-empty has none; x-filled inherits 'a'.
	require.ElementsMatch(t, []string{
		"defined $['x-empty'] 7",
		"truthy $['x-empty'] 7",
	}, got)
}
