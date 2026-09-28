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
