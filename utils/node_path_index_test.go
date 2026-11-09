package utils

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"strings"
	"sync"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

func TestNodePathIndexUniqueLocations(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("first: &shared {name: value}\nsecond: *shared\n"), &root))
	mapping := root.Content[0]
	index := BuildNodePathIndex(&root)
	for i, want := range []string{"$.first", "$.second"} {
		for _, node := range mapping.Content[2*i : 2*i+2] {
			path, ok := index.LookupUnique(node)
			require.True(t, ok)
			require.Equal(t, want, path)
		}
	}
	// Resolved references reuse the mapping and all its descendants.
	shared := mapping.Content[1]
	mapping.Content[3] = shared
	index = BuildNodePathIndex(&root)
	for _, node := range append([]*yaml.Node{shared}, shared.Content...) {
		_, ok := index.LookupUnique(node)
		require.False(t, ok)
	}
	path, ok := index.Lookup(shared)
	require.True(t, ok)
	require.Equal(t, "$.second", path)
	_, ok = index.LookupUnique(nil)
	require.False(t, ok)
	_, ok = (*NodePathIndex)(nil).LookupUnique(shared)
	require.False(t, ok)

	sequence := &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{shared, shared}}
	index = BuildNodePathIndex(sequence)
	_, ok = index.LookupUnique(shared)
	require.False(t, ok)
}

func TestCompactNodePathIndexMatchesFullIndex(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("items:\n  - a.b: {value: text}\n    'quote\"key': [one, two]\n  - name: other\n"), &root))
	// Compare the compact representation before testing shared-node fallback.
	compactTree := buildNodePathIndexForContract(&root, model.OpenLintV1Draft)
	require.NotNil(t, compactTree.positions)
	fullTree := BuildNodePathIndex(&root)
	for node, want := range fullTree.paths {
		got, ok := compactTree.Lookup(node)
		require.True(t, ok)
		require.Equal(t, want, got)
	}
	// Resolved references can place the same node at several paths.
	shared := root.Content[0].Content[1].Content[0].Content[1]
	root.Content[0].Content = append(root.Content[0].Content, &yaml.Node{Kind: yaml.ScalarNode, Value: "shared"}, shared)
	full := BuildNodePathIndex(&root)
	compact := buildNodePathIndexForContract(&root, model.OpenLintV1Draft)
	require.Nil(t, compact.positions, "shared graphs must not retain expanded path segments")
	var check func(*yaml.Node)
	check = func(node *yaml.Node) {
		want, wantOK := full.Lookup(node)
		got, gotOK := compact.Lookup(node)
		require.Equal(t, want, got)
		require.Equal(t, wantOK, gotOK)
		_, wantUnique := full.LookupUnique(node)
		_, gotUnique := compact.LookupUnique(node)
		require.Equal(t, wantUnique, gotUnique)
		for _, child := range node.Content {
			check(child)
		}
	}
	check(&root)
	_, ok := compact.Lookup(&yaml.Node{})
	require.False(t, ok)
	_, ok = compact.Lookup(nil)
	require.False(t, ok)
	cache := new(sync.Map)
	ctx := model.RuleFunctionContext{Contract: model.OpenLintV1Draft, SchemaPathCache: cache}
	require.Same(t, NodePathIndexForContext(ctx, &root), NodePathIndexForContext(ctx, &root))
	ctx.Contract = model.NativeContract
	require.Nil(t, NodePathIndexForContext(ctx, &root).positions)
}

func BenchmarkNodePathIndex(b *testing.B) {
	var root yaml.Node
	var source strings.Builder
	for i := 0; i < 4096; i++ {
		fmt.Fprintf(&source, "item%d: {name: value, description: text}\n", i)
	}
	require.NoError(b, yaml.Unmarshal([]byte(source.String()), &root))
	for _, contract := range []model.Contract{model.NativeContract, model.OpenLintV1Draft} {
		b.Run(fmt.Sprint(contract), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				index := buildNodePathIndexForContract(&root, contract)
				for j := 1; j < len(root.Content[0].Content); j += 2 {
					_, ok := index.Lookup(root.Content[0].Content[j].Content[3])
					if !ok {
						b.Fatal("missing path")
					}
				}
			}
		})
	}
}

func TestCompactNodePathIndexDoesNotExposeComplexKeyChildren(t *testing.T) {
	keyChild := &yaml.Node{Kind: yaml.ScalarNode, Value: "key"}
	key := &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{keyChild}}
	root := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{key, {Kind: yaml.ScalarNode, Value: "value"}}}
	for _, contract := range []model.Contract{model.NativeContract, model.OpenLintV1Draft} {
		index := buildNodePathIndexForContract(root, contract)
		_, ok := index.Lookup(keyChild)
		require.False(t, ok)
		_, ok = index.Lookup(key)
		require.True(t, ok)
	}
}
