// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package utils

import (
	"sync"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
)

// NodePathIndex maps YAML nodes back to their exact vacuum JSONPath.
// This is used when JSONPath expressions return nodes and vacuum needs to
// compare those matches against rule result paths.
type NodePathIndex struct {
	paths     map[*yaml.Node]string
	ambiguous map[*yaml.Node]bool
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

// LookupUnique returns a path only when the node occurs at one location in the tree.
// Resolved references can share nodes across several locations.
func (i *NodePathIndex) LookupUnique(node *yaml.Node) (string, bool) {
	path, ok := i.Lookup(node)
	return path, ok && !i.ambiguous[node]
}

func (i *NodePathIndex) record(node *yaml.Node, path string) {
	if node == nil {
		return
	}
	if previous, exists := i.paths[node]; exists && previous != path {
		if i.ambiguous == nil {
			i.ambiguous = make(map[*yaml.Node]bool)
		}
		i.ambiguous[node] = true
	}
	i.paths[node] = path
}

func (i *NodePathIndex) indexNode(node *yaml.Node, path string) {
	if i == nil || node == nil {
		return
	}
	i.record(node, path)

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

			i.record(keyNode, childPath)
			i.indexNode(valueNode, childPath)
		}
	case yaml.SequenceNode:
		for idx, child := range node.Content {
			childPath := AppendResultPathIndex(path, idx)
			i.indexNode(child, childPath)
		}
	}
}

type nodePathIndexCacheKey struct {
	root *yaml.Node
}

// NodePathIndexForContext returns an exact node path index, reusing the per-run cache when available.
func NodePathIndexForContext(context model.RuleFunctionContext, root *yaml.Node) *NodePathIndex {
	if root == nil {
		return nil
	}
	if context.SchemaPathCache == nil {
		return BuildNodePathIndex(root)
	}

	key := nodePathIndexCacheKey{root: root}
	if cached, ok := context.SchemaPathCache.Load(key); ok {
		return cached.(func() *NodePathIndex)()
	}

	build := sync.OnceValue(func() *NodePathIndex { return BuildNodePathIndex(root) })
	cached, _ := context.SchemaPathCache.LoadOrStore(key, build)
	return cached.(func() *NodePathIndex)()
}
