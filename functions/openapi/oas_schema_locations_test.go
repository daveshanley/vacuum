// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/testify/require"
)

const issue1034Spec = `openapi: 3.0.3
info: {title: Repro, version: 1.0.0}
paths: {}
components:
  schemas:
    Order:
      type: object
      properties:
        token:
          oneOf:
            - type: string
            - type: 'null'
`

func runSchemaLocationRule(t *testing.T, spec string) []model.RuleFunctionResult {
	t.Helper()
	doc, err := libopenapi.NewDocument([]byte(spec))
	require.NoError(t, err)
	info := doc.GetSpecInfo()
	rule := &model.Rule{Id: "oas3-schema"}
	return (OASSchema{}).RunRule([]*yaml.Node{info.RootNode}, model.RuleFunctionContext{SpecInfo: info, Document: doc, Rule: rule})
}

func TestOASSchema_Issue1034(t *testing.T) {
	results := runSchemaLocationRule(t, issue1034Spec)
	require.Len(t, results, 1)
	result := results[0]
	require.Equal(t, "$.components.schemas.Order.properties.token.oneOf[1].type", result.Path)
	require.Equal(t, []string{result.Path}, result.Paths)
	require.Equal(t, 12, result.StartNode.Line)
	require.Equal(t, 21, result.StartNode.Column)
	require.Equal(t, "null", result.StartNode.Value)
	require.Contains(t, result.Message, "value must be one of")
}

func TestOASSchema_Issue1034DistinctErrors(t *testing.T) {
	spec := issue1034Spec + "        other:\n          type: 'null'\n          minLength: -1\n"
	results := runSchemaLocationRule(t, spec)
	require.Len(t, results, 3)
	paths := make([]string, 0, len(results))
	for _, result := range results {
		paths = append(paths, result.Path)
	}
	require.ElementsMatch(t, []string{
		"$.components.schemas.Order.properties.token.oneOf[1].type",
		"$.components.schemas.Order.properties.other.type",
		"$.components.schemas.Order.properties.other.minLength",
	}, paths)
}

func TestOASSchema_Issue1034ValidTypes(t *testing.T) {
	for _, version := range []string{"3.0.3", "3.1.1"} {
		t.Run(version, func(t *testing.T) {
			spec := strings.Replace(issue1034Spec, "3.0.3", version, 1)
			if version == "3.0.3" {
				spec = strings.Replace(spec, "type: 'null'", "type: number", 1)
			}
			require.Empty(t, runSchemaLocationRule(t, spec))
		})
	}
}

func TestOASSchema_InstancePathsWithSpecialKeys(t *testing.T) {
	for _, key := range []string{"a/b~c", "0", "a.b", "a'b", `a\b`} {
		t.Run(key, func(t *testing.T) {
			spec := strings.Replace(issue1034Spec, "        token:", fmt.Sprintf("        '%s':", strings.ReplaceAll(key, "'", "''")), 1)
			result := runSchemaLocationRule(t, spec)
			require.Len(t, result, 1)
			paths := map[string]string{
				"a/b~c": "$.components.schemas.Order.properties['a/b~c'].oneOf[1].type",
				"0":     "$.components.schemas.Order.properties['0'].oneOf[1].type",
				"a.b":   "$.components.schemas.Order.properties['a.b'].oneOf[1].type",
				"a'b":   "$.components.schemas.Order.properties['a\\'b'].oneOf[1].type",
				`a\b`:   `$.components.schemas.Order.properties['a\\b'].oneOf[1].type`,
			}
			require.Equal(t, paths[key], result[0].Path)
			require.Equal(t, 12, result[0].StartNode.Line)
		})
	}
}

func TestOASSchema_DeeplyNestedFailure(t *testing.T) {
	const depth = 64
	var spec strings.Builder
	spec.WriteString("openapi: 3.0.3\ninfo: {title: Repro, version: '1'}\npaths: {}\ncomponents:\n  schemas:\n    Order:\n")
	for level := range depth {
		spec.WriteString(strings.Repeat(" ", 6+level*2) + "items:\n")
	}
	spec.WriteString(strings.Repeat(" ", 6+depth*2) + "type: 'null'\n")
	results := runSchemaLocationRule(t, spec.String())
	require.Len(t, results, 1)
	require.Equal(t, "$.components.schemas.Order"+strings.Repeat(".items", depth)+".type", results[0].Path)
	require.Equal(t, 7+depth, results[0].StartNode.Line)
}

func TestOASSchema_AliasAndMergeSourceLocations(t *testing.T) {
	for _, schema := range []string{"*bad", "\n      <<: *bad", "\n      <<: [*bad, *valid]", "\n      <<: *valid\n      type: 'null'"} {
		t.Run(schema, func(t *testing.T) {
			spec := "openapi: 3.0.3\ninfo: {title: Repro, version: '1'}\npaths: {}\nx-bad: &bad\n  type: 'null'\nx-valid: &valid\n  type: string\ncomponents:\n  schemas:\n    Order: " + schema + "\n"
			results := runSchemaLocationRule(t, spec)
			require.Len(t, results, 1)
			require.Equal(t, "$.components.schemas.Order.type", results[0].Path)
			line, column := 5, 9
			if strings.Contains(schema, "type: 'null'") {
				line, column = 12, 13
			}
			require.Equal(t, line, results[0].StartNode.Line)
			require.Equal(t, column, results[0].StartNode.Column)
			require.Equal(t, "null", results[0].StartNode.Value)
		})
	}
}

func TestOASSchema_AlternativeNoiseDoesNotHideIndependentErrors(t *testing.T) {
	spec := issue1034Spec + "  securitySchemes:\n    auth:\n      type: http\n"
	results := runSchemaLocationRule(t, spec)
	foundType, foundScheme := false, false
	for _, result := range results {
		require.NotEqual(t, "$.components.securitySchemes.auth.type", result.Path, "http is a valid security scheme type")
		if result.Path == "$.components.schemas.Order.properties.token.oneOf[1].type" {
			foundType = true
		}
		if result.Path == "$.components.securitySchemes.auth" && strings.Contains(result.Message, "scheme") {
			foundScheme = true
		}
	}
	require.True(t, foundType, "the independent invalid schema type must remain visible")
	require.True(t, foundScheme, "the missing HTTP scheme must remain visible")
}

func TestOASSchema_RootFailureDoesNotHideNestedEnum(t *testing.T) {
	spec := strings.Replace(issue1034Spec, "info: {title: Repro, version: 1.0.0}\n", "", 1)
	results := runSchemaLocationRule(t, spec)
	require.Len(t, results, 2)
	paths := []string{results[0].Path, results[1].Path}
	require.ElementsMatch(t, []string{"$", "$.components.schemas.Order.properties.token.oneOf[1].type"}, paths)
}
