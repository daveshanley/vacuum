package utils

import (
	"testing"

	"github.com/pb33f/libopenapi"
)

func TestGetSchemaDirectionsMatchesIndividualLookup(t *testing.T) {
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
		if old := GetSchemaDirection(&m.Model, name); got != old {
			t.Errorf("%s: indexed direction = %s, individual lookup = %s", name, got, old)
		}
	}
}
