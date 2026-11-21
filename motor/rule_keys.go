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
		for _, node := range nodes {
			if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
				node = node.Content[0]
			}
			var aliasPath string
			if node.Kind == yaml.AliasNode && node.Alias != nil {
				aliasPath, _ = lookup(node)
				node = node.Alias
			}
			if len(node.Content) == 0 && includeEmpty && (node.Kind == yaml.MappingNode || node.Kind == yaml.SequenceNode) {
				key := node
				if aliasPath != "" {
					clone := *node
					key = &clone
				}
				if !emit(key, node, aliasPath) {
					return
				}
				continue
			}
			switch node.Kind {
			case yaml.MappingNode:
				for i := 0; i+1 < len(node.Content); i += 2 {
					source := node.Content[i]
					key := source
					path := ""
					if source.Tag != "!!str" || aliasPath != "" {
						key = stringKey(source.Value, source)
						if aliasPath != "" {
							path = model.AppendResultPathSegment(aliasPath, source.Value)
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
