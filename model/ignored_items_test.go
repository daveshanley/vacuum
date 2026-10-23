package model

import (
	"github.com/pb33f/testify/require"
	"testing"
)

func TestIgnoredItemsMerge(t *testing.T) {
	var merged IgnoredItems
	source := IgnoredItems{"rule": {"one", "two"}}
	merged = merged.Merge(source).Merge(IgnoredItems{"rule": {"three"}, "other": {"four"}})
	require.Equal(t, IgnoredItems{"rule": {"one", "two", "three"}, "other": {"four"}}, merged)
	source["rule"][0] = "changed"
	require.Equal(t, "one", merged["rule"][0])
	require.Equal(t, merged, merged.Merge(nil))
	require.NotNil(t, IgnoredItems(nil).Merge(nil))
}
