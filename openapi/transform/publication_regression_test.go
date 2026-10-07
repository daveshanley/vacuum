package transform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestPrunePreservesOrdinaryAnchorReferences(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.1.0
paths:
  /x:
    get:
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema: {$ref: '#kept'}
components:
  schemas:
    Kept: {$anchor: kept, type: string}
    Removed: {type: string}
`)
	stats, err := PruneUnusedComponents(root, "3.1.0")
	require.NoError(t, err)
	assert.Equal(t, 1, stats.ComponentsRemoved)
	assert.NotNil(t, mapValue(mapValue(mapValue(documentRoot(root), "components"), "schemas"), "Kept"))
}

func TestFilterResolvesYAMLMergeAndAliasOperations(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.1.0
x-shared: &shared
  post: {tags: [internal], responses: {}}
x-public: &public
  tags: [public]
  responses: {}
paths:
  /x:
    <<: *shared
    get: *public
`)
	stats, err := FilterOperationsByTags(root, "3.1.0", TagFilterOptions{IncludeTags: []string{"public"}})
	require.NoError(t, err)
	assert.Equal(t, 1, stats.OperationsKept)
	assert.Equal(t, 1, stats.OperationsRemoved)
	path := mapValue(mapValue(documentRoot(root), "paths"), "/x")
	require.NotNil(t, path)
	assert.NotNil(t, mapValue(path, "get"))
	assert.Nil(t, mapValue(path, "post"))
	assert.Nil(t, mapValue(path, "<<"))
}

func TestPruneResolvesMergedComponentSections(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.1.0
x-shared: &shared
  Kept: {type: string}
paths:
  /x:
    get:
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Kept'}
components:
  schemas:
    <<: *shared
    Removed: {type: string}
`)
	stats, err := PruneUnusedComponents(root, "3.1.0")
	require.NoError(t, err)
	assert.Equal(t, 1, stats.ComponentsRemoved)
}

func TestPruneIgnoresSchemaLiteralData(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.1.0
paths:
  /x:
    get:
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                default: {$ref: 'payload.json'}
                const: {$ref: '#/components/schemas/Missing'}
                enum: [{$id: 'payload.json'}]
components:
  schemas:
    Removed: {type: string}
`)
	stats, err := PruneUnusedComponents(root, "3.1.0")
	require.NoError(t, err)
	assert.Equal(t, 1, stats.ComponentsRemoved)
}

func TestPruneRejectsScopedSchemaResourcesWithoutMutation(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.1.0
paths: {}
components:
  schemas:
    Scoped: {$id: 'schema.json', type: string}
`)
	_, err := PruneUnusedComponents(root, "3.1.0")
	require.ErrorContains(t, err, "schema $id scopes")
	assert.NotNil(t, mapValue(mapValue(mapValue(documentRoot(root), "components"), "schemas"), "Scoped"))
}

func TestFilterRejectsRecursiveYAMLAliases(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.1.0
paths: &paths
  /x: *paths
`)
	_, err := FilterOperationsByTags(root, "3.1.0", TagFilterOptions{IncludeTags: []string{"public"}})
	require.ErrorContains(t, err, "cannot expand YAML aliases")
}

func TestFilterRejectsUnbundledReferencesBeforeMutation(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.1.0
paths:
  /x:
    get: {tags: [internal], responses: {}}
components:
  schemas:
    External: {$ref: 'schema.yaml'}
`)
	_, err := FilterOperationsByTags(root, "3.1.0", TagFilterOptions{IncludeTags: []string{"public"}})
	require.ErrorContains(t, err, "require a bundled OpenAPI document")
	assert.NotNil(t, mapValue(mapValue(documentRoot(root), "paths"), "/x"))
}

func TestPruneFollowsAliasAtSchemaUseSite(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.1.0
paths:
  /x:
    get:
      responses:
        '200':
          description: ok
          content:
            application/json:
              example: &payload {$ref: '#/components/schemas/Kept'}
              schema: *payload
components:
  schemas:
    Kept: {type: string}
    Removed: {type: string}
`)
	stats, err := PruneUnusedComponents(root, "3.1.0")
	require.NoError(t, err)
	assert.Equal(t, 1, stats.ComponentsRemoved)
	assert.NotNil(t, mapValue(mapValue(mapValue(documentRoot(root), "components"), "schemas"), "Kept"))
}

func TestValidateBundledChecksAliasAtUseSite(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.1.0
paths:
  /x:
    get:
      responses:
        '200':
          description: ok
          content:
            application/json:
              example: &payload {$ref: 'other.yaml'}
              schema: *payload
`)
	require.ErrorContains(t, ValidateBundled(root), "external reference")
}

func TestPruneRetainsSecurityForNamedPathItems(t *testing.T) {
	for _, name := range []string{"x-Public", "examples"} {
		t.Run(name, func(t *testing.T) {
			root := parseTestYAML(t, `openapi: 3.1.0
paths:
  /test:
    $ref: '#/components/pathItems/`+name+`'
components:
  pathItems:
    `+name+`:
      get:
        security: [{ApiKey: []}]
        responses: {'200': {description: ok}}
  securitySchemes:
    ApiKey: {type: apiKey, in: header, name: token}
    Unused: {type: apiKey, in: header, name: other}
`)
			stats, err := PruneUnusedComponents(root, "3.1.0")
			require.NoError(t, err)
			assert.Equal(t, 1, stats.ComponentsRemoved)
			schemes := mapValue(mapValue(documentRoot(root), "components"), "securitySchemes")
			assert.NotNil(t, mapValue(schemes, "ApiKey"))
		})
	}
}

func TestPruneRetainsLinkedComponentOperation(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.1.0
paths:
  /test:
    get:
      tags: [public]
      responses:
        '200':
          description: ok
          links:
            next: {operationRef: '#/components/pathItems/Next/get'}
components:
  pathItems:
    Next:
      get:
        tags: [public]
        responses: {'200': {description: ok}}
    Unused:
      get:
        tags: [public]
        responses: {'200': {description: ok}}
`)
	filter, err := FilterOperationsByTags(root, "3.1.0", TagFilterOptions{IncludeTags: []string{"public"}})
	require.NoError(t, err)
	assert.Empty(t, filter.Warnings)
	stats, err := PruneUnusedComponents(root, "3.1.0")
	require.NoError(t, err)
	assert.Equal(t, 1, stats.ComponentsRemoved)
	items := mapValue(mapValue(documentRoot(root), "components"), "pathItems")
	assert.NotNil(t, mapValue(mapValue(items, "Next"), "get"))
	assert.Nil(t, mapValue(items, "Unused"))

	// A subsequent publication can remove that operation without suppressing
	// the warning on the retained link that still names it.
	tags := mapValue(mapValue(mapValue(items, "Next"), "get"), "tags")
	tags.Content[0].Value = "internal"
	filter, err = FilterOperationsByTags(root, "3.1.0", TagFilterOptions{IncludeTags: []string{"public"}})
	require.NoError(t, err)
	require.Len(t, filter.Warnings, 1)
	stats, err = PruneUnusedComponents(root, "3.1.0")
	require.NoError(t, err)
	warnings := RetainWarningsForPrunedDocument(filter.Warnings, stats)
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0].Message, "link targets filtered operation")
}

func TestFilterTreatsExtensionLikeWebhookNamesAsOperations(t *testing.T) {
	for _, tags := range []string{"internal", "public"} {
		t.Run(tags, func(t *testing.T) {
			root := parseTestYAML(t, `openapi: 3.1.0
paths: {}
webhooks:
  x-hook:
    post:
      tags: [`+tags+`]
      responses: {'200': {description: ok}}
`)
			assert.True(t, HasReachableOperations(root, "3.1.0"))
			stats, err := FilterOperationsByTags(root, "3.1.0", TagFilterOptions{IncludeTags: []string{"public"}})
			require.NoError(t, err)
			assert.Equal(t, 1, stats.OperationsSeen)
			assert.Equal(t, tags == "public", HasReachableOperations(root, "3.1.0"))
			hook := mapValue(mapValue(documentRoot(root), "webhooks"), "x-hook")
			if tags == "public" {
				assert.NotNil(t, hook)
				assert.Equal(t, 1, stats.OperationsKept)
			} else {
				assert.Nil(t, hook)
				assert.Equal(t, 1, stats.OperationsRemoved)
				assert.Equal(t, 1, stats.WebhooksRemoved)
			}
		})
	}
}

func TestPrunePreservesNumericExampleAndComponentAliasTargets(t *testing.T) {
	for _, tc := range []struct{ fixture, section, name string }{
		{"numeric-example.yaml", "examples", "Actual"},
		{"example-alias.yaml", "schemas", "User"},
	} {
		t.Run(tc.fixture, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "..", "cmd", "test_data", "issue_948", tc.fixture))
			require.NoError(t, err)
			root := parseTestYAML(t, string(data))
			_, err = PruneUnusedComponents(root, "3.1.0")
			require.NoError(t, err)
			component := mapValue(mapValue(mapValue(documentRoot(root), "components"), tc.section), tc.name)
			assert.NotNil(t, component)
			if tc.fixture == "numeric-example.yaml" {
				external := strings.ReplaceAll(string(data), "#/components/examples/Actual", "external.yaml#/Actual")
				require.ErrorContains(t, ValidateBundled(parseTestYAML(t, external)), "external reference")
			} else {
				assert.NotNil(t, mapValue(mapValue(mapValue(documentRoot(root), "components"), "examples"), "Template"))
			}
		})
	}
}

func TestFilterPreservesReferencedRootMetadataWithoutPrivateOperations(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "cmd", "test_data", "issue_948", "referenced-path.yaml"))
	require.NoError(t, err)
	for _, rootKey := range []string{"paths", "webhooks"} {
		for _, order := range []string{"forward", "reverse"} {
			t.Run(rootKey+"/"+order, func(t *testing.T) {
				source := strings.ReplaceAll(string(data), "paths", rootKey)
				root := parseTestYAML(t, source)
				container := mapValue(documentRoot(root), rootKey)
				if order == "reverse" {
					for left, right := 0, len(container.Content)-2; left < right; left, right = left+2, right-2 {
						container.Content[left], container.Content[right] = container.Content[right], container.Content[left]
						container.Content[left+1], container.Content[right+1] = container.Content[right+1], container.Content[left+1]
					}
				}
				stats, err := FilterOperationsByTags(root, "3.1.0", TagFilterOptions{IncludeTags: []string{"public"}})
				require.NoError(t, err)
				assert.Equal(t, 1, stats.OperationsKept)
				assert.Equal(t, 1, stats.OperationsRemoved)
				assert.Equal(t, 2, stats.PathItemsRemoved+stats.WebhooksRemoved)
				_, err = PruneUnusedComponents(root, "3.1.0")
				require.NoError(t, err)
				for _, name := range []string{"/public", "/bridge"} {
					ref := mapValue(mapValue(container, name), "$ref")
					require.NotNil(t, ref)
					assert.NotNil(t, resolveLocalNode(documentRoot(root), ref.Value))
				}
				shared := mapValue(container, "/source")
				require.NotNil(t, shared)
				assert.Nil(t, mapValue(shared, "post"))
				assert.Equal(t, "Shared metadata", mapValue(shared, "summary").Value)
				assert.NotNil(t, mapValue(shared, "parameters"))
				assert.Nil(t, mapValue(container, "/orphan-a"))
				assert.Nil(t, mapValue(container, "/orphan-b"))
				assert.NotNil(t, mapValue(mapValue(mapValue(documentRoot(root), "components"), "parameters"), "Trace"))
			})
		}
	}
}
