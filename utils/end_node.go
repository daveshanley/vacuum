// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package utils

import "gopkg.in/yaml.v3"

// BuildEndNode will return a new yaml.Node with the same line as the input node, but with a column
// that is the sum of the input node's column and the length of the input node's value.
func BuildEndNode(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}
	modifier := 0
	// quotes need to be accounted for
	if node.Style == yaml.DoubleQuotedStyle || node.Style == yaml.SingleQuotedStyle {
		modifier = 2
	}
	return &yaml.Node{Line: node.Line, Column: node.Column + len(node.Value) + modifier, Kind: node.Kind, Value: node.Value}
}
