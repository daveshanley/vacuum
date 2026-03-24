// Copyright 2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package rulesets

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAliases_SimpleArray(t *testing.T) {
	raw := map[string]interface{}{
		"PathItem": []interface{}{"$.paths[*]"},
	}
	parsed, err := ParseAliases(raw)
	require.NoError(t, err)
	require.Contains(t, parsed, "PathItem")
	assert.Equal(t, SimpleAlias{"$.paths[*]"}, parsed["PathItem"].Simple)
	assert.Nil(t, parsed["PathItem"].Targeted)
}

func TestParseAliases_SimpleString(t *testing.T) {
	raw := map[string]interface{}{
		"PathItem": "$.paths[*]",
	}
	parsed, err := ParseAliases(raw)
	require.NoError(t, err)
	require.Contains(t, parsed, "PathItem")
	assert.Equal(t, SimpleAlias{"$.paths[*]"}, parsed["PathItem"].Simple)
}

func TestParseAliases_Targeted(t *testing.T) {
	raw := map[string]interface{}{
		"SchemaObject": map[string]interface{}{
			"description": "Schema objects",
			"targets": []interface{}{
				map[string]interface{}{
					"formats": []interface{}{"oas3"},
					"given":   []interface{}{"$.components.schemas[*]"},
				},
				map[string]interface{}{
					"formats": []interface{}{"oas2"},
					"given":   "$.definitions[*]",
				},
			},
		},
	}
	parsed, err := ParseAliases(raw)
	require.NoError(t, err)
	require.Contains(t, parsed, "SchemaObject")
	pa := parsed["SchemaObject"]
	assert.Nil(t, pa.Simple)
	require.NotNil(t, pa.Targeted)
	assert.Equal(t, "Schema objects", pa.Targeted.Description)
	require.Len(t, pa.Targeted.Targets, 2)
	assert.Equal(t, []string{"oas3"}, pa.Targeted.Targets[0].Formats)
	assert.Equal(t, []string{"$.components.schemas[*]"}, pa.Targeted.Targets[0].Given)
	assert.Equal(t, []string{"oas2"}, pa.Targeted.Targets[1].Formats)
	assert.Equal(t, []string{"$.definitions[*]"}, pa.Targeted.Targets[1].Given)
}

func TestResolveAliasesForFormat_Match(t *testing.T) {
	parsed := map[string]*ParsedAlias{
		"Simple": {Simple: SimpleAlias{"$.paths[*]"}},
		"Targeted": {Targeted: &TargetedAlias{
			Targets: []AliasTarget{
				{Formats: []string{"oas3"}, Given: []string{"$.components.schemas[*]"}},
				{Formats: []string{"oas2"}, Given: []string{"$.definitions[*]"}},
			},
		}},
	}
	resolved := ResolveAliasesForFormat(parsed, "oas3")
	assert.Equal(t, []string{"$.paths[*]"}, resolved["Simple"])
	assert.Equal(t, []string{"$.components.schemas[*]"}, resolved["Targeted"])
}

func TestResolveAliasesForFormat_NoMatch(t *testing.T) {
	parsed := map[string]*ParsedAlias{
		"Targeted": {Targeted: &TargetedAlias{
			Targets: []AliasTarget{
				{Formats: []string{"oas2"}, Given: []string{"$.definitions[*]"}},
			},
		}},
	}
	resolved := ResolveAliasesForFormat(parsed, "oas3")
	assert.Empty(t, resolved["Targeted"])
}

func TestResolveAliasesForFormat_MultipleTargets(t *testing.T) {
	parsed := map[string]*ParsedAlias{
		"AllSchemas": {Targeted: &TargetedAlias{
			Targets: []AliasTarget{
				{Formats: []string{"oas3"}, Given: []string{"$.components.schemas[*]", "$.paths..schema"}},
				{Formats: []string{"oas3"}, Given: []string{"$.paths..requestBody.content..schema"}},
			},
		}},
	}
	resolved := ResolveAliasesForFormat(parsed, "oas3")
	assert.Len(t, resolved["AllSchemas"], 3)
}

func TestExpandAliasReferences_NoRefs(t *testing.T) {
	aliases := map[string][]string{
		"PathItem": {"$.paths[*]"},
	}
	result, err := ExpandAliasReferences(aliases)
	require.NoError(t, err)
	assert.Equal(t, []string{"$.paths[*]"}, result["PathItem"])
}

func TestExpandAliasReferences_SingleLevel(t *testing.T) {
	aliases := map[string][]string{
		"PathItem":   {"$.paths[*]"},
		"Operations": {"#PathItem[get,post,put,delete]"},
	}
	result, err := ExpandAliasReferences(aliases)
	require.NoError(t, err)
	assert.Equal(t, []string{"$.paths[*][get,post,put,delete]"}, result["Operations"])
}

func TestExpandAliasReferences_Nested(t *testing.T) {
	aliases := map[string][]string{
		"PathItem":   {"$.paths[*]"},
		"Operations": {"#PathItem[get,post]"},
		"Responses":  {"#Operations.responses"},
	}
	result, err := ExpandAliasReferences(aliases)
	require.NoError(t, err)
	assert.Equal(t, []string{"$.paths[*][get,post].responses"}, result["Responses"])
}

func TestExpandAliasReferences_Circular(t *testing.T) {
	aliases := map[string][]string{
		"A": {"#B.foo"},
		"B": {"#A.bar"},
	}
	_, err := ExpandAliasReferences(aliases)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circular")
}

func TestExpandAliasReferences_DotSuffix(t *testing.T) {
	aliases := map[string][]string{
		"Root":     {"$.root"},
		"Children": {"#Root.children"},
	}
	result, err := ExpandAliasReferences(aliases)
	require.NoError(t, err)
	assert.Equal(t, []string{"$.root.children"}, result["Children"])
}

func TestExpandAliasReferences_BracketSuffix(t *testing.T) {
	aliases := map[string][]string{
		"PathItem":   {"$.paths[*]"},
		"Operations": {"#PathItem[*]"},
	}
	result, err := ExpandAliasReferences(aliases)
	require.NoError(t, err)
	assert.Equal(t, []string{"$.paths[*][*]"}, result["Operations"])
}

func TestExpandAliasReferences_RecursiveSuffix(t *testing.T) {
	aliases := map[string][]string{
		"PathItem":  {"$.paths[*]"},
		"AllFields": {"#PathItem..properties"},
	}
	result, err := ExpandAliasReferences(aliases)
	require.NoError(t, err)
	assert.Equal(t, []string{"$.paths[*]..properties"}, result["AllFields"])
}

func TestExpandAliasReferences_SimpleRefsTargeted(t *testing.T) {
	// Simple alias references a targeted alias (cross-type dependency works naturally)
	aliases := map[string][]string{
		"SchemaObject":   {"$.components.schemas[*]"},
		"SchemaProperty": {"#SchemaObject.properties[*]"},
	}
	result, err := ExpandAliasReferences(aliases)
	require.NoError(t, err)
	assert.Equal(t, []string{"$.components.schemas[*].properties[*]"}, result["SchemaProperty"])
}

func TestExpandRuleGivenPaths_Simple(t *testing.T) {
	aliases := map[string][]string{
		"PathItem": {"$.paths[*]"},
	}
	paths := []string{"#PathItem"}
	result, err := ExpandRuleGivenPaths(paths, aliases)
	require.NoError(t, err)
	assert.Equal(t, []string{"$.paths[*]"}, result)
}

func TestExpandRuleGivenPaths_Mixed(t *testing.T) {
	aliases := map[string][]string{
		"PathItem": {"$.paths[*]"},
	}
	paths := []string{"$.info", "#PathItem[get]"}
	result, err := ExpandRuleGivenPaths(paths, aliases)
	require.NoError(t, err)
	assert.Equal(t, []string{"$.info", "$.paths[*][get]"}, result)
}

func TestExpandRuleGivenPaths_UnknownAlias(t *testing.T) {
	aliases := map[string][]string{}
	paths := []string{"#Unknown"}
	_, err := ExpandRuleGivenPaths(paths, aliases)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unknown")
}

func TestExpandRuleGivenPaths_NoRefs(t *testing.T) {
	aliases := map[string][]string{
		"PathItem": {"$.paths[*]"},
	}
	paths := []string{"$.info", "$.paths"}
	result, err := ExpandRuleGivenPaths(paths, aliases)
	require.NoError(t, err)
	// Should return the exact same slice (zero allocation)
	assert.Equal(t, paths, result)
}

func TestCreateRuleSetFromData_WithAliases(t *testing.T) {
	yamlData := []byte(`
extends: [[spectral:oas, off]]
aliases:
  PathItem:
    - "$.paths[*]"
  Operations:
    - "#PathItem[get,post]"
rules:
  test-rule:
    given: "#Operations"
    severity: warn
    then:
      function: truthy
      field: operationId
`)
	rs, err := CreateRuleSetFromData(yamlData)
	require.NoError(t, err)
	require.NotNil(t, rs)
	assert.NotNil(t, rs.Aliases)
	assert.Contains(t, rs.Aliases, "PathItem")
	assert.Contains(t, rs.Aliases, "Operations")
}

func TestRuleSetReuse_DifferentFormats(t *testing.T) {
	// Verify that using the same ParsedAliases with different spec formats
	// does not cause stale mutations
	parsed := map[string]*ParsedAlias{
		"SchemaObject": {Targeted: &TargetedAlias{
			Targets: []AliasTarget{
				{Formats: []string{"oas3"}, Given: []string{"$.components.schemas[*]"}},
				{Formats: []string{"oas2"}, Given: []string{"$.definitions[*]"}},
			},
		}},
	}

	// Resolve for oas3
	oas3Result := ResolveAliasesForFormat(parsed, "oas3")
	assert.Equal(t, []string{"$.components.schemas[*]"}, oas3Result["SchemaObject"])

	// Resolve for oas2 - should get different result, no stale oas3 paths
	oas2Result := ResolveAliasesForFormat(parsed, "oas2")
	assert.Equal(t, []string{"$.definitions[*]"}, oas2Result["SchemaObject"])

	// Verify oas3 result is unchanged (not mutated)
	assert.Equal(t, []string{"$.components.schemas[*]"}, oas3Result["SchemaObject"])
}

func TestGenerateRuleSet_AliasesReattachedAfterExtends(t *testing.T) {
	// When extends replaces rs entirely, aliases should survive
	rs := &RuleSet{
		Extends: []interface{}{[]interface{}{"spectral:oas", "off"}},
		Aliases: map[string]interface{}{
			"PathItem": []interface{}{"$.paths[*]"},
		},
		RuleDefinitions: map[string]interface{}{},
	}

	rsm := BuildDefaultRuleSets()
	result := rsm.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.NotNil(t, result.Aliases)
	assert.Contains(t, result.Aliases, "PathItem")
}

func TestExpandRuleGivenPaths_MultipleAliasPaths(t *testing.T) {
	// When an alias expands to multiple paths, each gets the suffix
	aliases := map[string][]string{
		"AllSchemas": {"$.components.schemas[*]", "$.paths..schema"},
	}
	paths := []string{"#AllSchemas.properties"}
	result, err := ExpandRuleGivenPaths(paths, aliases)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"$.components.schemas[*].properties",
		"$.paths..schema.properties",
	}, result)
}
