package transform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

func parseTestYAML(t *testing.T, source string) *yaml.Node {
	t.Helper()
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(source), &node))
	return &node
}

func TestFilterOperationsByTagsTruthTable(t *testing.T) {
	cases := []struct {
		name     string
		tags     string
		include  []string
		strategy MatchStrategy
		kept     int
	}{
		{"any first", "[public]", []string{"public", "partner"}, MatchAny, 1},
		{"any second", "[partner]", []string{"public", "partner"}, MatchAny, 1},
		{"any extra", "[public, internal]", []string{"public", "partner"}, MatchAny, 1},
		{"any none", "[internal]", []string{"public", "partner"}, MatchAny, 0},
		{"all partial", "[public]", []string{"public", "partner"}, MatchAll, 0},
		{"all exact", "[public, partner]", []string{"public", "partner"}, MatchAll, 1},
		{"all extra", "[public, partner, shared]", []string{"public", "partner"}, MatchAll, 1},
		{"case sensitive", "[Public]", []string{"public"}, MatchAny, 0},
		{"duplicate includes", "[public]", []string{"public", "public"}, MatchAll, 1},
		{"missing", "", []string{"public"}, MatchAny, 0},
		{"malformed scalar", "public", []string{"public"}, MatchAny, 0},
		{"malformed member", "[{name: public}]", []string{"public"}, MatchAny, 0},
		{"mixed malformed member", "[public, {name: public}]", []string{"public"}, MatchAny, 0},
		{"non-string scalar", "[1, public]", []string{"public"}, MatchAny, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tagLine := ""
			if tc.tags != "" {
				tagLine = "\n      tags: " + tc.tags
			}
			root := parseTestYAML(t, "openapi: 3.0.3\npaths:\n  /x:\n    get:"+tagLine+"\n      responses: {}\n")
			stats, err := FilterOperationsByTags(root, "3.0.3", TagFilterOptions{IncludeTags: tc.include, MatchStrategy: tc.strategy})
			require.NoError(t, err)
			assert.Equal(t, 1, stats.OperationsSeen)
			assert.Equal(t, tc.kept, stats.OperationsKept)
			assert.Equal(t, 1-tc.kept, stats.OperationsRemoved)
			paths := mapValue(documentRoot(root), "paths")
			assert.Equal(t, tc.kept, len(paths.Content)/2)
		})
	}
}

func TestFilterOperationsByTagsStructuralSurfaces(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.2.0
paths:
  /mixed:
    summary: metadata does not keep an empty path
    parameters: []
    get:
      tags: [public]
      operationId: publicGet
      callbacks:
        event:
          '{$request.body#/url}':
            post:
              tags: [internal]
              operationId: callbackInternal
    post:
      tags: [internal]
      operationId: internalPost
  /ref:
    $ref: '#/components/pathItems/PublicItem'
  /cycle:
    $ref: '#/components/pathItems/CycleA'
webhooks:
  event:
    query:
      tags: [public]
    post:
      tags: [internal]
components:
  pathItems:
    PublicItem:
      get:
        tags: [public]
      additionalOperations:
        COPY:
          tags: [public]
        PURGE:
          tags: [internal]
    CycleA:
      $ref: '#/components/pathItems/CycleB'
    CycleB:
      $ref: '#/components/pathItems/CycleA'
  callbacks:
    Empty:
      '{$request.body#/url}':
        post:
          tags: [internal]
`)
	stats, err := FilterOperationsByTags(root, "3.2.0", TagFilterOptions{IncludeTags: []string{"public"}, MatchStrategy: MatchAny})
	require.NoError(t, err)
	assert.Equal(t, 9, stats.OperationsSeen)
	assert.Equal(t, 4, stats.OperationsKept)
	assert.Equal(t, 5, stats.OperationsRemoved)
	assert.Equal(t, 1, stats.PathItemsRemoved)
	assert.Equal(t, 0, stats.WebhooksRemoved)
	assert.Equal(t, 2, stats.CallbackItemsRemoved)

	doc := documentRoot(root)
	paths := mapValue(doc, "paths")
	assert.Nil(t, mapValue(paths, "/cycle"))
	mixed := mapValue(paths, "/mixed")
	assert.NotNil(t, mapValue(mixed, "get"))
	assert.Nil(t, mapValue(mixed, "post"))
	assert.Nil(t, mapValue(mapValue(mixed, "get"), "callbacks"))
	webhook := mapValue(mapValue(doc, "webhooks"), "event")
	assert.NotNil(t, mapValue(webhook, "query"))
	assert.Nil(t, mapValue(webhook, "post"))
	publicItem := mapValue(mapValue(mapValue(doc, "components"), "pathItems"), "PublicItem")
	assert.Equal(t, 1, len(mapValue(publicItem, "additionalOperations").Content)/2)
}

func TestFilterOperationsByTagsSwaggerNoOpAndErrors(t *testing.T) {
	swagger := parseTestYAML(t, "swagger: '2.0'\npaths:\n  /x:\n    get:\n      tags: [public]\n      responses: {}\n    query:\n      tags: [public]\n")
	stats, err := FilterOperationsByTags(swagger, "2.0", TagFilterOptions{IncludeTags: []string{"public"}})
	require.NoError(t, err)
	assert.Equal(t, 1, stats.OperationsSeen)
	assert.NotNil(t, mapValue(mapValue(documentRoot(swagger), "paths"), "/x"))

	original := parseTestYAML(t, "openapi: 3.0.0\npaths: {}\n")
	before, err := yaml.Marshal(original)
	require.NoError(t, err)
	stats, err = FilterOperationsByTags(original, "3.0.0", TagFilterOptions{})
	require.NoError(t, err)
	assert.Equal(t, FilterStats{}, stats)
	after, err := yaml.Marshal(original)
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after))

	_, err = FilterOperationsByTags(original, "3.0.0", TagFilterOptions{IncludeTags: []string{"x"}, MatchStrategy: "nope"})
	assert.ErrorContains(t, err, "expected any or all")
	for _, tag := range []string{"", " \t"} {
		_, err = FilterOperationsByTags(original, "3.0.0", TagFilterOptions{IncludeTags: []string{tag}})
		assert.ErrorContains(t, err, "must not be empty or whitespace")
	}
	_, err = FilterOperationsByTags(&yaml.Node{}, "3.0.0", TagFilterOptions{IncludeTags: []string{"x"}})
	assert.ErrorContains(t, err, "root must be a mapping")

	oas30 := parseTestYAML(t, "openapi: 3.0.3\npaths: {}\nwebhooks:\n  future:\n    post: {tags: [internal]}\n")
	_, err = FilterOperationsByTags(oas30, "3.0.3", TagFilterOptions{IncludeTags: []string{"public"}})
	require.NoError(t, err)
	assert.NotNil(t, mapValue(mapValue(documentRoot(oas30), "webhooks"), "future"))
}

func TestHasReachableOperationsIgnoresOrphansAndPathsExtensions(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.1.0
paths:
  x-publication:
    get: {tags: [public]}
components:
  pathItems:
    Orphan:
      get: {tags: [public]}
`)
	stats, err := FilterOperationsByTags(root, "3.1.0", TagFilterOptions{IncludeTags: []string{"public"}})
	require.NoError(t, err)
	assert.Equal(t, 1, stats.OperationsKept)
	assert.False(t, HasReachableOperations(root, "3.1.0"))
	assert.NotNil(t, mapValue(mapValue(documentRoot(root), "paths"), "x-publication"))

	paths := mapValue(documentRoot(root), "paths")
	paths.Content = append(paths.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "/public"},
		&yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "$ref"},
			{Kind: yaml.ScalarNode, Value: "#/components/pathItems/Orphan"},
		}},
	)
	assert.True(t, HasReachableOperations(root, "3.1.0"))
	assert.False(t, HasReachableOperations(&yaml.Node{}, "3.1.0"))
	missing := parseTestYAML(t, "openapi: 3.1.0\npaths:\n  /missing: {$ref: '#/components/pathItems/Missing'}\n")
	assert.False(t, HasReachableOperations(missing, "3.1.0"))

	cycle := parseTestYAML(t, `openapi: 3.1.0
paths:
  /cycle: {$ref: '#/components/pathItems/A'}
components:
  pathItems:
    A: {$ref: '#/components/pathItems/B'}
    B: {$ref: '#/components/pathItems/A'}
`)
	assert.False(t, HasReachableOperations(cycle, "3.1.0"))

	additional := parseTestYAML(t, `openapi: 3.2.0
paths:
  /additional:
    additionalOperations:
      COPY: {tags: [public]}
`)
	assert.True(t, HasReachableOperations(additional, "3.2.0"))

	webhook := parseTestYAML(t, `openapi: 3.2.0
paths: bad
webhooks:
  event:
    query: {tags: [public]}
`)
	assert.True(t, HasReachableOperations(webhook, "3.2.0"))
	assert.False(t, HasReachableOperations(webhook, "3.0.3"))
}

func TestFilterOperationsByTagsDanglingLinks(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.0.3
paths:
  /public:
    get:
      tags: [public]
      responses:
        '200':
          description: ok
          links:
            byId:
              operationId: hidden
            byRef:
              operationRef: '#/paths/~1hidden/get'
            untouched:
              operationId: never-existed
  /hidden:
    get:
      tags: [internal]
      operationId: hidden
      responses: {}
`)
	stats, err := FilterOperationsByTags(root, "3.0.3", TagFilterOptions{IncludeTags: []string{"public"}, MatchStrategy: MatchAny})
	require.NoError(t, err)
	require.Len(t, stats.Warnings, 2)
	assert.Contains(t, stats.Warnings[0].Message+stats.Warnings[1].Message, "hidden")
	assert.True(t, strings.HasPrefix(stats.Warnings[0].Path, "$"))
}

func TestDanglingLinkWarningRemovedWithOwningComponent(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.0.3
paths:
  /public:
    get:
      tags: [public]
      responses: {'200': {description: ok}}
  /hidden:
    get:
      tags: [internal]
      operationId: hidden
      responses: {}
components:
  links:
    Unused:
      operationId: hidden
`)
	filter, err := FilterOperationsByTags(root, "3.0.3", TagFilterOptions{IncludeTags: []string{"public"}})
	require.NoError(t, err)
	require.Len(t, filter.Warnings, 1)
	prune, err := PruneUnusedComponents(root, "3.0.3")
	require.NoError(t, err)
	assert.Empty(t, RetainWarningsForPrunedDocument(filter.Warnings, prune))
}

func TestFilterReferenceCyclesWithConcreteOperation(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.0.3
paths:
  /cycle:
    $ref: '#/components/pathItems/A'
components:
  pathItems:
    A:
      $ref: '#/components/pathItems/B'
    B:
      $ref: '#/components/pathItems/A'
      get:
        tags: [public]
`)
	stats, err := FilterOperationsByTags(root, "3.0.3", TagFilterOptions{IncludeTags: []string{"public"}})
	require.NoError(t, err)
	assert.Equal(t, 1, stats.OperationsKept)
	assert.NotNil(t, mapValue(mapValue(documentRoot(root), "paths"), "/cycle"))
}

func TestFilterReferenceCycleRetainsEveryReachableEntry(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.1.0
paths:
  /a: {$ref: '#/components/pathItems/A'}
  /b: {$ref: '#/components/pathItems/B'}
  /callback:
    get:
      tags: [public]
      callbacks:
        event: {$ref: '#/components/callbacks/B'}
components:
  pathItems:
    A:
      $ref: '#/components/pathItems/B'
      get: {tags: [public]}
    B:
      $ref: '#/components/pathItems/A'
  callbacks:
    A:
      $ref: '#/components/callbacks/B'
      '{$request.body#/url}':
        post: {tags: [public]}
    B:
      $ref: '#/components/callbacks/A'
`)
	stats, err := FilterOperationsByTags(root, "3.1.0", TagFilterOptions{IncludeTags: []string{"public"}})
	require.NoError(t, err)
	assert.Equal(t, 3, stats.OperationsKept)
	paths := mapValue(documentRoot(root), "paths")
	assert.NotNil(t, mapValue(paths, "/a"))
	assert.NotNil(t, mapValue(paths, "/b"))
	callback := mapValue(mapValue(mapValue(paths, "/callback"), "get"), "callbacks")
	assert.NotNil(t, mapValue(callback, "event"))
}

func TestFilterDefersCallbackPathItemBackEdges(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.1.0
paths:
  /root: {$ref: '#/components/pathItems/A'}
components:
  pathItems:
    A:
      get:
        tags: [public]
        callbacks:
          reachable:
            '{$request.body#/url}': {$ref: '#/components/pathItems/A'}
          empty:
            '{$request.body#/url}': {$ref: '#/components/pathItems/EmptyA'}
    EmptyA: {$ref: '#/components/pathItems/EmptyB'}
    EmptyB: {$ref: '#/components/pathItems/EmptyA'}
`)
	_, err := FilterOperationsByTags(root, "3.1.0", TagFilterOptions{IncludeTags: []string{"public"}})
	require.NoError(t, err)
	operation := mapValue(mapValue(mapValue(mapValue(documentRoot(root), "components"), "pathItems"), "A"), "get")
	callbacks := mapValue(operation, "callbacks")
	assert.NotNil(t, mapValue(callbacks, "reachable"))
	assert.Nil(t, mapValue(callbacks, "empty"))
}

func TestDanglingLinksIgnoreUnrelatedLinksKeys(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.0.3
paths:
  /public:
    get:
      tags: [public]
      x-config:
        links:
          operationId: hidden
          operationRef: '#/Bad~2Pointer'
      responses: {}
  /hidden:
    get:
      tags: [internal]
      operationId: hidden
      responses: {}
`)
	stats, err := FilterOperationsByTags(root, "3.0.3", TagFilterOptions{IncludeTags: []string{"public"}})
	require.NoError(t, err)
	assert.Empty(t, stats.Warnings)
}

func BenchmarkFilterOperationsByTags(b *testing.B) {
	var source strings.Builder
	source.WriteString("openapi: 3.0.3\npaths:\n")
	for i := 0; i < 500; i++ {
		source.WriteString("  /p")
		source.WriteString(fmt.Sprint(i))
		source.WriteString(":\n    get:\n      tags: [public]\n")
	}
	data := source.String()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var root yaml.Node
		_ = yaml.Unmarshal([]byte(data), &root)
		_, _ = FilterOperationsByTags(&root, "3.0.3", TagFilterOptions{IncludeTags: []string{"public"}})
	}
}

func BenchmarkIssue948GitHubFixture(b *testing.B) {
	data, err := os.ReadFile(filepath.Join("..", "..", "model", "test_files", "api.github.com.yaml"))
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var root yaml.Node
		if err := yaml.Unmarshal(data, &root); err != nil {
			b.Fatal(err)
		}
		_, _ = FilterOperationsByTags(&root, "3.0.3", TagFilterOptions{IncludeTags: []string{"public"}})
	}
}
