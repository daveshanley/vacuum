package cmd

import (
	"path/filepath"
	"testing"

	"github.com/pb33f/testify/require"
)

func requireSingleGeneratedFile(t *testing.T, pattern string) string {
	t.Helper()

	files, err := filepath.Glob(pattern)
	require.NoError(t, err)
	require.Len(t, files, 1)

	return files[0]
}
