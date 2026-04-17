// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package utils

import "go.yaml.in/yaml/v4"

// NodePathIndex maps YAML nodes back to their exact vacuum JSONPath.
// This is used when JSONPath expressions return nodes and vacuum needs to
// compare those matches against rule result paths.
type NodePathIndex struct {
	paths map[*yaml.Node]string
}

// BuildNodePathIndex creates an exact path index for the supplied YAML tree.
func BuildNodePathIndex(root *yaml.Node) *NodePathIndex {
	if root == nil {
		return nil
	}
	index := &NodePathIndex{
		paths: make(map[*yaml.Node]string),
	}
	index.indexNode(root, "$")
	return index
}

// Lookup returns the exact JSONPath for a node if it exists in the index.
func (i *NodePathIndex) Lookup(node *yaml.Node) (string, bool) {
	if i == nil || node == nil {
		return "", false
	}
	path, ok := i.paths[node]
	return path, ok
}

func (i *NodePathIndex) indexNode(node *yaml.Node, path string) {
	if i == nil || node == nil {
		return
	}
	if _, exists := i.paths[node]; !exists {
		i.paths[node] = path
	}

	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			i.indexNode(child, path)
		}
	case yaml.MappingNode:
		for idx := 0; idx+1 < len(node.Content); idx += 2 {
			keyNode := node.Content[idx]
			valueNode := node.Content[idx+1]
			childPath := AppendResultPathSegment(path, keyNode.Value)

			if keyNode != nil {
				i.paths[keyNode] = childPath
			}
			if valueNode != nil {
				i.paths[valueNode] = childPath
			}
			i.indexNode(valueNode, childPath)
		}
	case yaml.SequenceNode:
		for idx, child := range node.Content {
			childPath := AppendResultPathIndex(path, idx)
			if child != nil {
				i.paths[child] = childPath
			}
			i.indexNode(child, childPath)
		}
	}
}
