package utils

import (
	"github.com/pb33f/jsonpath/pkg/jsonpath"
	"github.com/pb33f/testify/require"
	"testing"
)

func TestResultPathPreservesLiteralKeys(t *testing.T) {
	for _, key := range []string{"", "a'b", `a\b`, "line\nbreak", "a.b", `a"b`, "a/b~c"} {
		path := AppendResultPathSegment("$", key)
		parsed, err := jsonpath.NewPath(path)
		require.NoError(t, err, path)
		parts, err := parsed.GetSegmentInfo()
		require.NoError(t, err)
		require.Len(t, parts, 1)
		require.Equal(t, key, parts[0].Key)
	}
}
