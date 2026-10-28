package languageserver

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/testify/require"
)

func TestMultifileMalformedURLReturnsDiagnostic(t *testing.T) {
	root := "https://example.com/%zz"
	state := newMultifileTestServer(root, utils.HTTPClientConfig{})
	diagnostics := state.rootDiagnostics(context.Background())
	require.Len(t, diagnostics[root], 1)
	require.Contains(t, diagnostics[root][0].Message, "invalid URL escape")
}

func TestMultifileDocumentURIRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "API spec.yaml")
	require.Equal(t, path, fileURIToPath(documentURI(path)))
}
