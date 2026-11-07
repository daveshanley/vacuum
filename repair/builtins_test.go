// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package repair

import (
	"encoding/json"
	"github.com/pb33f/testify/require"
	"testing"
)

func TestBuiltinProviders(t *testing.T) {
	expected := map[string]string{
		"duplicated-entry-in-enum":               "enum: [1, '1', null]",
		"json-schema-enum-values-compatible":     "enum: [{a: 1}]",
		"openapi-tags":                           "tags: [{name: pet}]",
		"asyncapi-3-tags-uniqueness":             "tags: [{name: pet}]",
		"openapi-tags-alphabetical":              "tags: [{name: a}, {name: z}]",
		"asyncapi-3-tags-alphabetical":           "tags: [{name: a}, {name: z}]",
		"oas2-api-schemes":                       "schemes: [https]",
		"operation-parameters":                   "parameters: [{name: id, in: query}]",
		"arazzo-parameter":                       "parameters: [{name: id, value: 1}]",
		"path-params":                            "parameters: [{name: id, in: path, required: true}]",
		"operation-success-response":             "responses: {'200': {description: ok}}",
		"json-schema-dependent-required-defined": "dependentRequired: {id: [name]}",
		"oas-schema-check":                       "openapi: 3.1.0\nconst: 1",
		"no-unnecessary-combinator":              "schema: {type: string}",
		"duplicate-paths":                        "paths: {/pets: {get: {responses: {'200': {description: ok}}}}}",
	}

	for _, tc := range []struct{ rule, pointer, source string }{
		{"duplicated-entry-in-enum", "/enum", "enum: [1, 1.0, '1', null, null]\n"},
		{"json-schema-enum-values-compatible", "/enum/1", "enum: [{a: 1}, {a: 1}]\n"},
		{"openapi-tags", "/tags", "tags: [{name: pet}, {name: pet}]\n"},
		{"asyncapi-3-tags-uniqueness", "/tags/1/name", "tags: [{name: pet}, {name: pet}]\n"},
		{"openapi-tags-alphabetical", "/tags", "tags: [{name: z}, {name: a}]\n"},
		{"asyncapi-3-tags-alphabetical", "/tags", "tags: [{name: z}, {name: a}]\n"},
		{"oas2-api-schemes", "/schemes", "schemes: [https, https]\n"},
		{"operation-parameters", "/parameters/1", "parameters: [{name: id, in: query}, {name: id, in: query}]\n"},
		{"arazzo-parameter", "/parameters/1", "parameters: [{name: id, value: 1}, {name: id, value: 1}]\n"},
		{"path-params", "/parameters/0", "parameters: [{name: id, in: path, required: false}]\n"},
		{"operation-success-response", "/responses", "responses: {200: {description: ok}}\n"},
		{"json-schema-dependent-required-defined", "/dependentRequired", "dependentRequired: {id: [id, name]}\n"},
		{"oas-schema-check", "/enum", "openapi: 3.1.0\nenum: [1, 2]\nconst: 1\n"},
		{"no-unnecessary-combinator", "/schema/allOf", "schema: {allOf: [{type: string}]}\n"},
		{"duplicate-paths", "/paths/~1pets", "paths:\n  /pets: {get: {responses: {'200': {description: ok}}}}\n  /pets: {get: {responses: {'200': {description: ok}}}}\n"},
	} {
		t.Run(tc.rule, func(t *testing.T) {
			data := []byte(tc.source)
			plan, err := BuiltinPlan("s", data, []Finding{{RuleID: tc.rule, Pointer: tc.pointer}})
			require.NoError(t, err)
			require.NotEmpty(t, plan.Targets, "%+v", plan)
			candidate, err := Apply("s", data, plan)
			require.NoError(t, err)
			require.YAMLEq(t, expected[tc.rule], string(candidate))
		})
	}
}
func TestBuiltinRefusesAmbiguity(t *testing.T) {
	for _, tc := range []struct{ rule, pointer, source string }{
		{"asyncapi-3-tags-uniqueness", "/tags/1/name", "tags: [{name: pet, description: a}, {name: pet, description: b}]\n"},
		{"no-unnecessary-combinator", "/schema/allOf", "schema: {description: keep, allOf: [{type: string}]}\n"},
		{"duplicated-entry-in-enum", "/enum", "enum: [1, '1']\n"},
		{"oas-schema-check", "/enum", "openapi: 3.0.3\nenum: [1, 2]\nconst: 1\n"},
		{"duplicate-paths", "/paths/~1pets", "paths:\n  /pets: {description: first}\n  /pets: {description: second}\n"},
	} {
		t.Run(tc.rule, func(t *testing.T) {
			plan, err := BuiltinPlan("s", []byte(tc.source), []Finding{{RuleID: tc.rule, Pointer: tc.pointer}})
			require.NoError(t, err)
			require.Empty(t, plan.Targets)
			require.NotEmpty(t, plan.Declined)
		})
	}
}

func TestBuiltinRepairPreservesEnumKeyComment(t *testing.T) {
	data := []byte("openapi: 3.1.0\ncomponents:\n  schemas:\n    Pet:\n      const: a\n      # This comment documents the contract.\n      enum: [a]\n")
	plan, err := BuiltinPlan("api.yaml", data, []Finding{{RuleID: "oas-schema-check", Pointer: "/components/schemas/Pet/enum"}})
	require.NoError(t, err)
	require.Empty(t, plan.Operations)
	require.NotEmpty(t, plan.Declined)
}

func TestBuiltinPlanIsIndependentOfWorkerOrder(t *testing.T) {
	source := []byte("a: {enum: [1,1]}\nb: {enum: [2,2]}\n")
	findings := []Finding{{RuleID: "duplicated-entry-in-enum", Pointer: "/a/enum"}, {RuleID: "duplicated-entry-in-enum", Pointer: "/b/enum"}}
	a, err := BuiltinPlan("s", source, findings)
	require.NoError(t, err)
	findings[0], findings[1] = findings[1], findings[0]
	b, err := BuiltinPlan("s", source, findings)
	require.NoError(t, err)
	encodedA, err := json.Marshal(a)
	require.NoError(t, err)
	encodedB, err := json.Marshal(b)
	require.NoError(t, err)
	require.Equal(t, string(encodedA), string(encodedB))
}
