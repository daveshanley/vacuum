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
paths:
  /jobs:
    get: {operationId: listJobs, responses: {'200': {description: ok}}}
components:
  responses:
    Accepted:
      description: ok
      links:
        next: {operationId: listJobs}
        broken: {operationId: exampleOnly}
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
	require.Len(t, r.Results, 1)
	require.Equal(t, "$.components.responses.Accepted.links.broken.operationId", r.Results[0].Path)
	require.Equal(t, filepath.Join(dir, "other.yaml"), r.Results[0].Origin.AbsoluteLocation)
}
