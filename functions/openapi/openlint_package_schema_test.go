// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package openapi

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/testify/require"
)

func TestOpenLintPackageUnusedComponents(t *testing.T) {
	const document = `openapi: 3.0.3
components:
  schemas:
    Used: {type: string}
    Orphan: {type: boolean}
  requestBodies:
    Payload: {content: {application/json: {schema: {$ref: '#/components/schemas/Used'}}}}
  securitySchemes:
    Auth: {type: http, scheme: basic}
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(document), &root))
	idx := index.NewSpecIndexWithConfig(&root, index.CreateClosedAPIIndexConfig())
	ctx := model.RuleFunctionContext{Index: idx, Rule: &model.Rule{Message: "Potentially unused component has been detected."}}
	result := OpenLintPackageFunction("oasUnusedComponent").RunRule(root.Content, ctx)
	require.Len(t, result, 2)
	require.Equal(t, "$.components.schemas.Orphan", result[0].Path)
	require.Equal(t, "$.components.requestBodies.Payload", result[1].Path)
	require.Equal(t, ctx.Rule.Message, result[0].Message)
	require.Equal(t, yaml.MappingNode, result[0].StartNode.Kind)
}

func TestOpenLintPackageDocumentSchema(t *testing.T) {
	for _, tc := range []struct {
		name, data string
		invalid    bool
	}{
		{"swagger", "swagger: '2.0'\ninfo: {title: API, version: '1'}\npaths: {}\n", false},
		{"oas3", "openapi: 3.0.3\ninfo: {title: API, version: '1'}\npaths: {}\n", false},
		{"oas31", "openapi: 3.1.0\ninfo: {title: API, version: '1'}\npaths: {}\n", false},
		{"oas32", "openapi: 3.2.0\ninfo: {title: API, version: '1'}\npaths: {}\n", false},
		{"missing title", "openapi: 3.0.3\ninfo: {version: '1'}\npaths: {}\n", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var root yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(tc.data), &root))
			result := OpenLintPackageFunction("oasDocumentSchema").RunRule(root.Content, model.RuleFunctionContext{Rule: &model.Rule{}})
			if !tc.invalid {
				require.Empty(t, result)
				return
			}
			require.Len(t, result, 1)
			require.Equal(t, "$.info", result[0].Path)
			require.Equal(t, `"info" property must have required property "title"`, result[0].Message)
		})
	}
}

func TestOpenLintPackageExample(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(`schema: {type: integer}
examples:
  good: {value: 2}
  bad: {value: two}
  external: {externalValue: 'https://example.com/value'}
`), &root))
	ctx := model.RuleFunctionContext{Rule: &model.Rule{}, RuleAction: &model.RuleAction{}, Contract: model.OpenLintV1Draft, Given: "$", ExecutionRoot: &root, Options: map[string]any{"oasVersion": float64(3), "schemaField": "schema", "type": "media"}}
	result := OpenLintPackageFunction("oasExample").RunRule(root.Content, ctx)
	require.Len(t, result, 1)
	require.Equal(t, "two", result[0].StartNode.Value)
}

func TestOpenLintPackagedMediaAccessAndNestedExamplePaths(t *testing.T) {
	const data = `schema:
  type: object
  required: [id, secret, age]
  properties:
    id: {type: string, readOnly: true}
    secret: {type: string, writeOnly: true}
    age: {type: integer}
example: `
	for _, tc := range []struct {
		path, example string
		invalid       bool
	}{
		{"$.components.requestBodies.Create.content.json", "{secret: hi, age: 1}", false},
		{"$.paths['/pets'].get.responses['200'].content.json", "{id: hello, age: 1}", false},
		{"$.components.responses.Pet.content.json", "{id: hello, age: invalid}", true},
	} {
		t.Run(tc.path, func(t *testing.T) {
			var root yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(data+tc.example), &root))
			ctx := model.RuleFunctionContext{Rule: &model.Rule{}, RuleAction: &model.RuleAction{}, Given: tc.path, ExecutionRoot: &root, Options: map[string]any{"oasVersion": float64(3), "schemaField": "schema", "type": "media"}}
			got := OpenLintPackageFunction("oasExample").RunRule(root.Content, ctx)
			if !tc.invalid {
				require.Empty(t, got)
				return
			}
			require.Len(t, got, 1)
			require.Equal(t, tc.path+".example.age", got[0].Path)
			require.Equal(t, `"age" property type must be integer`, got[0].Message)
			require.Equal(t, "invalid", got[0].StartNode.Value)
		})
	}
}

// YAML-authored rulesets decode oasVersion as an int, JSON ones as a float64.
func TestOpenLintPackageExampleSwaggerVersionFromYAMLOptions(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(`{type: integer, x-example: two}`), &root))
	for _, version := range []any{2, float64(2)} {
		ctx := model.RuleFunctionContext{Rule: &model.Rule{}, RuleAction: &model.RuleAction{}, Contract: model.OpenLintV1Draft, Given: "$", ExecutionRoot: &root, Options: map[string]any{"oasVersion": version, "schemaField": "$", "type": "schema"}}
		result := OpenLintPackageFunction("oasExample").RunRule(root.Content, ctx)
		require.Len(t, result, 1, "oasVersion %T", version)
		require.Equal(t, "two", result[0].StartNode.Value)
	}
}

func TestOpenLintPackageSchemaUsesCoreOnSelectedValue(t *testing.T) {
	const path = "$.components.schemas.Value.enum"
	for _, tc := range []struct{ input, want string }{
		{"[{2: x}, {3: y}]", ""},
		{"[{2: x}, {2: x}]", `"enum" property must not have duplicate items (items ## 0 and 1 are identical)`},
		{"[1, 2, 1, 2]", `"enum" property must not have duplicate items (items ## 1 and 3 are identical)`},
		{"[2, 2, 0, -0.0]", `"enum" property must not have duplicate items (items ## 2 and 3 are identical)`},
		{"[2, 2, {a: [0]}, {a: [-0.0]}]", `"enum" property must not have duplicate items (items ## 2 and 3 are identical)`},
	} {
		var input yaml.Node
		require.NoError(t, yaml.Unmarshal([]byte(tc.input), &input))
		ctx := model.RuleFunctionContext{Given: path, Contract: model.OpenLintV1Draft, Rule: &model.Rule{}, RuleAction: &model.RuleAction{}, Options: map[string]any{"schema": map[string]any{"type": "array", "uniqueItems": true}}}
		got := OpenLintPackageFunction("oasSchema").RunRule(input.Content, ctx)
		if tc.want == "" {
			require.Empty(t, got)
			continue
		}
		require.Len(t, got, 1)
		require.Equal(t, path, got[0].Path)
		require.Same(t, input.Content[0], got[0].StartNode)
		require.Equal(t, tc.want, got[0].Message)
	}
}
