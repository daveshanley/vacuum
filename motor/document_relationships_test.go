// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package motor

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
)

func TestOpenAPINativeDocumentRelationships(t *testing.T) {
	for _, tc := range []struct {
		name, version, body string
		rule                *model.Rule
		violations          []string
	}{
		{"null operation", "3.1.1", `paths:
  /items: {get: null}
`, rulesets.GetOAS3SchemaRule(), []string{"$.paths['/items'].get"}},
		{"missing link target", "3.1.1", `components:
  links:
    next: {operationId: missing, server: {url: /}}
`, rulesets.GetOAS3DocumentRelationshipsRule(), []string{"$.components.links.next.operationId"}},
		{"link in a referenced response", "3.1.1", `paths:
  /shop:
    get: {responses: {'200': {$ref: '#/components/responses/Record'}}}
components:
  responses:
    Record: {description: ok, links: {orphan: {operationId: nowhere}}}
`, rulesets.GetOAS3DocumentRelationshipsRule(), []string{"$.components.responses.Record.links.orphan.operationId"}},
		{"existing target and link server", "3.1.1", `paths:
  /items:
    get: {operationId: list, responses: {'200': {description: ok}}}
components:
  links:
    next: {operationId: list, server: {url: /}}
`, rulesets.GetOAS3SchemaRule(), nil},
		{"additional operation target", "3.2.0", `paths:
  /items:
    additionalOperations:
      LOCK: {operationId: lock, responses: {'200': {description: ok}}}
components:
  links:
    next: {operationId: lock}
  examples:
    literal: {value: {operationId: missing}}
`, rulesets.GetOAS3DocumentRelationshipsRule(), nil},
		{"tag graph", "3.2.0", `tags: [{name: a, parent: b}, {name: b, parent: a}, {name: c, parent: missing}, {name: a}]
`, rulesets.GetOAS3DocumentRelationshipsRule(), []string{"$.tags[0].parent", "$.tags[2].parent", "$.tags[3].name"}},
		{"3.0 advisory default", "3.0.4", `servers: [{url: 'https://{s}', variables: {s: {default: c, enum: [a,b]}}}]
`, rulesets.GetOAS3DocumentRelationshipsRule(), nil},
		{"enum membership", "3.1.1", `servers: [{url: 'https://{s}', variables: {s: {default: c, enum: [a,b]}}}]
`, rulesets.GetOAS3DocumentRelationshipsRule(), []string{"$.servers[0].variables.s.default"}},
		{"empty string is an enum value", "3.1.1", `servers: [{url: 'https://{s}', variables: {s: {default: '', enum: ['']}}}]
`, rulesets.GetOAS3DocumentRelationshipsRule(), nil},
		{"xml presence", "3.2.0", `components:
  schemas: {X: {type: array, xml: {nodeType: element, wrapped: false}}}
`, rulesets.GetOAS3SchemaRule(), []string{"$.components.schemas['X'].xml.wrapped"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := fmt.Sprintf("openapi: %s\ninfo: {title: Test, version: '1'}\n", tc.version) + tc.body
			r := ApplyRulesToRuleSet(&RuleSetExecution{Spec: []byte(data), RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{tc.rule.Id: tc.rule}}})
			defer r.Release()
			var paths []string
			for _, finding := range r.Results {
				paths = append(paths, finding.Path)
				require.NotNil(t, finding.StartNode)
			}
			require.ElementsMatch(t, tc.violations, paths)
		})
	}
}

func TestOpenAPILocalOperationRefs(t *testing.T) {
	const spec = `openapi: 3.2.0
info: {title: Links, version: '1'}
paths:
  /fromArray: {$ref: '#/x-path-items/0'}
  /merged: {<<: {get: {responses: {'204': {description: ok}}}}}
  /alias: {$ref: '#/components/pathItems/Shared'}
  /a~b/{id}:
    get:
      responses:
        '200':
          description: ok
          links:
            escaped: {operationRef: '#/paths/~1a~0b~1%7Bid%7D/get'}
            missing: {operationRef: '#/paths/~1missing/get'}
            info: {operationRef: '#/info'}
            empty: {operationRef: ''}
            root: {operationRef: '#'}
            external: {operationRef: 'other.yaml#/paths/~1else/get'}
            web: {operationRef: 'https://example.test/doc#/paths/~1else/get'}
webhooks:
  lock:
    additionalOperations:
      LOCK: {}
components:
  links:
    fromArray: {operationRef: '#/x-path-items/0/get'}
    merged: {operationRef: '#/paths/~1merged/get'}
    shared: {operationRef: '#/components/pathItems/Shared/get'}
    absentAtUseSite: {operationRef: '#/paths/~1alias/get'}
    custom: {operationRef: '#/webhooks/lock/additionalOperations/LOCK'}
    example: {operationRef: '#/components/examples/demo/value/get'}
    badEscape: {operationRef: '#/paths/~2/get'}
  pathItems:
    Shared: {get: {}}
  examples:
    demo: {value: {get: {}, operationRef: '#/missing'}}
x-path-items: [{get: {responses: {'204': {description: ok}}}}]
`
	rule := rulesets.GetOAS3DocumentRelationshipsRule()
	r := ApplyRulesToRuleSet(&RuleSetExecution{Spec: []byte(spec), RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{rule.Id: rule}}})
	defer r.Release()
	var bad []string
	for _, f := range r.Results {
		bad = append(bad, f.StartNode.Value)
		require.Contains(t, f.Path, ".operationRef")
		require.Equal(t, model.SeverityError, f.RuleSeverity)
	}
	require.ElementsMatch(t, []string{"", "#", "#/paths/~1missing/get", "#/paths/~1alias/get", "#/info", "#/components/examples/demo/value/get", "#/paths/~2/get"}, bad)
}

func TestOpenAPI32EncodingExclusions(t *testing.T) {
	for _, tc := range []struct {
		content string
		valid   bool
	}{
		{"encoding: {}, prefixEncoding: []", false},
		{"encoding: {}, itemEncoding: {}", false},
		{"prefixEncoding: [{encoding: {}, prefixEncoding: []}]", false},
		{"prefixEncoding: [], itemEncoding: {}", true},
	} {
		t.Run(tc.content, func(t *testing.T) {
			spec := "openapi: 3.2.0\ninfo: {title: Test, version: '1'}\npaths:\n  /entry:\n    post:\n      requestBody:\n        content:\n          multipart/mixed: {" + tc.content + "}\n      responses: {'200': {description: ok}}\n"
			rule := rulesets.GetOAS3SchemaRule()
			r := ApplyRulesToRuleSet(&RuleSetExecution{Spec: []byte(spec), RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{rule.Id: rule}}})
			defer r.Release()
			if tc.valid {
				require.Empty(t, r.Results)
				return
			}
			require.Len(t, r.Results, 1)
			require.Contains(t, r.Results[0].Path, "Encoding")
			require.Contains(t, r.Results[0].Message, "Encoding` is not allowed")
		})
	}
}

func TestOpenAPILinkReportsExternalDefinition(t *testing.T) {
	dir := t.TempDir()
	owner := filepath.Join(dir, "responses.yaml")
	require.NoError(t, os.WriteFile(owner, []byte("Record:\n  description: ok\n  links:\n    orphan: {operationId: nowhere}\n"), 0600))
	spec := `openapi: 3.1.1
info: {title: Test, version: '1'}
paths:
  /shop:
    get: {responses: {'200': {$ref: 'responses.yaml#/Record'}}}
`
	rule := rulesets.GetOAS3DocumentRelationshipsRule()
	r := ApplyRulesToRuleSet(&RuleSetExecution{
		Spec: []byte(spec), SpecFileName: filepath.Join(dir, "root.yaml"),
		Base: dir, AllowLookup: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{rule.Id: rule}},
	})
	defer r.Release()
	require.Len(t, r.Results, 1)
	f := r.Results[0]
	require.Equal(t, "$.Record.links.orphan.operationId", f.Path)
	require.Equal(t, 4, f.StartNode.Line)
	require.NotNil(t, f.Origin)
	require.Equal(t, owner, f.Origin.AbsoluteLocation)
}

func TestOpenAPILinkTargetsInAlreadyLoadedDocuments(t *testing.T) {
	dir := t.TempDir()
	const other = `openapi: 3.1.1
info: {title: Jobs, version: '1'}
x-jobs: &jobs {get: {operationId: listJobs, responses: {'200': {description: ok}}}}
paths:
  /jobs: {<<: *jobs}
components:
  responses:
    Accepted:
      description: ok
      links:
        next: {operationId: listJobs}
        broken: {operationId: exampleOnly}
        sameDocument: {operationRef: '#/paths/~1jobs/get'}
        entryOnly: {operationRef: '#/paths/~1shop/get'}
  examples:
    Fake: {value: {operationId: exampleOnly}}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "other.yaml"), []byte(other), 0600))
	const root = `openapi: 3.1.1
info: {title: Test, version: '1'}
paths:
  /shop:
    get: {responses: {'200': {$ref: 'other.yaml#/components/responses/Accepted'}}}
`
	rule := rulesets.GetOAS3DocumentRelationshipsRule()
	r := ApplyRulesToRuleSet(&RuleSetExecution{
		Spec: []byte(root), SpecFileName: filepath.Join(dir, "root.yaml"),
		Base: dir, AllowLookup: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{rule.Id: rule}},
	})
	defer r.Release()
	var paths []string
	for _, f := range r.Results {
		paths = append(paths, f.Path)
		require.Equal(t, filepath.Join(dir, "other.yaml"), f.Origin.AbsoluteLocation)
	}
	require.ElementsMatch(t, []string{"$.components.responses.Accepted.links.broken.operationId", "$.components.responses.Accepted.links.entryOnly.operationRef"}, paths)
}

func TestOpenAPILocalOperationRefInPathItemFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "path.yaml")
	require.NoError(t, os.WriteFile(file, []byte(`get:
  responses:
    '200':
      description: ok
      links:
        next: {operationRef: '#/post'}
        missing: {operationRef: '#post'}
post:
  <<: {responses: {'204': {description: ok}}}
`), 0600))
	const root = `openapi: 3.1.1
info: {title: Test, version: '1'}
paths:
  /shop: {$ref: 'path.yaml'}
`
	rule := rulesets.GetOAS3DocumentRelationshipsRule()
	r := ApplyRulesToRuleSet(&RuleSetExecution{
		Spec: []byte(root), SpecFileName: filepath.Join(dir, "root.yaml"),
		Base: dir, AllowLookup: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{rule.Id: rule}},
	})
	defer r.Release()
	require.Len(t, r.Results, 1)
	require.Equal(t, "$.get.responses['200'].links.missing.operationRef", r.Results[0].Path)
	require.Equal(t, file, r.Results[0].Origin.AbsoluteLocation)
}

func TestServerVariableDefaultSeverityByVersion(t *testing.T) {
	const doc = `openapi: %s
info: {title: Servers, version: '1'}
paths: {}
servers:
  - url: https://{env}.example.com/{region}/{stage}
    variables:
      env: {default: qa, enum: [prod, staging]}
      region: {default: us, enum: [us, eu]}
      stage: {default: dev}
components:
  examples:
    literal: {value: {variables: {env: {default: qa, enum: [prod]}}}}
`
	for _, tc := range []struct{ version, rule, severity, word string }{
		{"3.0.4", rulesets.Oas30ServerVariableDefault, model.SeverityWarn, "should"},
		{"3.1.1", rulesets.Oas3DocumentRelationships, model.SeverityError, "must"},
		{"3.2.0", rulesets.Oas3DocumentRelationships, model.SeverityError, "must"},
	} {
		t.Run(tc.version, func(t *testing.T) {
			rs := rulesets.BuildDefaultRuleSets().GenerateOpenAPIRecommendedRuleSet()
			r := ApplyRulesToRuleSet(&RuleSetExecution{Spec: []byte(fmt.Sprintf(doc, tc.version)), RuleSet: rs})
			defer r.Release()
			var matches []model.RuleFunctionResult
			for _, f := range r.Results {
				if f.RuleId == rulesets.Oas30ServerVariableDefault || f.RuleId == rulesets.Oas3DocumentRelationships {
					matches = append(matches, f)
				}
			}
			require.Len(t, matches, 1)
			f := matches[0]
			require.Equal(t, tc.rule, f.RuleId)
			require.Equal(t, tc.severity, f.RuleSeverity)
			require.Contains(t, f.Message, tc.word)
			require.Equal(t, "$.servers[0].variables.env.default", f.Path)
			require.Equal(t, "qa", f.StartNode.Value)
		})
	}
}

func TestServerVariableDefaultsWithSharedAndEmptyEnums(t *testing.T) {
	const spec = `openapi: 3.0.4
info: {title: Test, version: '1'}
x-default: &d {default: qa}
x-empty: &empty []
paths: {}
servers:
  - url: '/{good}/{bad}/{empty}'
    variables:
      good: {<<: *d, enum: [qa]}
      bad: {<<: *d, enum: [prod]}
      empty: {default: qa, enum: *empty}
`
	rule := rulesets.GetOAS30ServerVariableDefaultRule()
	r := ApplyRulesToRuleSet(&RuleSetExecution{Spec: []byte(spec), RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{rule.Id: rule}}})
	defer r.Release()
	var messages []string
	for _, f := range r.Results {
		messages = append(messages, f.Message)
	}
	require.ElementsMatch(t, []string{
		`server variable "bad" default "qa" should be a member of its enum`,
		`server variable "empty" default "qa" should be a member of its enum`,
	}, messages)
}
