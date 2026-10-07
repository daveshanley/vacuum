package utils

import (
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
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

func TestGetSchemaDirections_ProgrammaticCycle(t *testing.T) {
	// A cycle without source nodes must terminate, while parsed descendants
	// still receive their request direction.
	doc, err := libopenapi.NewDocument([]byte("openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\ncomponents:\n  schemas:\n    Leaf: {type: string}"))
	require.NoError(t, err)
	defer doc.Release()
	m, err := doc.BuildV3Model()
	require.NoError(t, err)
	leaf := m.Model.Components.Schemas.GetOrZero("Leaf")
	schema := &base.Schema{Type: []string{"object"}}
	proxy := base.CreateSchemaProxy(schema)
	schema.AllOf = []*base.SchemaProxy{proxy, leaf}
	paths := orderedmap.New[string, *v3.PathItem]()
	paths.Set("/items", &v3.PathItem{Parameters: []*v3.Parameter{{Name: "filter", In: "query", Schema: proxy}}})
	programmatic := &v3.Document{Paths: &v3.Paths{PathItems: paths}}
	require.Equal(t, DirectionNone, GetSchemaDirection(programmatic, "missing"))
	require.Equal(t, DirectionRequest, GetSchemaNodeDirections(programmatic)[leaf.Schema().GoLow().RootNode])
}

func TestGetSchemaDirections_RequestAndResponseEntryPoints(t *testing.T) {
	const schema = "{$ref: '#/components/schemas/S'}"
	content := "{application/json: {schema: " + schema + "}}"
	request := "{requestBody: {content: " + content + "}}"
	for _, tc := range []struct {
		name, paths, webhooks string
		want                  DirectionType
	}{
		{"parameter content", "{/items: {parameters: [{name: filter, in: query, content: " + content + "}]}}", "{}", DirectionRequest},
		{"header content", "{/items: {get: {responses: {'200': {description: OK, headers: {X-Value: {content: " + content + "}}}}}}}", "{}", DirectionResponse},
		{"callback", "{/items: {get: {callbacks: {event: {'{$request.body#/url}': {post: " + request + "}}}}}}", "{}", DirectionRequest},
		{"webhook", "{}", "{event: {post: " + request + "}}", DirectionRequest},
		{"query", "{/items: {query: " + request + "}}", "{}", DirectionRequest},
		{"additional operation", "{/items: {additionalOperations: {CUSTOM: " + request + "}}}", "{}", DirectionRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := libopenapi.NewDocument([]byte("openapi: 3.2.0\ninfo: {title: Entry points, version: '1'}\npaths: " + tc.paths + "\nwebhooks: " + tc.webhooks + "\ncomponents: {schemas: {S: {type: string}}}\n"))
			require.NoError(t, err)
			defer doc.Release()
			m, err := doc.BuildV3Model()
			require.NoError(t, err)
			require.Equal(t, tc.want, GetSchemaDirection(&m.Model, "S"))
			node := m.Model.Components.Schemas.GetOrZero("S").Schema().GoLow().RootNode
			require.Equal(t, tc.want, GetSchemaNodeDirections(&m.Model)[node])
		})
	}
}
