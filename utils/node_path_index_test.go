package utils

import (
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
