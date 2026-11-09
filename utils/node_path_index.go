// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package utils

import (
	"strings"
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
	positions map[*yaml.Node]int
	segments  []nodePathSegment
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
	if i.positions != nil {
		position, ok := i.positions[node]
		if !ok {
			return "", false
		}
		return i.compactPath(position), true
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
	root     *yaml.Node
	contract model.Contract
}

// NodePathIndexForContext returns an exact node path index, reusing the per-run cache when available.
func NodePathIndexForContext(context model.RuleFunctionContext, root *yaml.Node) *NodePathIndex {
	if root == nil {
		return nil
	}
	if context.SchemaPathCache == nil {
		return buildNodePathIndexForContract(root, context.Contract)
	}

	key := nodePathIndexCacheKey{root: root, contract: context.Contract}
	if cached, ok := context.SchemaPathCache.Load(key); ok {
		return cached.(func() *NodePathIndex)()
	}

	build := sync.OnceValue(func() *NodePathIndex { return buildNodePathIndexForContract(root, context.Contract) })
	cached, _ := context.SchemaPathCache.LoadOrStore(key, build)
	return cached.(func() *NodePathIndex)()
}

// nodePathSegment shares the prefix of a path with its parent. Index zero is $.
type nodePathSegment struct {
	parent int
	suffix string
}

func buildNodePathIndexForContract(root *yaml.Node, contract model.Contract) *NodePathIndex {
	if contract != model.OpenLintV1Draft {
		return BuildNodePathIndex(root)
	}
	// A resolved graph can reuse a large subtree at many paths. Keep the
	// original index for those graphs; compact segments would retain every
	// occurrence even though lookups only use the last path for each node.
	positions := make(map[*yaml.Node]int)
	segments := 0
	var count func(*yaml.Node) bool
	count = func(node *yaml.Node) bool {
		if node == nil {
			return true
		}
		if _, seen := positions[node]; seen {
			return false
		}
		positions[node] = 0
		if node.Kind == yaml.MappingNode {
			segments += len(node.Content) / 2
		}
		if node.Kind == yaml.SequenceNode {
			segments += len(node.Content)
		}
		for _, child := range node.Content {
			if !count(child) {
				return false
			}
		}
		return true
	}
	if !count(root) {
		return BuildNodePathIndex(root)
	}
	// Counting includes complex mapping-key children, which the path index
	// does not visit. Reuse the capacity without exposing those nodes as $.
	clear(positions)
	index := &NodePathIndex{positions: positions, segments: make([]nodePathSegment, 1, segments+1)}
	index.indexCompactNode(root, 0)
	return index
}

func (i *NodePathIndex) compactPath(position int) string {
	var local [32]int
	chain := local[:0]
	length := 1
	for position > 0 {
		chain = append(chain, position)
		length += len(i.segments[position].suffix)
		position = i.segments[position].parent
	}
	var path strings.Builder
	path.Grow(length)
	path.WriteByte('$')
	for j := len(chain) - 1; j >= 0; j-- {
		path.WriteString(i.segments[chain[j]].suffix)
	}
	return path.String()
}

func (i *NodePathIndex) indexCompactNode(node *yaml.Node, position int) {
	if node == nil {
		return
	}
	i.positions[node] = position
	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			i.indexCompactNode(child, position)
		}
	case yaml.MappingNode:
		for j := 0; j+1 < len(node.Content); j += 2 {
			key, value := node.Content[j], node.Content[j+1]
			child := len(i.segments)
			i.segments = append(i.segments, nodePathSegment{position, AppendResultPathSegment("", key.Value)})
			i.positions[key] = child
			i.indexCompactNode(value, child)
		}
	case yaml.SequenceNode:
		for j, value := range node.Content {
			child := len(i.segments)
			i.segments = append(i.segments, nodePathSegment{position, AppendResultPathIndex("", j)})
			i.indexCompactNode(value, child)
		}
	}
}
