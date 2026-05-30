// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package jsonschema

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"go.yaml.in/yaml/v4"
)

func RootNode(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return node.Content[0]
	}
	return node
}

func NodeToInterface(node *yaml.Node) (any, error) {
	node = RootNode(node)
	if node == nil {
		return nil, nil
	}
	switch node.Kind {
	case yaml.MappingNode:
		m := make(map[string]any, len(node.Content)/2)
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i].Value
			val, err := NodeToInterface(node.Content[i+1])
			if err != nil {
				return nil, err
			}
			m[key] = val
		}
		return m, nil
	case yaml.SequenceNode:
		arr := make([]any, 0, len(node.Content))
		for _, child := range node.Content {
			val, err := NodeToInterface(child)
			if err != nil {
				return nil, err
			}
			arr = append(arr, val)
		}
		return arr, nil
	case yaml.ScalarNode:
		return scalarToInterface(node)
	case yaml.AliasNode:
		return NodeToInterface(node.Alias)
	default:
		return nil, nil
	}
}

func IsFragmentRoot(root *yaml.Node) bool {
	root = RootNode(root)
	if root == nil || root.Kind != yaml.MappingNode {
		return false
	}
	hasDefs := false
	for i := 0; i+1 < len(root.Content); i += 2 {
		key := root.Content[i].Value
		switch key {
		case "$schema", "$id", "$defs", "definitions", "$comment", "title", "description":
			if key == "$defs" || key == "definitions" {
				hasDefs = true
			}
		default:
			if !strings.HasPrefix(key, "x-") {
				return false
			}
		}
	}
	return hasDefs
}

func IsDelegatingRefRoot(root *yaml.Node) bool {
	root = RootNode(root)
	if root == nil || root.Kind != yaml.MappingNode {
		return false
	}
	hasRef := false
	for i := 0; i+1 < len(root.Content); i += 2 {
		key := root.Content[i].Value
		switch key {
		case "$ref":
			hasRef = true
		case "$schema", "$id", "$defs", "definitions", "$comment", "title", "description":
		default:
			if !strings.HasPrefix(key, "x-") {
				return false
			}
		}
	}
	return hasRef
}

func FindNodeByLocation(root *yaml.Node, location []string) (*yaml.Node, string) {
	root = RootNode(root)
	if root == nil {
		return nil, "$"
	}
	node := root
	path := "$"
	for _, segment := range location {
		if node == nil {
			return root, "$"
		}
		switch node.Kind {
		case yaml.MappingNode:
			next := mappingValueNode(node, segment)
			if next == nil {
				return node, path
			}
			node = next
			path = vacuumUtils.AppendResultPathSegment(path, segment)
		case yaml.SequenceNode:
			idx, err := strconv.Atoi(segment)
			if err != nil || idx < 0 || idx >= len(node.Content) {
				return node, path
			}
			node = node.Content[idx]
			path = vacuumUtils.AppendResultPathIndex(path, idx)
		default:
			return node, path
		}
	}
	return node, path
}

func InstanceLocationPointer(location []string) string {
	if len(location) == 0 {
		return "#"
	}
	escaped := make([]string, len(location))
	for i, segment := range location {
		segment = strings.ReplaceAll(segment, "~", "~0")
		segment = strings.ReplaceAll(segment, "/", "~1")
		escaped[i] = segment
	}
	return "#/" + strings.Join(escaped, "/")
}

func MappingValueNode(node *yaml.Node, key string) *yaml.Node {
	return mappingValueNode(node, key)
}

func ToJSON(root *yaml.Node, pretty bool) ([]byte, error) {
	data, err := NodeToInterface(root)
	if err != nil {
		return nil, err
	}
	if pretty {
		return json.MarshalIndent(data, "", "  ")
	}
	return json.Marshal(data)
}

func scalarToInterface(node *yaml.Node) (any, error) {
	switch node.Tag {
	case "!!null":
		return nil, nil
	case "!!bool":
		return strconv.ParseBool(node.Value)
	case "!!int":
		i, err := strconv.ParseInt(node.Value, 10, 64)
		if err == nil {
			return i, nil
		}
		return nil, fmt.Errorf("unable to parse integer value %q at line %d, column %d: %w", node.Value, node.Line, node.Column, err)
	case "!!float":
		f, err := strconv.ParseFloat(node.Value, 64)
		if err == nil {
			return f, nil
		}
		return nil, fmt.Errorf("unable to parse float value %q at line %d, column %d: %w", node.Value, node.Line, node.Column, err)
	}
	return node.Value, nil
}

func mappingValueNode(node *yaml.Node, key string) *yaml.Node {
	node = RootNode(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func mappingScalarValue(node *yaml.Node, key string) string {
	val := mappingValueNode(node, key)
	if val == nil || val.Kind != yaml.ScalarNode {
		return ""
	}
	return val.Value
}
