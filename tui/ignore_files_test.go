package tui

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestWatchIgnoreFiles(t *testing.T) {
	dir := t.TempDir()
	first, second := filepath.Join(dir, "one.yaml"), filepath.Join(dir, "two.yaml")
	require.NoError(t, os.WriteFile(first, []byte("rule: [one]\n"), 0600))
	require.NoError(t, os.WriteFile(second, []byte("rule: [two]\n"), 0600))
	config := &WatchConfig{IgnoreFile: first, IgnoreFiles: []string{second}}
	merged, err := loadWatchIgnoreFiles(config)
	require.NoError(t, err)
	require.Equal(t, model.IgnoredItems{"rule": {"one", "two"}}, merged)
	require.NoError(t, os.WriteFile(second, []byte("rule: [three]\n"), 0600))
	merged, err = loadWatchIgnoreFiles(config)
	require.NoError(t, err)
	require.Equal(t, model.IgnoredItems{"rule": {"one", "three"}}, merged)
	require.NoError(t, os.WriteFile(second, []byte("rule: ["), 0600))
	merged, err = loadWatchIgnoreFiles(config)
	require.ErrorContains(t, err, second)
	require.Nil(t, merged)
	require.NoError(t, os.Remove(second))
	_, err = loadWatchIgnoreFiles(config)
	require.ErrorContains(t, err, second)
}
