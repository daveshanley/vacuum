package transform

import (
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
	"testing"
)

func TestPruneUnusedTagsIssue995(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.2.0
tags:
  - {name: unused, description: discard}
  - {name: parent, summary: retain parent}
  - {name: child, parent: parent, description: keep metadata}
  - {name: callback}
  - {name: webhook}
  - {name: unreferenced}
paths:
  /ref: {$ref: '#/components/pathItems/Active'}
webhooks:
  event:
    post: {tags: [webhook]}
components:
  pathItems:
    Active:
      get:
        tags: [child]
        callbacks:
          event:
            '{$request.body#/url}':
              post: {tags: [callback]}
    Unused:
      get: {tags: [unreferenced]}
`)
	_, stats, err := Transform(root, "3.2.0", Options{PruneUnused: true})
	require.NoError(t, err)
	require.Equal(t, 2, stats.TagsRemoved)
	tags := mapValue(documentRoot(root), "tags")
	require.Len(t, tags.Content, 4)
	for i, name := range []string{"parent", "child", "callback", "webhook"} {
		require.Equal(t, name, mapValue(tags.Content[i], "name").Value)
	}
	require.Equal(t, "keep metadata", mapValue(tags.Content[1], "description").Value)
	_, again, err := Transform(root, "3.2.0", Options{PruneUnused: true})
	require.NoError(t, err)
	require.Zero(t, again.TagsRemoved)
}

func TestPruneTagsPreservesAliasUses(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.1.0
tags:
  - name: unused
    description: &description Shared text
  - name: active
x-tags: &tags [active]
paths:
  /pets:
    get:
      description: *description
      tags: *tags
`)
	_, stats, err := Transform(root, "3.1.0", Options{PruneUnused: true})
	require.NoError(t, err)
	require.Equal(t, 1, stats.TagsRemoved)
	raw, err := yaml.Marshal(root)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &decoded))
	require.Contains(t, string(raw), "Shared text")
	require.Equal(t, "active", mapValue(mapValue(documentRoot(root), "tags").Content[0], "name").Value)
}

func TestFilteringDoesNotPruneTagsWithoutFlag(t *testing.T) {
	root := parseTestYAML(t, "openapi: 3.0.3\ntags: [{name: public}, {name: unused}]\npaths:\n  /pets:\n    get: {tags: [public]}\n")
	_, _, err := Transform(root, "3.0.3", Options{TagFilter: TagFilterOptions{IncludeTags: []string{"public"}}})
	require.NoError(t, err)
	require.Len(t, mapValue(documentRoot(root), "tags").Content, 2)
}
