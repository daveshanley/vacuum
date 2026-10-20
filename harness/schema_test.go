package harness

import (
	"encoding/json"
	schemautil "github.com/daveshanley/vacuum/jsonschema"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
	"testing"
)

func TestPublishedSchemasAcceptContracts(t *testing.T) {
	s, path := testSuite(t)
	for name, v := range map[string]any{"suite": s, "result": Run(s, path, RunOptions{})} {
		raw, err := Schema(name)
		require.NoError(t, err)
		var root yaml.Node
		require.NoError(t, yaml.Unmarshal(raw, &root))
		compiled, err := schemautil.CompileSchema(&root)
		require.NoError(t, err)
		data, err := json.Marshal(v)
		require.NoError(t, err)
		var value any
		require.NoError(t, json.Unmarshal(data, &value))
		require.NoError(t, compiled.Validate(value), string(data))
	}
	data, err := json.Marshal(s)
	require.NoError(t, err)
	_, err = Parse(data)
	require.NoError(t, err)
}
