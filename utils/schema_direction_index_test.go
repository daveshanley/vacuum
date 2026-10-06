package utils

import (
	"testing"

	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/testify/require"
)

func TestGetSchemaDirections(t *testing.T) {
	const spec = `openapi: 3.1.0
info: {title: Direction index, version: "1"}
paths:
  /items:
    post:
      parameters:
        - name: filter
          in: query
          schema: {$ref: '#/components/schemas/RequestOnly'}
      requestBody:
        content:
          application/json:
            schema: {$ref: '#/components/schemas/RequestWrapper'}
      responses:
        '200':
          description: OK
          headers:
            X-Result:
              schema: {$ref: '#/components/schemas/ResponseOnly'}
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Shared'}
components:
  schemas:
    RequestWrapper:
      type: object
      properties:
        shared: {$ref: '#/components/schemas/Shared'}
    RequestOnly: {type: string}
    ResponseOnly: {type: string}
    Shared: {type: string}
    Unused: {type: string}
`
	doc, err := libopenapi.NewDocument([]byte(spec))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(doc.Release)
	m, err := doc.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}
	directions := GetSchemaDirections(&m.Model)
	for name, want := range map[string]DirectionType{
		"RequestWrapper": DirectionRequest,
		"RequestOnly":    DirectionRequest,
		"ResponseOnly":   DirectionResponse,
		"Shared":         DirectionBoth,
		"Unused":         DirectionNone,
	} {
		got := directions[name]
		if got == "" {
			got = DirectionNone
		}
		if got != want {
			t.Errorf("%s: indexed direction = %s, want %s", name, got, want)
		}
		if got := GetSchemaDirection(&m.Model, name); got != want {
			t.Errorf("%s: single direction = %s, want %s", name, got, want)
		}
	}
}

func TestGetSchemaDirections_NestedReferences(t *testing.T) {
	for _, schema := range []struct {
		name string
		body string
	}{
		{"allOf", "allOf: [{$ref: '#/components/schemas/Leaf'}]"},
		{"anyOf", "anyOf: [{$ref: '#/components/schemas/Leaf'}]"},
		{"oneOf", "oneOf: [{$ref: '#/components/schemas/Leaf'}]"},
		{"not", "not: {$ref: '#/components/schemas/Leaf'}"},
		{"properties", "properties: {value: {$ref: '#/components/schemas/Leaf'}}"},
		{"additionalProperties", "additionalProperties: {$ref: '#/components/schemas/Leaf'}"},
		{"patternProperties", "patternProperties: {'.*': {$ref: '#/components/schemas/Leaf'}}"},
		{"items", "items: {$ref: '#/components/schemas/Leaf'}"},
		{"prefixItems", "prefixItems: [{$ref: '#/components/schemas/Leaf'}]"},
		{"contains", "contains: {$ref: '#/components/schemas/Leaf'}"},
		{"if", "if: {$ref: '#/components/schemas/Leaf'}"},
		{"then", "then: {$ref: '#/components/schemas/Leaf'}"},
		{"else", "else: {$ref: '#/components/schemas/Leaf'}"},
		{"dependentSchemas", "dependentSchemas: {value: {$ref: '#/components/schemas/Leaf'}}"},
		{"propertyNames", "propertyNames: {$ref: '#/components/schemas/Leaf'}"},
		{"unevaluatedItems", "unevaluatedItems: {$ref: '#/components/schemas/Leaf'}"},
		{"unevaluatedProperties", "unevaluatedProperties: {$ref: '#/components/schemas/Leaf'}"},
	} {
		t.Run(schema.name, func(t *testing.T) {
			doc, err := libopenapi.NewDocument([]byte(`openapi: 3.1.0
info: {title: Nested references, version: "1"}
paths:
  /items:
    parameters:
      - name: filter
        in: query
        schema: {$ref: '#/components/schemas/Wrapper'}
    get:
      responses:
        default:
          description: OK
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Leaf'}
components:
  schemas:
    Wrapper:
      ` + schema.body + `
    Leaf: {type: string}
`))
			require.NoError(t, err)
			t.Cleanup(doc.Release)
			m, err := doc.BuildV3Model()
			require.NoError(t, err)
			require.Equal(t, map[string]DirectionType{"Wrapper": DirectionRequest, "Leaf": DirectionBoth}, GetSchemaDirections(&m.Model))
		})
	}
}

func TestGetSchemaDirections_CircularReferences(t *testing.T) {
	doc, err := libopenapi.NewDocument([]byte(`openapi: 3.1.0
info: {title: Circular references, version: "1"}
paths:
  /items:
    post:
      requestBody:
        content:
          application/json:
            schema: {$ref: '#/components/schemas/A'}
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema: {$ref: '#/components/schemas/B'}
components:
  schemas:
    A:
      type: object
      properties:
        child: {$ref: '#/components/schemas/B'}
    B:
      type: object
      properties:
        parent: {$ref: '#/components/schemas/A'}
`))
	require.NoError(t, err)
	t.Cleanup(doc.Release)
	m, err := doc.BuildV3Model()
	require.NoError(t, err)
	require.Equal(t, map[string]DirectionType{"A": DirectionBoth, "B": DirectionBoth}, GetSchemaDirections(&m.Model))
}

func TestGetSchemaDirections_EmptyDocument(t *testing.T) {
	for _, doc := range []*v3.Document{nil, {}, {Paths: &v3.Paths{}}} {
		require.Empty(t, GetSchemaDirections(doc))
		require.Equal(t, DirectionNone, GetSchemaDirection(doc, "missing"))
	}
}

func TestGetSchemaDirections_EmptyReferenceName(t *testing.T) {
	doc, err := libopenapi.NewDocument([]byte(`openapi: 3.1.0
info: {title: Empty schema name, version: "1"}
paths:
  /items:
    get:
      parameters:
        - name: value
          in: query
          schema: {$ref: '#/components/schemas/'}
components:
  schemas:
    '': {type: string}
`))
	require.NoError(t, err)
	t.Cleanup(doc.Release)
	m, err := doc.BuildV3Model()
	require.NoError(t, err)
	require.Empty(t, GetSchemaDirections(&m.Model))
	require.Equal(t, DirectionNone, GetSchemaDirection(&m.Model, ""))
	require.Equal(t, DirectionNone, GetSchemaDirection(&m.Model, "#/components/schemas/"))
}
