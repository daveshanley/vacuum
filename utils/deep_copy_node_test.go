// Copyright 2022-2025 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestDeepCopyNode_Nil(t *testing.T) {
	assert.Nil(t, DeepCopyNode(nil))
}

func TestDeepCopyNode_PreservesFields(t *testing.T) {
	node := &yaml.Node{
		Kind:        yaml.ScalarNode,
		Style:       yaml.DoubleQuotedStyle,
		Tag:         "!!str",
		Value:       "hello",
		Anchor:      "myAnchor",
		HeadComment: "head",
		LineComment: "line",
		FootComment: "foot",
		Line:        42,
		Column:      7,
	}

	cp := DeepCopyNode(node)

	assert.Equal(t, node.Kind, cp.Kind)
	assert.Equal(t, node.Style, cp.Style)
	assert.Equal(t, node.Tag, cp.Tag)
	assert.Equal(t, node.Value, cp.Value)
	assert.Equal(t, node.Anchor, cp.Anchor)
	assert.Equal(t, node.HeadComment, cp.HeadComment)
	assert.Equal(t, node.LineComment, cp.LineComment)
	assert.Equal(t, node.FootComment, cp.FootComment)
	assert.Equal(t, node.Line, cp.Line)
	assert.Equal(t, node.Column, cp.Column)
}

func TestDeepCopyNode_Independence(t *testing.T) {
	node := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "original",
		Line:  10,
	}

	cp := DeepCopyNode(node)
	cp.Value = "modified"
	cp.Line = 99

	assert.Equal(t, "original", node.Value)
	assert.Equal(t, 10, node.Line)
}

func TestDeepCopyNode_TreeStructure(t *testing.T) {
	root := &yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "key", Line: 1, Column: 1},
					{Kind: yaml.ScalarNode, Value: "value", Line: 1, Column: 6},
				},
			},
		},
	}

	cp := DeepCopyNode(root)

	// Structure preserved
	assert.Equal(t, yaml.DocumentNode, cp.Kind)
	assert.Len(t, cp.Content, 1)
	assert.Equal(t, yaml.MappingNode, cp.Content[0].Kind)
	assert.Len(t, cp.Content[0].Content, 2)
	assert.Equal(t, "key", cp.Content[0].Content[0].Value)
	assert.Equal(t, "value", cp.Content[0].Content[1].Value)
	assert.Equal(t, 1, cp.Content[0].Content[0].Line)
	assert.Equal(t, 6, cp.Content[0].Content[1].Column)

	// Independence: mutating copy doesn't affect original
	cp.Content[0].Content[0].Value = "changed"
	assert.Equal(t, "key", root.Content[0].Content[0].Value)

	// Pointer independence
	assert.NotSame(t, root, cp)
	assert.NotSame(t, root.Content[0], cp.Content[0])
	assert.NotSame(t, root.Content[0].Content[0], cp.Content[0].Content[0])
}

func TestDeepCopyNode_EmptyContent(t *testing.T) {
	node := &yaml.Node{
		Kind:    yaml.MappingNode,
		Content: nil,
	}
	cp := DeepCopyNode(node)
	assert.Equal(t, yaml.MappingNode, cp.Kind)
	assert.Nil(t, cp.Content)
}

func TestDeepCopyNode_RealYAML(t *testing.T) {
	input := `openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /test:
    get:
      summary: A test endpoint`

	var root yaml.Node
	err := yaml.Unmarshal([]byte(input), &root)
	assert.NoError(t, err)

	cp := DeepCopyNode(&root)

	// Re-marshal both and compare
	origBytes, err := yaml.Marshal(&root)
	assert.NoError(t, err)
	cpBytes, err := yaml.Marshal(cp)
	assert.NoError(t, err)

	assert.Equal(t, string(origBytes), string(cpBytes))

	// Mutate the copy and verify original is unchanged
	if cp.Content[0].Kind == yaml.MappingNode && len(cp.Content[0].Content) >= 2 {
		cp.Content[0].Content[1].Value = "MUTATED"
	}
	origBytes2, _ := yaml.Marshal(&root)
	assert.Equal(t, string(origBytes), string(origBytes2))
}
