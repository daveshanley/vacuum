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

func TestCanonicalSchemaPathPreservesLiteralKeys(t *testing.T) {
	for _, path := range []string{"$['x.properties.foo']", "$['x.patternProperties.foo']", "$['x.components.schemas.foo']", `$.components.schemas["a'b"]`, "$.properties['a.b']"} {
		require.Equal(t, path, CanonicalSchemaPath(path))
	}
	require.Equal(t, "$.components.schemas['Thing'].properties['name']", CanonicalSchemaPath("$.components.schemas.Thing.properties.name"))
	require.Equal(t, "$.components.schemas['Thing'].allOf[0].patternProperties['name']", CanonicalSchemaPath("$.components.schemas.Thing.allOf[0].patternProperties.name"))
	require.Equal(t, "$.components.schemas.*", CanonicalSchemaPath("$.components.schemas.*"))
	require.Equal(t, "invalid[", CanonicalSchemaPath("invalid["))
}
