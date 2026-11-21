// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package motor

import (
	"iter"
	"strconv"
	"sync"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/go-yaml"
)

// Ordinary mapping keys retain their original nodes. Synthetic string keys
// refer back to their source item, or to the path at which an alias was used.
type ruleKeySource struct {
	node      *yaml.Node
	aliasPath string
}
type ruleKeySources map[*yaml.Node]ruleKeySource

func (s ruleKeySources) path(key *yaml.Node, lookup func(*yaml.Node) (string, bool)) (string, bool) {
	if source, ok := s[key]; ok {
		if source.aliasPath != "" {
			return source.aliasPath, true
		}
		key = source.node
	}
	return lookup(key)
}

// targets streams keys for ordinary functions. Batch functions retain source
// locations until they have processed the complete input slice.
func (s *ruleKeySources) targets(nodes []*yaml.Node, includeEmpty, batch bool, lookup func(*yaml.Node) (string, bool)) iter.Seq[*yaml.Node] {
	return func(yield func(*yaml.Node) bool) {
		emit := func(key, source *yaml.Node, aliasPath string) bool {
			if key == source && aliasPath == "" {
				return yield(key)
			}
			if *s == nil {
				*s = make(ruleKeySources)
			}
			(*s)[key] = ruleKeySource{source, aliasPath}
			more := yield(key)
			if !batch {
				delete(*s, key)
			}
			return more
		}
		var merged mergedKeys
		for _, node := range nodes {
			if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
				node = node.Content[0]
			}
			var aliasPath string
			if node.Kind == yaml.AliasNode && node.Alias != nil {
				aliasPath, _ = lookup(node)
				node = node.Alias
			}
			var mapKeys []mappingKey
			if node.Kind == yaml.MappingNode {
				mapKeys = merged.keys(node)
			}
			// A mapping whose merge keys inherit nothing has no keys either, so it is empty.
			empty := len(node.Content) == 0 || node.Kind == yaml.MappingNode && len(mapKeys) == 0
			if empty && includeEmpty && (node.Kind == yaml.MappingNode || node.Kind == yaml.SequenceNode) {
				// Functions receive an empty container; the source node still locates it.
				key := node
				if aliasPath != "" || len(node.Content) > 0 {
					clone := *node
					clone.Content = nil
					key = &clone
				}
				if !emit(key, node, aliasPath) {
					return
				}
				continue
			}
			switch node.Kind {
			case yaml.MappingNode:
				// Inherited keys belong to this mapping, not to their anchor.
				container, located := aliasPath, aliasPath != ""
				for _, entry := range mapKeys {
					source := entry.node
					key := source
					path := ""
					if source.Tag != "!!str" || aliasPath != "" || entry.inherited {
						key = stringKey(source.Value, source)
						if entry.inherited && !located {
							container, _ = lookup(node)
							located = true
						}
						if container != "" {
							path = model.AppendResultPathSegment(container, source.Value)
						}
					}
					if !emit(key, source, path) {
						return
					}
				}
			case yaml.SequenceNode:
				for i, item := range node.Content {
					path := ""
					if aliasPath != "" {
						path = model.AppendResultPathIndex(aliasPath, i)
					}
					if !emit(stringKey(strconv.Itoa(i), item), item, path) {
						return
					}
				}
			case yaml.ScalarNode:
				key := node
				if aliasPath != "" {
					clone := *node
					key = &clone
				}
				if !emit(key, node, aliasPath) {
					return
				}
			}
		}
	}
}

type mappingKey struct {
	node      *yaml.Node
	inherited bool
}

// mergedKeys expands YAML merge keys. Results are memoized per mapping, so
// repeated anchors cannot make expansion exponential, and cycles stop.
type mergedKeys struct {
	memo   map[*yaml.Node][]mappingKey
	active map[*yaml.Node]bool
}

// keys lists a mapping's effective keys. Local keys win over inherited keys,
// and within a merge sequence the first source wins, as YAML requires.
func (m *mergedKeys) keys(node *yaml.Node) []mappingKey {
	hasMerge := false
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Tag == "!!merge" {
			hasMerge = true
			break
		}
	}
	if !hasMerge {
		keys := make([]mappingKey, 0, len(node.Content)/2)
		for i := 0; i+1 < len(node.Content); i += 2 {
			keys = append(keys, mappingKey{node: node.Content[i]})
		}
		return keys
	}
	if keys, ok := m.memo[node]; ok {
		return keys
	}
	if m.active[node] {
		return nil
	}
	if m.memo == nil {
		m.memo = make(map[*yaml.Node][]mappingKey)
		m.active = make(map[*yaml.Node]bool)
	}
	m.active[node] = true
	defer delete(m.active, node)

	seen := make(map[string]bool, len(node.Content)/2)
	for i := 0; i+1 < len(node.Content); i += 2 {
		if key := node.Content[i]; key.Tag != "!!merge" {
			seen[key.Value] = true
		}
	}
	var keys []mappingKey
	for i := 0; i+1 < len(node.Content); i += 2 {
		key, value := node.Content[i], node.Content[i+1]
		if key.Tag != "!!merge" {
			keys = append(keys, mappingKey{node: key})
			continue
		}
		sources := []*yaml.Node{value}
		if value = dereferenceAlias(value); value.Kind == yaml.SequenceNode {
			sources = value.Content
		}
		for _, source := range sources {
			if source = dereferenceAlias(source); source.Kind != yaml.MappingNode {
				continue
			}
			for _, inherited := range m.keys(source) {
				if !seen[inherited.node.Value] {
					seen[inherited.node.Value] = true
					keys = append(keys, mappingKey{node: inherited.node, inherited: true})
				}
			}
		}
	}
	m.memo[node] = keys
	return keys
}

func dereferenceAlias(node *yaml.Node) *yaml.Node {
	if node.Kind == yaml.AliasNode && node.Alias != nil {
		return node.Alias
	}
	return node
}

func stringKey(value string, source *yaml.Node) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value, Line: source.Line, Column: source.Column}
}

func locateKeyResults(results []model.RuleFunctionResult, lookup func(*yaml.Node) (string, bool), custom bool) {
	for i := range results {
		r := &results[i]
		if custom && !r.PathFromRuleGiven && r.Path != "" {
			continue
		}
		if path, ok := lookup(r.StartNode); ok {
			r.Path = path
			r.KeyTarget = true
		}
	}
}

// Repeated references can place the same node at several paths in the resolved
// tree. Use the authored location in that case, independent of document order.
func nativeKeyPathLookup(ctx ruleContext, rfc model.RuleFunctionContext) func(*yaml.Node) (string, bool) {
	resolved := sync.OnceValue(func() *vacuumUtils.NodePathIndex { return vacuumUtils.NodePathIndexForContext(rfc, ctx.specNode) })
	authored := sync.OnceValue(func() *vacuumUtils.NodePathIndex {
		return vacuumUtils.NodePathIndexForContext(rfc, ctx.specNodeUnresolved)
	})
	return func(node *yaml.Node) (string, bool) {
		if path, ok := resolved().LookupUnique(node); ok {
			return path, true
		}
		if node != nil && ctx.indexUnresolved != nil && ctx.specNodeUnresolved != ctx.specNode {
			if origin := ctx.indexUnresolved.FindNodeOrigin(node); origin != nil {
				source := origin.Node
				if source.Kind == yaml.MappingNode && node.Kind == yaml.ScalarNode && len(source.Content) > 0 {
					source = source.Content[0]
				}
				if source.Kind == node.Kind && source.Value == node.Value {
					if path, ok := authored().Lookup(source); ok {
						return path, true
					}
				}
			}
		}
		return resolved().Lookup(node)
	}
}
