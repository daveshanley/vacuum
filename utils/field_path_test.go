// Copyright 2024-2025 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestParseFieldPath_SimpleKey(t *testing.T) {
	segments, err := ParseFieldPath("name")
	require.NoError(t, err)
	require.Len(t, segments, 1)
	assert.Equal(t, SegmentKey, segments[0].Type)
	assert.Equal(t, "name", segments[0].Key)
}

func TestParseFieldPath_NestedPath(t *testing.T) {
	segments, err := ParseFieldPath("a.b.c")
	require.NoError(t, err)
	require.Len(t, segments, 3)
	assert.Equal(t, SegmentKey, segments[0].Type)
	assert.Equal(t, "a", segments[0].Key)
	assert.Equal(t, SegmentKey, segments[1].Type)
	assert.Equal(t, "b", segments[1].Key)
	assert.Equal(t, SegmentKey, segments[2].Type)
	assert.Equal(t, "c", segments[2].Key)
}

func TestParseFieldPath_EscapedDot(t *testing.T) {
	segments, err := ParseFieldPath(`some\.key`)
	require.NoError(t, err)
	require.Len(t, segments, 1)
	assert.Equal(t, SegmentKey, segments[0].Type)
	assert.Equal(t, "some.key", segments[0].Key)
}

func TestParseFieldPath_EscapedDotInMiddle(t *testing.T) {
	segments, err := ParseFieldPath(`a\.b.c`)
	require.NoError(t, err)
	require.Len(t, segments, 2)
	assert.Equal(t, "a.b", segments[0].Key)
	assert.Equal(t, "c", segments[1].Key)
}

func TestParseFieldPath_EscapedBackslash(t *testing.T) {
	segments, err := ParseFieldPath(`a\\b`)
	require.NoError(t, err)
	require.Len(t, segments, 1)
	assert.Equal(t, `a\b`, segments[0].Key)
}

func TestParseFieldPath_TrailingBackslash(t *testing.T) {
	segments, err := ParseFieldPath(`name\`)
	require.NoError(t, err)
	require.Len(t, segments, 1)
	assert.Equal(t, `name\`, segments[0].Key)
}

func TestParseFieldPath_OtherEscape(t *testing.T) {
	// \x where x is not . or \ should preserve the backslash
	segments, err := ParseFieldPath(`a\nb`)
	require.NoError(t, err)
	require.Len(t, segments, 1)
	assert.Equal(t, `a\nb`, segments[0].Key)
}

func TestParseFieldPath_NumericIndex(t *testing.T) {
	segments, err := ParseFieldPath("items[0]")
	require.NoError(t, err)
	require.Len(t, segments, 2)
	assert.Equal(t, SegmentKey, segments[0].Type)
	assert.Equal(t, "items", segments[0].Key)
	assert.Equal(t, SegmentArrayIndex, segments[1].Type)
	assert.Equal(t, 0, segments[1].Index)
}

func TestParseFieldPath_NumericIndexThenKey(t *testing.T) {
	segments, err := ParseFieldPath("items[0].type")
	require.NoError(t, err)
	require.Len(t, segments, 3)
	assert.Equal(t, SegmentKey, segments[0].Type)
	assert.Equal(t, "items", segments[0].Key)
	assert.Equal(t, SegmentArrayIndex, segments[1].Type)
	assert.Equal(t, 0, segments[1].Index)
	assert.Equal(t, SegmentKey, segments[2].Type)
	assert.Equal(t, "type", segments[2].Key)
}

func TestParseFieldPath_StringIndexSingleQuotes(t *testing.T) {
	segments, err := ParseFieldPath("responses['200']")
	require.NoError(t, err)
	require.Len(t, segments, 2)
	assert.Equal(t, SegmentKey, segments[0].Type)
	assert.Equal(t, "responses", segments[0].Key)
	assert.Equal(t, SegmentMapKey, segments[1].Type)
	assert.Equal(t, "200", segments[1].Key)
}

func TestParseFieldPath_StringIndexDoubleQuotes(t *testing.T) {
	segments, err := ParseFieldPath(`responses["200"]`)
	require.NoError(t, err)
	require.Len(t, segments, 2)
	assert.Equal(t, SegmentKey, segments[0].Type)
	assert.Equal(t, "responses", segments[0].Key)
	assert.Equal(t, SegmentMapKey, segments[1].Type)
	assert.Equal(t, "200", segments[1].Key)
}

func TestParseFieldPath_StringIndexWithSpecialChars(t *testing.T) {
	segments, err := ParseFieldPath("paths['/pet'].get")
	require.NoError(t, err)
	require.Len(t, segments, 3)
	assert.Equal(t, SegmentKey, segments[0].Type)
	assert.Equal(t, "paths", segments[0].Key)
	assert.Equal(t, SegmentMapKey, segments[1].Type)
	assert.Equal(t, "/pet", segments[1].Key)
	assert.Equal(t, SegmentKey, segments[2].Type)
	assert.Equal(t, "get", segments[2].Key)
}

func TestParseFieldPath_MultipleIndexes(t *testing.T) {
	segments, err := ParseFieldPath("a[0][1]")
	require.NoError(t, err)
	require.Len(t, segments, 3)
	assert.Equal(t, SegmentKey, segments[0].Type)
	assert.Equal(t, "a", segments[0].Key)
	assert.Equal(t, SegmentArrayIndex, segments[1].Type)
	assert.Equal(t, 0, segments[1].Index)
	assert.Equal(t, SegmentArrayIndex, segments[2].Type)
	assert.Equal(t, 1, segments[2].Index)
}

func TestParseFieldPath_MultipleIndexesThenKey(t *testing.T) {
	segments, err := ParseFieldPath("a[0][1].b")
	require.NoError(t, err)
	require.Len(t, segments, 4)
	assert.Equal(t, SegmentKey, segments[0].Type)
	assert.Equal(t, "a", segments[0].Key)
	assert.Equal(t, SegmentArrayIndex, segments[1].Type)
	assert.Equal(t, 0, segments[1].Index)
	assert.Equal(t, SegmentArrayIndex, segments[2].Type)
	assert.Equal(t, 1, segments[2].Index)
	assert.Equal(t, SegmentKey, segments[3].Type)
	assert.Equal(t, "b", segments[3].Key)
}

func TestParseFieldPath_EmptyPath(t *testing.T) {
	segments, err := ParseFieldPath("")
	require.NoError(t, err)
	assert.Nil(t, segments)
}

// Error cases

func TestParseFieldPath_Error_NegativeIndex(t *testing.T) {
	_, err := ParseFieldPath("items[-1]")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "negative")
}

func TestParseFieldPath_Error_UnclosedBracket(t *testing.T) {
	_, err := ParseFieldPath("items[0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unclosed")
}

func TestParseFieldPath_Error_MismatchedQuotes(t *testing.T) {
	_, err := ParseFieldPath(`items['key"]`)
	require.Error(t, err)
}

func TestParseFieldPath_Error_NonNumericWithoutQuotes(t *testing.T) {
	_, err := ParseFieldPath("items[abc]")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-numeric")
}

func TestParseFieldPath_Error_EmptySegment(t *testing.T) {
	_, err := ParseFieldPath("a..b")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty segment")
}

func TestParseFieldPath_Error_PathStartsWithIndex(t *testing.T) {
	_, err := ParseFieldPath("[0].name")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot start with an index")
}

func TestParseFieldPath_Error_QuoteInBracketString(t *testing.T) {
	// Single quote inside double-quoted string should error
	_, err := ParseFieldPath(`items["a'b"]`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "quotes cannot appear")
}

func TestParseFieldPath_Error_EmptyIndex(t *testing.T) {
	_, err := ParseFieldPath("items[]")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty index")
}

func TestParseFieldPath_Error_UnclosedQuote(t *testing.T) {
	_, err := ParseFieldPath("items['key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unclosed quote")
}

// FindFieldPath tests

func parseYAML(t *testing.T, yamlStr string) []*yaml.Node {
	var node yaml.Node
	err := yaml.Unmarshal([]byte(yamlStr), &node)
	require.NoError(t, err)
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return node.Content[0].Content
	}
	return node.Content
}

func TestFindFieldPath_SingleLevelField(t *testing.T) {
	yamlStr := `
name: test
age: 30
`
	nodes := parseYAML(t, yamlStr)
	result := FindFieldPath("name", nodes, FieldPathOptions{})
	assert.True(t, result.Found)
	assert.NotNil(t, result.KeyNode)
	assert.NotNil(t, result.ValueNode)
	assert.Equal(t, "test", result.ValueNode.Value)
}

func TestFindFieldPath_SingleLevelField_NotExists(t *testing.T) {
	yamlStr := `name: test`
	nodes := parseYAML(t, yamlStr)
	result := FindFieldPath("missing", nodes, FieldPathOptions{})
	assert.False(t, result.Found)
}

func TestFindFieldPath_TwoLevelPath(t *testing.T) {
	yamlStr := `
properties:
  data:
    type: string
`
	nodes := parseYAML(t, yamlStr)
	result := FindFieldPath("properties.data", nodes, FieldPathOptions{})
	assert.True(t, result.Found)
	assert.NotNil(t, result.KeyNode)
	assert.Equal(t, "data", result.KeyNode.Value)
}

func TestFindFieldPath_ThreeLevelPath(t *testing.T) {
	yamlStr := `
a:
  b:
    c: value
`
	nodes := parseYAML(t, yamlStr)
	result := FindFieldPath("a.b.c", nodes, FieldPathOptions{})
	assert.True(t, result.Found)
	assert.Equal(t, "value", result.ValueNode.Value)
}

func TestFindFieldPath_MiddleSegmentMissing(t *testing.T) {
	yamlStr := `
a:
  x: 1
`
	nodes := parseYAML(t, yamlStr)
	result := FindFieldPath("a.b.c", nodes, FieldPathOptions{})
	assert.False(t, result.Found)
}

func TestFindFieldPath_MiddleSegmentNotAMap(t *testing.T) {
	yamlStr := `
a:
  b: scalar
`
	nodes := parseYAML(t, yamlStr)
	result := FindFieldPath("a.b.c", nodes, FieldPathOptions{})
	assert.False(t, result.Found)
}

func TestFindFieldPath_EmptyFieldPath(t *testing.T) {
	yamlStr := `name: test`
	nodes := parseYAML(t, yamlStr)
	result := FindFieldPath("", nodes, FieldPathOptions{})
	assert.False(t, result.Found)
}

func TestFindFieldPath_ArrayIndex(t *testing.T) {
	yamlStr := `
items:
  - first
  - second
  - third
`
	nodes := parseYAML(t, yamlStr)
	result := FindFieldPath("items[1]", nodes, FieldPathOptions{})
	assert.True(t, result.Found)
	assert.Equal(t, "second", result.ValueNode.Value)
}

func TestFindFieldPath_ArrayIndexThenKey(t *testing.T) {
	yamlStr := `
items:
  - name: first
    type: a
  - name: second
    type: b
`
	nodes := parseYAML(t, yamlStr)
	result := FindFieldPath("items[1].name", nodes, FieldPathOptions{})
	assert.True(t, result.Found)
	assert.Equal(t, "second", result.ValueNode.Value)
}

func TestFindFieldPath_ArrayIndexOutOfBounds(t *testing.T) {
	yamlStr := `
items:
  - first
  - second
`
	nodes := parseYAML(t, yamlStr)
	result := FindFieldPath("items[5]", nodes, FieldPathOptions{})
	assert.False(t, result.Found)
}

func TestFindFieldPath_ArrayIndexOnNonArray(t *testing.T) {
	yamlStr := `
items:
  key: value
`
	nodes := parseYAML(t, yamlStr)
	result := FindFieldPath("items[0]", nodes, FieldPathOptions{})
	assert.False(t, result.Found)
}

func TestFindFieldPath_StringIndex(t *testing.T) {
	yamlStr := `
responses:
  '200':
    description: OK
  '404':
    description: Not Found
`
	nodes := parseYAML(t, yamlStr)
	result := FindFieldPath("responses['200']", nodes, FieldPathOptions{})
	assert.True(t, result.Found)
}

func TestFindFieldPath_StringIndexThenKey(t *testing.T) {
	yamlStr := `
responses:
  '200':
    description: OK
`
	nodes := parseYAML(t, yamlStr)
	result := FindFieldPath("responses['200'].description", nodes, FieldPathOptions{})
	assert.True(t, result.Found)
	assert.Equal(t, "OK", result.ValueNode.Value)
}

func TestFindFieldPath_RealWorldPattern_PropertiesData(t *testing.T) {
	yamlStr := `
schema:
  type: object
  properties:
    data:
      type: array
    pagination:
      type: object
`
	nodes := parseYAML(t, yamlStr)
	result := FindFieldPath("schema.properties.data", nodes, FieldPathOptions{})
	assert.True(t, result.Found)
	assert.Equal(t, "data", result.KeyNode.Value)
}

func TestFindFieldPath_RecursiveFirstSegment(t *testing.T) {
	// This tests that RecursiveFirstSegment option works
	// FindKeyNode searches recursively into children, while FindKeyNodeTop only looks at top level
	yamlStr := `
outer:
  inner:
    target: found
`
	nodes := parseYAML(t, yamlStr)

	// Without recursive, looking for "inner" at top level should fail
	result := FindFieldPath("inner", nodes, FieldPathOptions{RecursiveFirstSegment: false})
	assert.False(t, result.Found)

	// With recursive, looking for "inner" should find it
	result = FindFieldPath("inner", nodes, FieldPathOptions{RecursiveFirstSegment: true})
	assert.True(t, result.Found)
}

func TestFindFieldPath_MultipleArrayIndexes(t *testing.T) {
	yamlStr := `
matrix:
  - - a
    - b
  - - c
    - d
`
	nodes := parseYAML(t, yamlStr)
	result := FindFieldPath("matrix[1][0]", nodes, FieldPathOptions{})
	assert.True(t, result.Found)
	assert.Equal(t, "c", result.ValueNode.Value)
}

func TestFindFieldPath_EscapedDotInYAMLKey(t *testing.T) {
	yamlStr := `
some.key: value
other: data
`
	nodes := parseYAML(t, yamlStr)
	result := FindFieldPath(`some\.key`, nodes, FieldPathOptions{})
	assert.True(t, result.Found)
	assert.Equal(t, "value", result.ValueNode.Value)
}

func TestFindFieldPath_PathWithSlashInStringIndex(t *testing.T) {
	yamlStr := `
paths:
  /pet:
    get:
      summary: Get pet
  /store:
    post:
      summary: Create store
`
	nodes := parseYAML(t, yamlStr)
	result := FindFieldPath("paths['/pet'].get", nodes, FieldPathOptions{})
	assert.True(t, result.Found)
}

func TestFindFieldPath_ComplexRealWorld(t *testing.T) {
	// Simulates the issue #787 scenario
	yamlStr := `
type: object
required:
  - data
  - pagination
properties:
  data:
    type: array
    items:
      type: object
  pagination:
    type: object
`
	nodes := parseYAML(t, yamlStr)
	result := FindFieldPath("properties.data", nodes, FieldPathOptions{})
	assert.True(t, result.Found)
	assert.Equal(t, "data", result.KeyNode.Value)
}

// Test backward compatibility - single level fields should work exactly as before
func TestFindFieldPath_BackwardCompatibility_SimplePath(t *testing.T) {
	yamlStr := `
name: test
description: a description
`
	nodes := parseYAML(t, yamlStr)

	// Test that simple paths still work
	result := FindFieldPath("name", nodes, FieldPathOptions{})
	assert.True(t, result.Found)
	assert.Equal(t, "test", result.ValueNode.Value)

	result = FindFieldPath("description", nodes, FieldPathOptions{})
	assert.True(t, result.Found)
	assert.Equal(t, "a description", result.ValueNode.Value)
}
