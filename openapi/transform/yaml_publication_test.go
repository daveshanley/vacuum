// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package transform

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestFilterYAMLAliasesPreservePublicationPresentation(t *testing.T) {
	root := parseTestYAML(t, `# OpenAPI first
openapi: 3.1.0
info: {title: "Public API", version: '1.0'} # keep info
x-operation: &operation # anchor comment
  # operation summary
  summary: "Keep this quote"
  x-number: 00123
  description: |
    First line.
    Second line.
  tags: [public]
  responses: {}
paths:
  /first:
    get: *operation # alias comment
  /second:
    get: *operation
components: {schemas: {}}
`)
	_, err := FilterOperationsByTags(root, "3.1.0", TagFilterOptions{IncludeTags: []string{"public"}})
	require.NoError(t, err)
	mapping := documentRoot(root)
	var keys []string
	for i := 0; i < len(mapping.Content); i += 2 {
		keys = append(keys, mapping.Content[i].Value)
	}
	assert.Equal(t, []string{"openapi", "info", "x-operation", "paths", "components"}, keys)
	paths := mapValue(mapping, "paths")
	first := mapValue(mapValue(paths, "/first"), "get")
	second := mapValue(mapValue(paths, "/second"), "get")
	require.NotNil(t, first)
	require.NotNil(t, second)
	assert.NotSame(t, first, second)
	assert.Equal(t, yaml.DoubleQuotedStyle, mapValue(first, "summary").Style)
	assert.Equal(t, yaml.LiteralStyle, mapValue(first, "description").Style)
	assert.Equal(t, yaml.FlowStyle, mapValue(mapping, "info").Style)
	encoded, err := yaml.Marshal(root)
	require.NoError(t, err)
	output := string(encoded)
	for _, comment := range []string{"# OpenAPI first", "# keep info", "# operation summary", "# alias comment", "# anchor comment"} {
		assert.Contains(t, output, comment)
	}
	assert.NotContains(t, output, "&operation")
	assert.NotContains(t, output, "*operation")
	var reparsed yaml.Node
	require.NoError(t, yaml.Unmarshal(encoded, &reparsed))
	assert.Equal(t, "Keep this quote", mapValue(first, "summary").Value)
	assert.Equal(t, "00123", mapValue(first, "x-number").Value)
	assert.Contains(t, output, "x-number: 00123")
}

func TestFilterYAMLMergePrecedenceAndOrder(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.1.0
x-first: &first
  tags: [internal]
  summary: 'First wins'
  responses: {}
x-second: &second
  tags: [internal]
  summary: 'Second loses'
  description: >-
    Folded description
paths:
  /x:
    get:
      operationId: getX
      responses: {'200': {description: explicit before}}
      # merge comment
      <<: [*first, *second]
      tags: [public]
`)
	stats, err := FilterOperationsByTags(root, "3.1.0", TagFilterOptions{IncludeTags: []string{"public"}})
	require.NoError(t, err)
	assert.Equal(t, 1, stats.OperationsKept)
	operation := mapValue(mapValue(mapValue(documentRoot(root), "paths"), "/x"), "get")
	assert.Equal(t, "First wins", mapValue(operation, "summary").Value)
	assert.Equal(t, "explicit before", mapValue(mapValue(mapValue(operation, "responses"), "200"), "description").Value)
	assert.Equal(t, yaml.SingleQuotedStyle, mapValue(operation, "summary").Style)
	assert.Equal(t, yaml.FoldedStyle, mapValue(operation, "description").Style)
	var keys []string
	for i := 0; i < len(operation.Content); i += 2 {
		keys = append(keys, operation.Content[i].Value)
	}
	assert.Equal(t, []string{"operationId", "responses", "summary", "description", "tags"}, keys)
	encoded, err := yaml.Marshal(root)
	require.NoError(t, err)
	assert.Contains(t, string(encoded), "# merge comment")
	assert.NotContains(t, string(encoded), "<<:")
}

func TestPublicationRejectsInvalidYAMLWithoutMutation(t *testing.T) {
	for _, test := range []struct {
		name, source, message string
	}{
		{"scalar merge", "x-data: {<<: invalid}", "merge source must be a mapping"},
		{"invalid merge sequence", "x-data: {<<: [{description: ok}, invalid]}", "merge source must be a mapping"},
		{"recursive alias", "x-data: &cycle {child: *cycle}", "recursive YAML alias"},
		{"duplicate local key", "x-data: &base {a: b}\nx-copy: {<<: *base, a: first, a: second}", "duplicate YAML mapping key"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := parseTestYAML(t, "openapi: 3.1.0\npaths: {}\n"+test.source+"\n")
			before, err := yaml.Marshal(root)
			require.NoError(t, err)
			_, err = FilterOperationsByTags(root, "3.1.0", TagFilterOptions{IncludeTags: []string{"public"}})
			require.ErrorContains(t, err, test.message)
			after, err := yaml.Marshal(root)
			require.NoError(t, err)
			assert.Equal(t, string(before), string(after))
		})
	}
}

func TestFilterRejectsExponentialYAMLAliasExpansion(t *testing.T) {
	var source strings.Builder
	source.WriteString("openapi: 3.1.0\npaths: {}\nx-a0: &a0 [small]\n")
	for i := 1; i <= 7; i++ {
		fmt.Fprintf(&source, "x-a%d: &a%d [%s]\n", i, i, strings.TrimSuffix(strings.Repeat(fmt.Sprintf("*a%d, ", i-1), 10), ", "))
	}
	root := parseTestYAML(t, source.String())
	before, err := yaml.Marshal(root)
	require.NoError(t, err)
	_, err = FilterOperationsByTags(root, "3.1.0", TagFilterOptions{IncludeTags: []string{"public"}})
	require.ErrorContains(t, err, "maximum node count")
	require.ErrorContains(t, ValidateBundled(root), "maximum node count")
	after, err := yaml.Marshal(root)
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after))
}

func TestPublicationRejectsExcessiveYAMLAliasDepth(t *testing.T) {
	var source strings.Builder
	source.WriteString("openapi: 3.1.0\npaths: {}\nx-a0: &a0 [small]\n")
	for i := 1; i <= 501; i++ {
		fmt.Fprintf(&source, "x-a%d: &a%d [*a%d]\n", i, i, i-1)
	}
	root := parseTestYAML(t, source.String())
	require.ErrorContains(t, ValidateBundled(root), "maximum depth")
}

func TestPruneMergeKeepsOrdinaryAliasOwnership(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.1.0
x-defaults: &defaults
  description: 'Keep order'
paths:
  /x:
    get:
      <<: *defaults
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema: &used {type: string}
components:
  schemas:
    Kept: *used
    Removed: {type: string}
`)
	_, err := PruneUnusedComponents(root, "3.1.0")
	require.NoError(t, err)
	encoded, err := yaml.Marshal(root)
	require.NoError(t, err)
	assert.Equal(t, 1, strings.Count(string(encoded), "&used"))
	var reparsed yaml.Node
	require.NoError(t, yaml.Unmarshal(encoded, &reparsed))
	assert.Contains(t, string(encoded), "description: 'Keep order'")
}

func TestPublicationReferenceErrorsUseVacuumJSONPaths(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.1.0
paths:
  /x:
    get:
      responses:
        '200': {$ref: remote.yaml}
`)
	err := ValidateBundled(root)
	require.ErrorContains(t, err, "$.paths['/x'].get.responses['200'].$ref")
	assert.Equal(t, `$.components.schemas['a.b']['quote\'slash\\']`, jsonPath([]string{"components", "schemas", "a.b", "quote'slash\\"}))
	assert.Equal(t, `$['']`, jsonPath([]string{""}))
}
