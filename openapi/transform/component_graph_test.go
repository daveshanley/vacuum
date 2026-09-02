package transform

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestComponentGraphReachabilityAndCycles(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.2.0
paths:
  /x:
    get:
      tags: [public]
      security:
        - publicAuth: []
      responses:
        '200':
          $ref: '#/components/responses/Root'
x-conservative:
  $ref: '#/components/schemas/ExtensionRoot'
components:
  schemas:
    A:
      allOf:
        - $ref: '#/components/schemas/B'
        - anyOf:
            - $ref: '#/components/schemas/C'
    B:
      items:
        $ref: '#/components/schemas/D'
    C:
      oneOf:
        - $ref: '#/components/schemas/D'
    D:
      $ref: '#/components/schemas/D'
    OrphanSelf:
      $ref: '#/components/schemas/OrphanSelf'
    CycleOne:
      $ref: '#/components/schemas/CycleTwo'
    CycleTwo:
      $ref: '#/components/schemas/CycleOne'
    ExtensionRoot: {type: string}
    Disc:
      discriminator:
        propertyName: kind
        mapping:
          a: A
        defaultMapping: '#/components/schemas/B'
  responses:
    Root:
      description: ok
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/A'
  securitySchemes:
    publicAuth: {type: apiKey, in: header, name: key}
    privateAuth: {type: apiKey, in: header, name: private}
  examples:
    orphan: {value: 1}
  mediaTypes:
    orphanMedia: {schema: {type: string}}
  x-future:
    keep:
      $ref: '#/components/schemas/Disc'
`)
	graph, err := BuildComponentGraph(root, "3.2.0")
	require.NoError(t, err)
	unreachable := graph.Unreachable()
	assert.Contains(t, unreachable, ComponentID{Section: "schemas", Name: "OrphanSelf"})
	assert.Contains(t, unreachable, ComponentID{Section: "schemas", Name: "CycleOne"})
	assert.Contains(t, unreachable, ComponentID{Section: "schemas", Name: "CycleTwo"})
	assert.Contains(t, unreachable, ComponentID{Section: "securitySchemes", Name: "privateAuth"})
	assert.Contains(t, unreachable, ComponentID{Section: "examples", Name: "orphan"})
	assert.Contains(t, unreachable, ComponentID{Section: "mediaTypes", Name: "orphanMedia"})
	assert.NotContains(t, unreachable, ComponentID{Section: "schemas", Name: "A"})
	assert.NotContains(t, unreachable, ComponentID{Section: "schemas", Name: "D"})
	assert.NotContains(t, unreachable, ComponentID{Section: "schemas", Name: "Disc"})
	assert.NotContains(t, unreachable, ComponentID{Section: "securitySchemes", Name: "publicAuth"})
}

func TestComponentGraphPointersAnchorsAndErrors(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.1.0
paths:
  /x:
    get:
      tags: [x]
      responses:
        '200':
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Foo%20Bar'
components:
  schemas:
    Foo Bar:
      $anchor: rootAnchor
      properties:
        child:
          $dynamicRef: '#rootAnchor'
        nested:
          $ref: '#/components/schemas/Foo Bar/properties/child'
    Foo:
      type: string
    FooBar:
      type: string
    Foo/Bar:
      type: string
    Foo~Bar:
      type: string
`)
	graph, err := BuildComponentGraph(root, "3.1.0")
	require.NoError(t, err)
	unreachable := graph.Unreachable()
	assert.NotContains(t, unreachable, ComponentID{Section: "schemas", Name: "Foo Bar"})
	assert.Contains(t, unreachable, ComponentID{Section: "schemas", Name: "Foo"})
	assert.Contains(t, unreachable, ComponentID{Section: "schemas", Name: "FooBar"})

	tests := []struct {
		name string
		doc  string
		want string
	}{
		{"missing", "openapi: 3.0.0\npaths:\n  /x:\n    get:\n      responses:\n        '200': {$ref: '#/components/responses/Nope'}\ncomponents:\n  responses: {}\n", "missing component"},
		{"malformed pointer", "openapi: 3.0.0\npaths:\n  /x:\n    get:\n      responses:\n        '200': {$ref: '#/components/schemas/Bad~2Name'}\ncomponents:\n  schemas: {}\n", "invalid JSON Pointer"},
		{"invalid component shape", "openapi: 3.0.0\npaths:\n  /x:\n    get:\n      responses:\n        '200': {$ref: '#/components/schemas'}\ncomponents:\n  schemas: {}\n", "invalid local component reference"},
		{"ambiguous anchor", "openapi: 3.1.0\npaths: {}\ncomponents:\n  schemas:\n    A: {$dynamicAnchor: same, $dynamicRef: '#same'}\n    B: {$dynamicAnchor: same}\n", "cannot statically resolve"},
		{"unowned recursive", "openapi: 3.1.0\npaths: {}\nx-ref: {$recursiveRef: '#'}\ncomponents: {schemas: {A: {type: string}}}\n", "cannot statically resolve"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := BuildComponentGraph(parseTestYAML(t, tc.doc), "3.1.0")
			assert.ErrorContains(t, err, tc.want)
		})
	}
}

func TestComponentGraphSwaggerAndAllSections(t *testing.T) {
	root := parseTestYAML(t, `swagger: '2.0'
paths:
  /x:
    get:
      security: [{auth: []}]
      parameters:
        - {$ref: '#/parameters/P'}
      responses:
        '200': {$ref: '#/responses/R'}
definitions:
  Used: {type: string}
  Unused: {type: string}
parameters:
  P: {name: p, in: query, type: string, x-schema: {$ref: '#/definitions/Used'}}
responses:
  R: {description: ok}
securityDefinitions:
  auth: {type: apiKey, in: header, name: key}
  unusedAuth: {type: apiKey, in: header, name: other}
`)
	graph, err := BuildComponentGraph(root, "2.0")
	require.NoError(t, err)
	unreachable := graph.Unreachable()
	assert.Contains(t, unreachable, ComponentID{Section: "definitions", Name: "Unused"})
	assert.Contains(t, unreachable, ComponentID{Section: "securityDefinitions", Name: "unusedAuth"})
	assert.NotContains(t, unreachable, ComponentID{Section: "definitions", Name: "Used"})
	assert.NotContains(t, unreachable, ComponentID{Section: "parameters", Name: "P"})
	assert.NotContains(t, unreachable, ComponentID{Section: "responses", Name: "R"})
}

func TestPruneAllOpenAPIComponentSections(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.2.0
paths: {}
components:
  schemas: {unused: {type: string}}
  responses: {unused: {description: no}}
  parameters: {unused: {name: p, in: query}}
  examples: {unused: {value: no}}
  requestBodies: {unused: {content: {}}}
  headers: {unused: {schema: {type: string}}}
  securitySchemes: {unused: {type: apiKey, in: header, name: key}}
  links: {unused: {operationId: nowhere}}
  callbacks: {unused: {'{$request.body#/url}': {}}}
  pathItems: {unused: {description: no}}
  mediaTypes: {unused: {schema: {type: string}}}
`)
	stats, err := PruneUnusedComponents(root, "3.2.0")
	require.NoError(t, err)
	assert.Equal(t, 11, stats.ComponentsSeen)
	assert.Equal(t, 11, stats.ComponentsRemoved)
	require.Len(t, stats.RemovedBySection, 11)
	assert.Nil(t, mapValue(documentRoot(root), "components"))
}

func TestComponentGraphOperationReferencesAndStructuralSecurity(t *testing.T) {
	valid := parseTestYAML(t, `openapi: 3.0.0
paths:
  /x:
    get:
      operationId: getX
      responses:
        '200':
          description: ok
          links:
            self:
              operationRef: '#/paths/~1x/get'
x-example:
  get:
    security: [{notAScheme: []}]
components:
  examples:
    Example:
      value:
        security: [{alsoNotAScheme: []}]
`)
	_, err := BuildComponentGraph(valid, "3.0.0")
	require.NoError(t, err)

	for _, tc := range []struct {
		name    string
		ref     string
		wantErr string
	}{
		{"missing is a surviving dangling link", "#/paths/~1missing/get", ""},
		{"malformed", "#/paths/Bad~2/get", "invalid JSON Pointer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := parseTestYAML(t, "openapi: 3.0.0\npaths:\n  /x:\n    get:\n      responses:\n        '200':\n          description: ok\n          links:\n            bad: {operationRef: '"+tc.ref+"'}\n")
			_, err := BuildComponentGraph(root, "3.0.0")
			if tc.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.wantErr)
			}
		})
	}
}

func TestComponentGraphIgnoresExampleReferencesButPreservesLocalExtensionReferences(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.1.0
paths:
  /things:
    get:
      responses:
        '200':
          description: ok
          content:
            application/json:
              example:
                $ref: '#/components/schemas/ExampleOnly'
x-local-reference:
  $ref: '#/components/schemas/ExtensionRoot'
x-external-reference:
  $ref: https://example.com/extension-data
components:
  schemas:
    ExampleOnly: {type: string}
    ExtensionRoot: {type: string}
`)
	graph, err := BuildComponentGraph(root, "3.1.0")
	require.NoError(t, err)
	unreachable := graph.Unreachable()
	assert.Contains(t, unreachable, ComponentID{Section: "schemas", Name: "ExampleOnly"})
	assert.NotContains(t, unreachable, ComponentID{Section: "schemas", Name: "ExtensionRoot"})
}

func TestComponentGraphIgnoresOperationRefOutsideLinkObjects(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.0.0
paths:
  /x:
    get:
      x-config:
        links:
          operationRef: '#/Bad~2Pointer'
      responses: {}
components:
  schemas:
    HasLinksProperty:
      properties:
        links:
          operationRef: '#/Also~2NotALink'
`)
	_, err := BuildComponentGraph(root, "3.0.0")
	require.NoError(t, err)
}

func TestLinkObjectPathClassification(t *testing.T) {
	for _, tc := range []struct {
		path []string
		want bool
	}{
		{[]string{"components", "links", "Reusable"}, true},
		{[]string{"components", "responses", "Reusable", "links", "Link"}, true},
		{[]string{"paths", "/x", "get", "responses", "200", "links", "Link"}, true},
		{[]string{"paths", "/x", "additionalOperations", "COPY", "responses", "200", "links", "Link"}, true},
		{[]string{"x-config", "links", "Link"}, false},
		{[]string{"schemas", "responses", "links", "Link"}, false},
	} {
		assert.Equal(t, tc.want, isLinkObjectPath(tc.path))
	}
}

func TestPrunePreservesComponentsOwningReachableYAMLAliases(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.1.0
paths:
  /x:
    get:
      responses:
        '200':
          content:
            application/json:
              schema: {$ref: '#/components/schemas/B'}
components:
  schemas:
    A: &shared {type: string}
    B:
      allOf:
        - *shared
    Unused: &unused {type: integer}
`)
	stats, err := PruneUnusedComponents(root, "3.1.0")
	require.NoError(t, err)
	assert.Equal(t, 2, stats.ComponentsKept)
	assert.Equal(t, 1, stats.ComponentsRemoved)
	schemas := mapValue(mapValue(documentRoot(root), "components"), "schemas")
	assert.NotNil(t, mapValue(schemas, "A"))
	assert.NotNil(t, mapValue(schemas, "B"))
	assert.Nil(t, mapValue(schemas, "Unused"))
	output, err := yaml.Marshal(root)
	require.NoError(t, err)
	var reparsed yaml.Node
	require.NoError(t, yaml.Unmarshal(output, &reparsed))
}

func TestComponentGraphTreatsRootYAMLAliasAsReachabilityRoot(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.1.0
paths: {}
components:
  schemas:
    A: &shared {type: string}
x-schema: *shared
`)
	graph, err := BuildComponentGraph(root, "3.1.0")
	require.NoError(t, err)
	assert.NotContains(t, graph.Unreachable(), ComponentID{Section: "schemas", Name: "A"})
}

func TestPruneUnusedComponentsMutationIdempotenceAndAtomicity(t *testing.T) {
	root := parseTestYAML(t, `openapi: 3.0.3
paths:
  /x:
    get:
      responses:
        '200':
          content:
            application/json:
              schema: {$ref: '#/components/schemas/First'}
components:
  schemas:
    First: # retained comment
      $ref: '#/components/schemas/Third'
    Second:
      type: string
    Third:
      type: string
  responses:
    Gone: {description: unused}
  x-keep:
    value: true
`)
	stats, err := PruneUnusedComponents(root, "3.0.3")
	require.NoError(t, err)
	assert.Equal(t, 4, stats.ComponentsSeen)
	assert.Equal(t, 2, stats.ComponentsKept)
	assert.Equal(t, 2, stats.ComponentsRemoved)
	assert.Equal(t, 1, stats.RemovedBySection["schemas"])
	assert.Equal(t, 1, stats.RemovedBySection["responses"])
	components := mapValue(documentRoot(root), "components")
	schemas := mapValue(components, "schemas")
	require.Equal(t, 4, len(schemas.Content))
	assert.Equal(t, "First", schemas.Content[0].Value)
	assert.Equal(t, "Third", schemas.Content[2].Value)
	assert.NotNil(t, mapValue(components, "x-keep"))
	assert.Nil(t, mapValue(components, "responses"))

	again, err := PruneUnusedComponents(root, "3.0.3")
	require.NoError(t, err)
	assert.Equal(t, 0, again.ComponentsRemoved)

	bad := parseTestYAML(t, "openapi: 3.0.0\npaths:\n  /x:\n    get:\n      responses:\n        '200': {$ref: '#/components/responses/Missing'}\ncomponents:\n  schemas:\n    Gone: {type: string}\n")
	before, err := yaml.Marshal(bad)
	require.NoError(t, err)
	_, err = PruneUnusedComponents(bad, "3.0.0")
	require.Error(t, err)
	after, err := yaml.Marshal(bad)
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after))
}

func TestPruneRemovesEmptyContainersAndPreservesUnknown(t *testing.T) {
	openapi := parseTestYAML(t, "openapi: 3.0.0\npaths: {}\ncomponents:\n  schemas:\n    Gone: {type: string}\n")
	_, err := PruneUnusedComponents(openapi, "3.0.0")
	require.NoError(t, err)
	assert.Nil(t, mapValue(documentRoot(openapi), "components"))

	unknown := parseTestYAML(t, "openapi: 3.0.0\npaths: {}\ncomponents:\n  schemas:\n    Gone: {type: string}\n  future: {}\n")
	_, err = PruneUnusedComponents(unknown, "3.0.0")
	require.NoError(t, err)
	assert.NotNil(t, mapValue(documentRoot(unknown), "components"))

	futureMediaTypes := parseTestYAML(t, "openapi: 3.0.0\npaths: {}\ncomponents:\n  mediaTypes:\n    Future: {schema: {type: string}}\n")
	stats, err := PruneUnusedComponents(futureMediaTypes, "3.0.0")
	require.NoError(t, err)
	assert.Zero(t, stats.ComponentsSeen)
	assert.NotNil(t, mapValue(mapValue(documentRoot(futureMediaTypes), "components"), "mediaTypes"))

	swagger := parseTestYAML(t, "swagger: '2.0'\npaths: {}\ndefinitions:\n  Gone: {type: string}\nresponses:\n  Gone: {description: no}\n")
	_, err = PruneUnusedComponents(swagger, "2.0")
	require.NoError(t, err)
	assert.Nil(t, mapValue(documentRoot(swagger), "definitions"))
	assert.Nil(t, mapValue(documentRoot(swagger), "responses"))
}

func BenchmarkPruneComponentGraphs(b *testing.B) {
	for _, shape := range []string{"chain", "wide", "cycle"} {
		b.Run(shape, func(b *testing.B) {
			var source strings.Builder
			source.WriteString("openapi: 3.0.3\npaths: {}\n")
			if shape != "cycle" {
				source.WriteString("x-root: {$ref: '#/components/schemas/S0'}\n")
			}
			source.WriteString("components:\n  schemas:\n")
			for i := 0; i < 500; i++ {
				fmt.Fprintf(&source, "    S%d:\n", i)
				switch shape {
				case "chain":
					if i < 499 {
						fmt.Fprintf(&source, "      $ref: '#/components/schemas/S%d'\n", i+1)
					} else {
						source.WriteString("      type: string\n")
					}
				case "wide":
					if i == 0 {
						source.WriteString("      allOf:\n")
						for j := 1; j < 500; j++ {
							fmt.Fprintf(&source, "        - $ref: '#/components/schemas/S%d'\n", j)
						}
					} else {
						source.WriteString("      type: string\n")
					}
				case "cycle":
					fmt.Fprintf(&source, "      $ref: '#/components/schemas/S%d'\n", (i+1)%500)
				}
			}
			data := []byte(source.String())
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var root yaml.Node
				_ = yaml.Unmarshal(data, &root)
				_, _ = PruneUnusedComponents(&root, "3.0.3")
			}
		})
	}
}
