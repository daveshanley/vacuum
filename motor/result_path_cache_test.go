package motor

import (
	"fmt"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/testify/require"
)

func TestResultPathCacheRetainsOnlyResultLocations(t *testing.T) {
	const unrelatedFields = 1000
	root := &yaml.Node{Kind: yaml.MappingNode}
	for i := range unrelatedFields {
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprintf("field%d", i), Line: i + 1, Column: 1},
			&yaml.Node{Kind: yaml.ScalarNode, Value: "value", Line: i + 1, Column: 12})
	}
	target := root.Content[1]
	missing := &yaml.Node{Kind: yaml.ScalarNode, Line: unrelatedFields + 2, Column: 12}
	results := []model.RuleFunctionResult{{StartNode: target}, {StartNode: missing}}
	cache := newResultPathCache(root, "", results)
	require.Len(t, cache.nodePaths, 2)
	require.Len(t, cache.positionPaths, 2)
	require.Len(t, cache.precisePositionMap, 2)
	path, found := cache.lookupNodePath(target)
	require.True(t, found)
	require.Equal(t, "$", path)
	path, found = cache.lookupPrecisePositionPathForNode(target)
	require.True(t, found)
	require.Equal(t, "$.field0", path)
	_, found = cache.lookupNodePath(missing)
	require.False(t, found)
	_, found = cache.lookupPositionPathForNode(missing)
	require.False(t, found)
	_, found = cache.lookupPrecisePositionPathForNode(missing)
	require.False(t, found)
	_, found = cache.lookupNodePath(root.Content[3])
	require.False(t, found)
	empty := newResultPathCache(root, "", nil)
	require.Empty(t, empty.nodePaths)
	require.Empty(t, empty.positionPaths)
	require.Empty(t, empty.precisePositionMap)
}

func TestResultPathCacheSeedsOriginLocations(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("first:\n  value: one\nsecond:\n  value: two\n"), &root))
	first := root.Content[0].Content[1].Content[1]
	second := root.Content[0].Content[3].Content[1]
	tests := []struct {
		name   string
		result model.RuleFunctionResult
		want   string
	}{
		{"origin node", model.RuleFunctionResult{Origin: &index.NodeOrigin{Node: first}}, "$.first"},
		{"origin value node", model.RuleFunctionResult{Origin: &index.NodeOrigin{ValueNode: second}}, "$.second"},
		{"origin position", model.RuleFunctionResult{Origin: &index.NodeOrigin{Line: first.Line, Column: first.Column}}, "$.first"},
		{"origin value position", model.RuleFunctionResult{Origin: &index.NodeOrigin{LineValue: second.Line, ColumnValue: second.Column}}, "$.second"},
		{"copied start node", model.RuleFunctionResult{StartNode: &yaml.Node{Line: second.Line, Column: second.Column}}, "$.second"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := newResultPathCache(&root, "", []model.RuleFunctionResult{tt.result})
			path, found := cache.canonicalPathForResult(&tt.result)
			require.True(t, found)
			require.Equal(t, tt.want, path)
		})
	}
}

func TestResultPathCachePreservesSharedPositionPrecedence(t *testing.T) {
	// Synthetic nodes can share positions after reference resolution. The first
	// containing path wins, while the precise path must retain the longest one.
	outer := &yaml.Node{Kind: yaml.ScalarNode, Value: "outer", Line: 1, Column: 1}
	inner := &yaml.Node{Kind: yaml.ScalarNode, Value: "inner", Line: 1, Column: 1}
	value := &yaml.Node{Kind: yaml.ScalarNode, Value: "value", Line: 1, Column: 1}
	root := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{outer, {Kind: yaml.MappingNode, Content: []*yaml.Node{inner, value}}}}
	copied := &yaml.Node{Line: 1, Column: 1}
	cache := newResultPathCache(root, "", []model.RuleFunctionResult{{StartNode: copied}})
	path, found := cache.lookupPositionPathForNode(copied)
	require.True(t, found)
	require.Equal(t, "$", path)
	path, found = cache.lookupPrecisePositionPathForNode(copied)
	require.True(t, found)
	require.Equal(t, "$.outer.inner", path)
}
