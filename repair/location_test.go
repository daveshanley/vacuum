// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package repair

import (
	"testing"

	"github.com/pb33f/testify/require"
)

func TestLocationsFindExactPreservesSourceOwnership(t *testing.T) {
	root, err := ParseDocument([]byte(`{"name":"same value"}`))
	require.NoError(t, err)
	other, err := ParseDocument([]byte(`{"name":"same value"}`))
	require.NoError(t, err)
	locations := IndexLocations(root)
	for _, node := range root.Content[0].Content {
		pointer, found := locations.FindExact(node)
		require.True(t, found)
		require.Equal(t, "/name", pointer)
	}
	for _, node := range other.Content[0].Content {
		_, found := locations.FindExact(node)
		require.False(t, found, "same parser position does not prove source ownership")
		_, found = locations.Find(node)
		require.True(t, found, "cross-parse lookup remains available when ownership is known")
	}
	_, found := locations.FindExact(nil)
	require.False(t, found)
}
