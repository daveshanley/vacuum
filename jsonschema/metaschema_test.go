// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package jsonschema

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/pb33f/go-yaml"
	schemaengine "github.com/pb33f/jsonschema/v6"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestCompileSchemaValidatesLocalReferencesAcrossDialects(t *testing.T) {
	for _, dialect := range []string{SchemaURL07, SchemaURL2019, SchemaURL2020} {
		t.Run(dialect, func(t *testing.T) {
			var root yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(fmt.Sprintf(`$schema: %s
type: object
properties:
  count:
    $ref: '#/definitions/count'
required: [count]
definitions:
  count:
    type: integer
    minimum: 1
`, dialect)), &root))

			compiled, err := CompileSchema(&root)
			require.NoError(t, err)
			require.NoError(t, compiled.Validate(map[string]any{"count": 2}))
			for _, invalid := range []map[string]any{{}, {"count": 0}, {"count": "two"}} {
				err := compiled.Validate(invalid)
				var validationErr *schemaengine.ValidationError
				require.ErrorAs(t, err, &validationErr)
			}
		})
	}
}

func TestCompileSchemaRejectsExternalReferenceWithoutRequest(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte(`{"type":"string"}`))
	}))
	defer server.Close()

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("$ref: "+server.URL+"/schema.json\n"), &root))
	_, err := CompileSchema(&root)
	require.ErrorContains(t, err, "remote schema loading is disabled")
	assert.Zero(t, requests.Load())
}
