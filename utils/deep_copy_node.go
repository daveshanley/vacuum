// Copyright 2022-2025 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package utils

import "go.yaml.in/yaml/v4"

// DeepCopyNode creates a recursive deep copy of a *yaml.Node tree.
// All Node structs and Content slices are newly allocated.
// Line, Column, Kind, Tag, Value, Style, Anchor, and comments are preserved.
func DeepCopyNode(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}
	cp := &yaml.Node{
		Kind:        node.Kind,
		Style:       node.Style,
		Tag:         node.Tag,
		Value:       node.Value,
		Anchor:      node.Anchor,
		HeadComment: node.HeadComment,
		LineComment: node.LineComment,
		FootComment: node.FootComment,
		Line:        node.Line,
		Column:      node.Column,
	}
	if node.Alias != nil {
		cp.Alias = DeepCopyNode(node.Alias)
	}
	if len(node.Content) > 0 {
		cp.Content = make([]*yaml.Node, len(node.Content))
		for i, child := range node.Content {
			cp.Content[i] = DeepCopyNode(child)
		}
	}
	return cp
}
