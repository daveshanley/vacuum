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
	_, stats, err := Transform(root, "3.2.0", Options{PruneUnused: true, PruneUnusedTags: true})
	require.NoError(t, err)
	require.Equal(t, 2, stats.TagsRemoved)
	tags := mapValue(documentRoot(root), "tags")
	require.Len(t, tags.Content, 4)
	for i, name := range []string{"parent", "child", "callback", "webhook"} {
		require.Equal(t, name, mapValue(tags.Content[i], "name").Value)
	}
	require.Equal(t, "keep metadata", mapValue(tags.Content[1], "description").Value)
	_, again, err := Transform(root, "3.2.0", Options{PruneUnused: true, PruneUnusedTags: true})
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
	_, stats, err := Transform(root, "3.1.0", Options{PruneUnused: true, PruneUnusedTags: true})
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

func TestPruneTagsIgnoresExtensionDataAndUnsupportedWebhooks(t *testing.T) {
	for _, version := range []string{"3.0.3", "3.1.0"} {
		t.Run(version, func(t *testing.T) {
			root := parseTestYAML(t, "openapi: "+version+`
tags: [{name: extensionOnly}, {name: webhook}]
paths:
  x-example:
    get: {tags: [extensionOnly]}
webhooks:
  x-event:
    post: {tags: [webhook]}
`)
			_, stats, err := Transform(root, version, Options{PruneUnused: true, PruneUnusedTags: true})
			require.NoError(t, err)
			doc := documentRoot(root)
			extension := mapValue(mapValue(doc, "paths"), "x-example")
			require.Equal(t, "extensionOnly", mapValue(mapValue(extension, "get"), "tags").Content[0].Value)
			tags := mapValue(doc, "tags")
			if version == "3.0.3" {
				require.Equal(t, 2, stats.TagsRemoved)
				require.Empty(t, tags.Content)
			} else {
				require.Equal(t, 1, stats.TagsRemoved)
				require.Len(t, tags.Content, 1)
				require.Equal(t, "webhook", mapValue(tags.Content[0], "name").Value)
			}
		})
	}
}

func TestPruneComponentsPreservesTagAPIContract(t *testing.T) {
	for _, direct := range []bool{false, true} {
		root := parseTestYAML(t, "openapi: 3.1.0\ntags: [{name: unused}]\npaths: {}\ncomponents: {schemas: {Unused: {type: string}}}\n")
		if direct {
			_, err := PruneUnusedComponents(root, "3.1.0")
			require.NoError(t, err)
		} else {
			_, _, err := Transform(root, "3.1.0", Options{PruneUnused: true})
			require.NoError(t, err)
		}
		require.Len(t, mapValue(documentRoot(root), "tags").Content, 1)
		require.Nil(t, mapValue(documentRoot(root), "components"))
	}
}

func TestPruneTagsPreservesDocumentationAndUnrelatedAliases(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.2.0
tags:
  - {name: Pets}
  - {name: Overview, parent: Parent}
  - {name: Parent}
  - {name: Narrative, x-traitTag: true}
  - {name: UpperNarrative, x-traitTag: TRUE}
  - {name: unused}
x-tagGroups: [{name: Core, tags: [Pets, Overview]}]
x-operation: &op {tags: [Pets]}
paths:
  /pets:
    get: *op
components: {schemas: {Unused: {type: string}}}
`)
	_, stats, err := Transform(root, "3.2.0", Options{PruneUnusedTags: true})
	require.NoError(t, err)
	require.Equal(t, 1, stats.TagsRemoved)
	require.Len(t, mapValue(documentRoot(root), "tags").Content, 5)
	require.NotNil(t, mapValue(documentRoot(root), "components"))
	raw, err := yaml.Marshal(root)
	require.NoError(t, err)
	require.Contains(t, string(raw), "&op")
	require.Contains(t, string(raw), "*op")
}

func TestPruneTagsKeepsUsedAnchorAndAliasedTags(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.1.0
tags:
  - name: active
    description: &description Shared text
x-tags: &tags [active]
paths:
  /pets:
    get:
      description: *description
      tags: *tags
`)
	_, stats, err := Transform(root, "3.1.0", Options{PruneUnused: true, PruneUnusedTags: true})
	require.NoError(t, err)
	require.Zero(t, stats.TagsRemoved)
	raw, err := yaml.Marshal(root)
	require.NoError(t, err)
	require.Contains(t, string(raw), "&description")
	require.Contains(t, string(raw), "*tags")
}

func TestPruneTagsCountsAliasedAdditionalOperations(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.2.0
tags: [{name: active}]
x-ops: &ops {CUSTOM: {tags: [active]}}
paths:
  /pets:
    additionalOperations: *ops
`)
	_, stats, err := Transform(root, "3.2.0", Options{PruneUnusedTags: true})
	require.NoError(t, err)
	require.Zero(t, stats.TagsRemoved)
	require.Len(t, mapValue(documentRoot(root), "tags").Content, 1)
	raw, err := yaml.Marshal(root)
	require.NoError(t, err)
	require.Contains(t, string(raw), "*ops")
}
