package transform

import (
	"github.com/pb33f/testify/require"
	"testing"
)

func TestOperationFiltersIssues997And998(t *testing.T) {
	const source = `openapi: 3.1.0
paths:
  /pets:
    get: {operationId: listPets, tags: [public, shared]}
    post: {operationId: addPet, tags: [public, internal]}
    delete: {operationId: deletePets, tags: [internal]}
    patch: {operationId: patchPet}
    head: {}
`
	cases := []struct {
		name    string
		options TagFilterOptions
		want    []string
	}{
		{"exclude tag", TagFilterOptions{ExcludeTags: []string{"internal"}}, []string{"get", "patch", "head"}},
		{"exclude all", TagFilterOptions{ExcludeTags: []string{"public", "internal"}, MatchStrategy: MatchAll}, []string{"get", "delete", "patch", "head"}},
		{"include ids", TagFilterOptions{IncludeOperationIDs: []string{"listPets", "patchPet"}}, []string{"get", "patch"}},
		{"exclude ids", TagFilterOptions{ExcludeOperationIDs: []string{"listPets", "addPet"}}, []string{"delete", "patch", "head"}},
		{"intersection", TagFilterOptions{IncludeTags: []string{"public"}, IncludeOperationIDs: []string{"patchPet", "listPets"}}, []string{"get"}},
		{"exclusion wins", TagFilterOptions{IncludeOperationIDs: []string{"listPets"}, ExcludeTags: []string{"shared"}}, nil},
		{"id exclusion wins", TagFilterOptions{IncludeOperationIDs: []string{"listPets"}, ExcludeOperationIDs: []string{"listPets"}}, nil},
		{"case sensitive", TagFilterOptions{IncludeOperationIDs: []string{"ListPets"}}, nil},
		{"duplicate tags", TagFilterOptions{IncludeTags: []string{"public", "public"}, MatchStrategy: MatchAll}, []string{"get", "post"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := parseTestYAML(t, source)
			stats, _, err := Transform(root, "3.1.0", Options{TagFilter: tc.options})
			require.NoError(t, err)
			require.Equal(t, len(tc.want), stats.OperationsKept)
			paths := mapValue(documentRoot(root), "paths")
			item := mapValue(paths, "/pets")
			if len(tc.want) == 0 {
				require.Nil(t, item)
				return
			}
			require.Len(t, item.Content, len(tc.want)*2)
			for _, method := range tc.want {
				require.NotNil(t, mapValue(item, method), method)
			}
		})
	}
}

func TestOperationIDFiltersTraverseReferencesAndCallbacks(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.2.0
paths:
  /ref: {$ref: '#/components/pathItems/Pets'}
webhooks:
  event:
    post: {operationId: webhook}
components:
  pathItems:
    Pets:
      get:
        operationId: listPets
        callbacks:
          notify: {$ref: '#/components/callbacks/Notify'}
      additionalOperations:
        CUSTOM: {operationId: custom}
  callbacks:
    Notify:
      '{$request.body#/url}':
        post: {operationId: callback}
`)
	stats, _, err := Transform(root, "3.2.0", Options{TagFilter: TagFilterOptions{ExcludeOperationIDs: []string{"callback", "webhook", "custom"}}, PruneUnused: true})
	require.NoError(t, err)
	require.Equal(t, 1, stats.OperationsKept)
	require.Equal(t, 3, stats.OperationsRemoved)
	doc := documentRoot(root)
	require.Empty(t, mapValue(doc, "webhooks").Content)
	components := mapValue(doc, "components")
	require.Nil(t, mapValue(components, "callbacks"))
	item := mapValue(mapValue(components, "pathItems"), "Pets")
	require.Nil(t, mapValue(item, "additionalOperations"))
	require.Nil(t, mapValue(mapValue(item, "get"), "callbacks"))
}
