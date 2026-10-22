// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package transform

import (
	"fmt"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestTransformFiltersBeforePruning(t *testing.T) {
	const source = `openapi: 3.1.0
info: {title: publication, version: '1'}
paths:
  /public:
    get:
      tags: [public]
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Kept'}
  /internal:
    get:
      tags: [internal]
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Private'}
components:
  schemas:
    Kept: {type: string}
    Private: {type: integer}
`
	root := parseTestYAML(t, source)
	filter, prune, err := Transform(root, "3.1.0", Options{
		TagFilter: TagFilterOptions{IncludeTags: []string{"public"}}, PruneUnused: true,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, filter.OperationsRemoved)
	assert.Equal(t, 1, prune.ComponentsRemoved)
	assert.Nil(t, mapValue(mapValue(documentRoot(root), "paths"), "/internal"))
	schemas := mapValue(mapValue(documentRoot(root), "components"), "schemas")
	assert.NotNil(t, mapValue(schemas, "Kept"))
	assert.Nil(t, mapValue(schemas, "Private"))

	sequential := parseTestYAML(t, source)
	_, err = FilterOperationsByTags(sequential, "3.1.0", TagFilterOptions{IncludeTags: []string{"public"}})
	require.NoError(t, err)
	_, err = PruneUnusedComponents(sequential, "3.1.0")
	require.NoError(t, err)
	combinedBytes, err := yaml.Marshal(root)
	require.NoError(t, err)
	sequentialBytes, err := yaml.Marshal(sequential)
	require.NoError(t, err)
	assert.Equal(t, string(sequentialBytes), string(combinedBytes))
}

func TestTransformDoesNotPublishFilteringWhenPruningFails(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.1.0
paths:
  /public:
    get: {tags: [public], responses: {'200': {$ref: '#/components/responses/Missing'}}}
  /internal:
    get: {tags: [internal], responses: {}}
`)
	before, err := yaml.Marshal(root)
	require.NoError(t, err)
	_, _, err = Transform(root, "3.1.0", Options{
		TagFilter: TagFilterOptions{IncludeTags: []string{"public"}}, PruneUnused: true,
	})
	require.ErrorContains(t, err, "missing component")
	after, err := yaml.Marshal(root)
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after))
}

func TestOperationFeaturesRespectVersion(t *testing.T) {
	operations := []struct{ name, pathItem string }{
		{"query", `query:
      tags: [public]
      security: [{QueryAuth: []}]
      responses: {}`},
		{"additional operation", `additionalOperations:
      COPY:
        tags: [public]
        security: [{QueryAuth: []}]
        responses: {}`},
	}
	for _, version := range []string{"3.0.3", "3.1.0", "3.2.0"} {
		for _, operation := range operations {
			t.Run(version+"/"+operation.name, func(t *testing.T) {
				source := fmt.Sprintf(`openapi: %s
paths:
  /x:
    %s
components:
  securitySchemes:
    QueryAuth: {type: http, scheme: bearer}
`, version, operation.pathItem)
				root := parseTestYAML(t, source)
				supported := version == "3.2.0"
				assert.Equal(t, supported, HasReachableOperations(root, version))
				_, err := PruneUnusedComponents(root, version)
				require.NoError(t, err)
				security := mapValue(mapValue(documentRoot(root), "components"), "securitySchemes")
				assert.Equal(t, supported, mapValue(security, "QueryAuth") != nil)
				filter, err := FilterOperationsByTags(parseTestYAML(t, source), version, TagFilterOptions{IncludeTags: []string{"public"}})
				require.NoError(t, err)
				if supported {
					assert.Equal(t, 1, filter.OperationsSeen)
				} else {
					assert.Zero(t, filter.OperationsSeen)
				}
			})
		}
	}
}

func TestQueryLinkClassificationRespectsVersion(t *testing.T) {
	for _, version := range []string{"3.0.3", "3.2.0"} {
		root := parseTestYAML(t, `openapi: `+version+`
paths:
  /x:
    query:
      responses:
        '200':
          links:
            Invalid: {operationRef: '#/components/pathItems/Missing/query'}
`)
		_, err := BuildComponentGraph(root, version)
		if version == "3.2.0" {
			require.ErrorContains(t, err, "missing component")
		} else {
			require.NoError(t, err)
		}
	}
}

func TestValidateBundledRejectsQueryOnlyExternalReference(t *testing.T) {
	root := parseTestYAML(t, "openapi: 3.1.0\npaths: {}\ncomponents:\n  schemas:\n    External: {$ref: '?revision=2#/Thing'}\n")
	require.ErrorContains(t, ValidateBundled(root), "external reference")
}

func TestFilterRemovesCallbackCyclesWithoutOperations(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.0.3
paths:
  /x:
    get:
      tags: [public]
      responses: {}
      callbacks:
        loop: {$ref: '#/components/callbacks/A'}
components:
  callbacks:
    A: {$ref: '#/components/callbacks/B'}
    B: {$ref: '#/components/callbacks/A'}
`)
	_, _, err := Transform(root, "3.0.3", Options{
		TagFilter: TagFilterOptions{IncludeTags: []string{"public"}}, PruneUnused: true,
	})
	require.NoError(t, err)
	operation := mapValue(mapValue(mapValue(documentRoot(root), "paths"), "/x"), "get")
	assert.Nil(t, mapValue(operation, "callbacks"))
	assert.Nil(t, mapValue(documentRoot(root), "components"))
}
