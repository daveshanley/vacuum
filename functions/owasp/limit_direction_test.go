// Copyright 2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
	"fmt"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/testify/require"
)

func TestLimitRules_DirectionIsScopedToEachDocument(t *testing.T) {
	for _, ruleCase := range []struct {
		name       string
		schemaType string
		function   model.RuleFunction
	}{
		{name: "array", schemaType: "array\n      items: {type: integer}", function: ArrayLimit{}},
		{name: "string", schemaType: "string", function: StringLimit{}},
	} {
		t.Run(ruleCase.name, func(t *testing.T) {
			// Reuse the rule and schema name across documents with different usage.
			for _, usage := range []struct {
				name        string
				requestRef  string
				responseRef string
				wantResult  bool
			}{
				{name: "request", requestRef: "thing", responseRef: "bounded", wantResult: true},
				{name: "response", requestRef: "bounded", responseRef: "thing"},
				{name: "both", requestRef: "thing", responseRef: "thing", wantResult: true},
				{name: "unused", requestRef: "bounded", responseRef: "bounded"},
			} {
				t.Run(usage.name, func(t *testing.T) {
					requestWrapper, responseWrapper := "bounded", "bounded"
					if usage.requestRef == "thing" {
						requestWrapper = "wrapper"
					}
					if usage.responseRef == "thing" {
						responseWrapper = "wrapper"
					}
					spec := fmt.Sprintf(`openapi: 3.1.0
info: {title: Limits, version: "1"}
paths:
  /items:
    post:
      requestBody:
        content:
          application/json:
            schema: {$ref: '#/components/schemas/%s'}
          application/yaml:
            schema: {$ref: '#/components/schemas/%s'}
          application/cbor:
            schema: {$ref: '#/components/schemas/%s'}
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema: {$ref: '#/components/schemas/%s'}
            application/cbor:
              schema: {$ref: '#/components/schemas/%s'}
components:
  schemas:
    thing:
      type: %s
    wrapper:
      type: object
      properties:
        thing:
          type: %s
    bounded:
      type: string
      maxLength: 10
`, usage.requestRef, usage.requestRef, requestWrapper, usage.responseRef, responseWrapper, ruleCase.schemaType, strings.ReplaceAll(ruleCase.schemaType, "\n      ", "\n          "))
					document, err := libopenapi.NewDocument([]byte(spec))
					require.NoError(t, err)
					m, errs := document.BuildV3Model()
					require.Empty(t, errs)
					drDocument := drModel.NewDrDocument(m)
					t.Cleanup(drDocument.Release)
					rule := buildOpenApiTestRuleAction("$", ruleCase.name+"_limit", "", nil)
					ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
					ctx.Document, ctx.DrDocument, ctx.Rule = document, drDocument, &rule

					candidates := 0
					for _, schema := range drDocument.Schemas {
						if schema.Name == "thing" {
							candidates++
						}
					}
					require.Equal(t, 2, candidates, "fixture must exercise repeated candidate names")
					results := ruleCase.function.RunRule(nil, ctx)
					if !usage.wantResult {
						require.Empty(t, results)
						return
					}
					require.Len(t, results, 2)
					wantPaths := []string{"$.components.schemas['thing']", "$.components.schemas['wrapper'].properties['thing']"}
					wantAliases := [][]string{
						{wantPaths[0], "$.paths['/items'].post.requestBody.content['application/json'].schema", "$.paths['/items'].post.requestBody.content['application/yaml'].schema"},
						{wantPaths[1], "$.paths['/items'].post.requestBody.content['application/cbor'].schema.properties['thing']"},
					}
					if usage.name == "both" {
						wantAliases[0] = append(wantAliases[0], "$.paths['/items'].post.responses['200'].content['application/json'].schema")
						wantAliases[1] = append(wantAliases[1], "$.paths['/items'].post.responses['200'].content['application/cbor'].schema.properties['thing']")
					}
					for i, result := range results {
						require.Equal(t, wantPaths[i], result.Path)
						require.NotNil(t, result.StartNode)
						require.Equal(t, "type", result.StartNode.Value)
						require.Equal(t, result.StartNode.Line, result.EndNode.Line)
						require.Same(t, &rule, result.Rule)
						require.Equal(t, wantAliases[i], result.Paths)
					}

				})
			}
		})
	}
}

func TestLimitRules_EmptyDocument(t *testing.T) {
	for _, rule := range []model.RuleFunction{ArrayLimit{}, StringLimit{}} {
		require.Empty(t, rule.RunRule(nil, model.RuleFunctionContext{}))
		require.Empty(t, rule.RunRule(nil, model.RuleFunctionContext{DrDocument: &drModel.DrDocument{}}))
	}
}

// Issue 953: inline children have no reference name, and equal property names
// in response-only or unused schemas must not inherit request usage.
func TestLimitRules_NestedRequestSchemaIdentity(t *testing.T) {
	const spec = `openapi: 3.1.0
info: {title: Nested limits, version: "1"}
paths:
  /widgets:
    post:
      requestBody:
        content:
          application/json:
            schema: {$ref: '#/components/schemas/Widget'}
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Response'}
components:
  schemas:
    Widget:
      type: object
      properties:
        nested:
          type: object
          properties:
            tags: {type: array, items: {type: string}}
            name: {type: string}
            bounded: {type: string, maxLength: 20}
            parent: {$ref: '#/components/schemas/Widget'}
    Response:
      type: object
      properties:
        tags: {type: array, items: {type: string}}
        name: {type: string}
    Unused:
      type: object
      properties:
        tags: {type: array, items: {type: string}}
        name: {type: string}
`
	document, err := libopenapi.NewDocument([]byte(spec))
	require.NoError(t, err)
	t.Cleanup(document.Release)
	m, err := document.BuildV3Model()
	require.NoError(t, err)
	drDocument := drModel.NewDrDocument(m)
	t.Cleanup(drDocument.Release)
	for _, tc := range []struct {
		name     string
		function model.RuleFunction
		suffixes []string
	}{
		{"array", ArrayLimit{}, []string{".properties['tags']"}},
		{"string", StringLimit{}, []string{".properties['tags'].items", ".properties['name']"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rule := buildOpenApiTestRuleAction("$", tc.name+"_limit", "", nil)
			ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
			ctx.Document, ctx.DrDocument, ctx.Rule = document, drDocument, &rule
			results := tc.function.RunRule(nil, ctx)
			var paths []string
			for _, result := range results {
				paths = append(paths, result.Path)
				require.Greater(t, result.StartNode.Line, 0)
				require.Equal(t, "type", result.StartNode.Value)
			}
			var expected []string
			for _, suffix := range tc.suffixes {
				expected = append(expected, "$.components.schemas['Widget'].properties['nested']"+suffix)
			}
			require.ElementsMatch(t, expected, paths)
		})
	}
}
