package transform

import (
	"testing"

	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestGraphDefensiveAndErrorPaths(t *testing.T) {
	_, err := BuildComponentGraph(&yaml.Node{}, "3.0")
	assert.Error(t, err)
	_, err = BuildComponentGraph(parseTestYAML(t, "openapi: 3.0.0\npaths: {}\nx: {$ref: other.yaml}\n"), "3.0")
	assert.Error(t, err)

	empty := parseTestYAML(t, "openapi: 3.0.0\npaths: {}\n")
	graph, err := BuildComponentGraph(empty, "3.0")
	require.NoError(t, err)
	assert.Empty(t, graph.components)

	g := &ComponentGraph{
		components: map[ComponentID]*yaml.Node{},
		adjacency:  map[ComponentID]map[ComponentID]struct{}{},
		roots:      map[ComponentID]struct{}{},
		locations:  map[ComponentID][]string{},
	}
	err = g.walkNonComponents(documentRoot(parseTestYAML(t, "openapi: 3.0.0\npaths: {}\ncomponents:\n  x-future: {$ref: '#/components/schemas/Missing'}\n")), nil, false, nil)
	assert.Error(t, err)
	err = g.walkOwned(documentRoot(parseTestYAML(t, "security: [{missing: []}]\n")), []string{"paths", "/x", "get"}, nil, false, nil)
	assert.Error(t, err)
	err = g.walkOwned(documentRoot(parseTestYAML(t, "discriminator: {mapping: {x: '#/components/schemas/Missing'}}\n")), nil, nil, false, nil)
	assert.Error(t, err)
	err = g.walkOwned(documentRoot(parseTestYAML(t, "- {$ref: '#/components/schemas/Missing'}\n")), nil, nil, false, nil)
	assert.Error(t, err)

	schemaOwner := ComponentID{Section: "schemas", Name: "A"}
	g.components[schemaOwner] = &yaml.Node{}
	id, err := g.resolveReference("#", "$recursiveRef", &schemaOwner, false, nil)
	require.NoError(t, err)
	assert.Equal(t, schemaOwner, *id)
	id, err = g.resolveReference("#anchor", "$ref", nil, false, nil)
	require.NoError(t, err)
	assert.Nil(t, id)
	id, err = g.resolveReference("#", "$ref", nil, false, nil)
	require.NoError(t, err)
	assert.Nil(t, id)
	swaggerID := ComponentID{Section: "definitions", Name: "A"}
	g.components[swaggerID] = &yaml.Node{}
	id, err = g.resolveReference("#/definitions/A", "$ref", nil, true, nil)
	require.NoError(t, err)
	assert.Equal(t, swaggerID, *id)
	id, err = g.resolveReference("#/paths/~1x", "$ref", nil, true, nil)
	require.NoError(t, err)
	assert.Nil(t, id)

	assert.NoError(t, g.addDiscriminatorEdges(nil, nil, nil, false))
	assert.NoError(t, g.addDiscriminatorEdges(&yaml.Node{Kind: yaml.ScalarNode}, nil, nil, false))
	nonScalar := documentRoot(parseTestYAML(t, "mapping: {x: [bad]}\n"))
	assert.NoError(t, g.addDiscriminatorEdges(nonScalar, nil, nil, false))
	malformed := documentRoot(parseTestYAML(t, "mapping: {x: '#/definitions/Bad~2'}\n"))
	assert.Error(t, g.addDiscriminatorEdges(malformed, nil, nil, true))
	validSwagger := documentRoot(parseTestYAML(t, "mapping: {x: '#/definitions/A', y: A}\n"))
	assert.NoError(t, g.addDiscriminatorEdges(validSwagger, nil, nil, true))
	invalidLocal := documentRoot(parseTestYAML(t, "mapping: {x: '#/paths/~1x'}\n"))
	assert.Error(t, g.addDiscriminatorEdges(invalidLocal, nil, nil, false))
	missingMapping := documentRoot(parseTestYAML(t, "mapping: {x: Missing}\n"))
	assert.Error(t, g.addDiscriminatorEdges(missingMapping, nil, nil, false))
	noMapping := documentRoot(parseTestYAML(t, "propertyName: kind\n"))
	assert.NoError(t, g.addDiscriminatorEdges(noMapping, nil, nil, false))

	security := documentRoot(parseTestYAML(t, "security: [bad, {}]\n"))
	assert.NoError(t, g.addSecurityEdges(security, nil, false, nil))
	assert.False(t, isSecurityRequirementOwner(nil, nil))
	assert.False(t, isSecurityRequirementOwner([]string{"paths", "/x", "get"}, &ComponentID{Section: "schemas", Name: "A"}))
	assert.False(t, isSecurityRequirementOwner([]string{"components", "future", "get"}, nil))
	assert.False(t, isSecurityRequirementOwner([]string{"x-private", "get"}, nil))
	assert.False(t, isSecurityRequirementOwner([]string{"components", "callbacks", "C", "x-private", "get"}, &ComponentID{Section: "callbacks", Name: "C"}))
	assert.True(t, isSecurityRequirementOwner([]string{"paths", "/x", "get"}, nil))
	assert.True(t, isSecurityRequirementOwner([]string{"components", "pathItems", "A", "additionalOperations", "CONNECT"}, &ComponentID{Section: "pathItems", Name: "A"}))
	assert.False(t, isSecurityRequirementOwner([]string{"paths", "/x", "summary"}, nil))
	walkYAML(nil, func(*yaml.Node) { t.Fatal("must not run") })
}

func TestFilterDefensiveCallbackAndReferencePaths(t *testing.T) {
	webhook := parseTestYAML(t, "openapi: 3.1.0\npaths: {}\nwebhooks:\n  gone:\n    post: {tags: [internal]}\n")
	stats, err := FilterOperationsByTags(webhook, "3.1.0", TagFilterOptions{IncludeTags: []string{"public"}})
	require.NoError(t, err)
	assert.Equal(t, 1, stats.WebhooksRemoved)

	root := documentRoot(parseTestYAML(t, `openapi: 3.0.0
paths: {}
components:
  callbacks:
    A:
      $ref: '#/components/callbacks/B'
    B:
      $ref: '#/components/callbacks/A'
    Kept:
      x-note: ok
      '{$request.body#/url}':
        post:
          tags: [public]
    RefKept:
      $ref: '#/components/callbacks/Kept'
`))
	var statsValue FilterStats
	ctx := filterContext{
		root: root, version: "3.0.0", included: map[string]struct{}{"public": {}},
		strategy: MatchAny, processed: map[*yaml.Node]bool{}, processing: map[*yaml.Node]bool{},
		callbacks: map[*yaml.Node]bool{}, callbacking: map[*yaml.Node]bool{}, stats: &statsValue,
	}
	ctx.filterRootMap("missing", false)
	assert.False(t, ctx.filterPathItem(nil))
	assert.False(t, ctx.filterPathItem(&yaml.Node{Kind: yaml.ScalarNode}))
	assert.False(t, ctx.filterCallback(nil))
	assert.False(t, ctx.filterCallback(&yaml.Node{Kind: yaml.ScalarNode}))
	callbacks := mapValue(mapValue(root, "components"), "callbacks")
	a := mapValue(callbacks, "A")
	assert.False(t, ctx.filterCallback(a))
	assert.False(t, ctx.filterCallback(a))
	ctx.callbacking[a] = true
	assert.False(t, ctx.filterCallback(a))
	delete(ctx.callbacking, a)
	kept := mapValue(callbacks, "Kept")
	assert.True(t, ctx.filterCallback(kept))
	assert.True(t, ctx.filterCallback(kept))
	assert.True(t, ctx.filterCallback(mapValue(callbacks, "RefKept")))

	operation := documentRoot(parseTestYAML(t, "callbacks:\n  keep:\n    '{$request.body#/url}':\n      post: {tags: [public]}\n"))
	ctx.filterOperationCallbacks(operation)
	assert.NotNil(t, mapValue(operation, "callbacks"))
	assert.False(t, ctx.filterAdditionalOperations(&yaml.Node{Kind: yaml.ScalarNode}))
	removedAdditional := documentRoot(parseTestYAML(t, "additionalOperations:\n  PURGE: {tags: [internal]}\n"))
	ctx.version = "3.2.0"
	assert.False(t, ctx.filterPathItem(removedAdditional))
	assert.Nil(t, resolveLocalNode(root, "bad"))
	assert.Nil(t, resolveLocalNode(root, "#/components/pathItems/Missing"))
	assert.Nil(t, resolveLocalNode(root, "#"))

	ids, refs := snapshotOperationTargets(&yaml.Node{Kind: yaml.ScalarNode}, "3.0")
	assert.Empty(t, ids)
	assert.Empty(t, refs)
	snapshotRoot := documentRoot(parseTestYAML(t, `openapi: 3.0.0
paths:
  /bad: scalar
  /good:
    get:
      callbacks:
        malformed: scalar
        referenced:
          $ref: '#/components/callbacks/Reusable'
components:
  callbacks:
    Reusable:
      '{$request.body#/url}':
        post: {operationId: nested}
`))
	ids, refs = snapshotOperationTargets(snapshotRoot, "3.0")
	assert.Contains(t, ids, "nested")
	assert.NotEmpty(t, refs)
	assert.Empty(t, danglingLinkWarnings(nil, nil, nil, nil, nil))
	assert.Nil(t, localReferencePath("not-local"))
}

func TestPruneAndReferenceDefensivePaths(t *testing.T) {
	_, err := PruneUnusedComponents(&yaml.Node{}, "3.0")
	assert.Error(t, err)
	_, err = PruneUnusedComponents(parseTestYAML(t, "openapi: 3.0.0\npaths: {}\nx: {$ref: other.yaml}\n"), "3.0")
	assert.Error(t, err)
	stats, err := PruneUnusedComponents(parseTestYAML(t, "openapi: 3.0.0\npaths: {}\n"), "3.0")
	require.NoError(t, err)
	assert.Zero(t, stats.ComponentsSeen)
	pruneEntries(&yaml.Node{Kind: yaml.ScalarNode}, "schemas", nil)

	sequence := parseTestYAML(t, "openapi: 3.0.0\npaths: {}\nx:\n  - {$ref: other.yaml}\n")
	assert.Error(t, ValidateBundled(sequence))
	assert.NoError(t, validateDiscriminatorReferences(nil, nil))
	assert.NoError(t, validateDiscriminatorReferences(&yaml.Node{Kind: yaml.ScalarNode}, nil))
	assert.NoError(t, validateDiscriminatorReferences(documentRoot(parseTestYAML(t, "mapping: bad\n")), nil))
	assert.Equal(t, "$['a']", jsonPath([]string{"", "a"}))
	assert.Nil(t, componentOwnerFromPath([]string{"paths", "/x"}))
	owner := componentOwnerFromPath([]string{"components", "links", "L", "operationId"})
	require.NotNil(t, owner)
	assert.Equal(t, ComponentID{Section: "links", Name: "L"}, *owner)
	warnings := []Warning{{Message: "kept"}}
	assert.Equal(t, warnings, RetainWarningsForPrunedDocument(warnings, PruneStats{}))
	assert.Equal(t, warnings, RetainWarningsForPrunedDocument(warnings, PruneStats{removed: map[ComponentID]struct{}{{Section: "links", Name: "L"}: {}}}))
	assert.Empty(t, RetainWarningsForPrunedDocument(nil, PruneStats{removed: map[ComponentID]struct{}{{Section: "links", Name: "L"}: {}}}))
}
