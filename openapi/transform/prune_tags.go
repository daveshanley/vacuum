// Copyright 2026 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package transform

import "github.com/pb33f/go-yaml"

func pruneUnusedTags(root *yaml.Node, version string) int {
	tags := mapValue(root, "tags")
	if tags == nil || tags.Kind != yaml.SequenceNode {
		return 0
	}
	used := make(map[string]struct{})
	visitOperations(root, version, func(operation *yaml.Node, _ []string) {
		if values := mapValue(operation, "tags"); values != nil && values.Kind == yaml.SequenceNode {
			for _, value := range values.Content {
				if value.Kind == yaml.ScalarNode && value.Tag == "!!str" {
					used[value.Value] = struct{}{}
				}
			}
		}
	})
	// OpenAPI 3.2 tag parents remain necessary when a child tag is used.
	if supportsQueryAndAdditionalOperations(version) {
		parents := make(map[string]string)
		for _, tag := range tags.Content {
			name, parent := mapValue(tag, "name"), mapValue(tag, "parent")
			if name != nil && parent != nil {
				parents[name.Value] = parent.Value
			}
		}
		for name := range used {
			for parent := parents[name]; parent != ""; parent = parents[parent] {
				if _, exists := used[parent]; exists {
					break
				}
				used[parent] = struct{}{}
			}
		}
	}
	kept := tags.Content[:0]
	removed := 0
	for _, tag := range tags.Content {
		name := mapValue(tag, "name")
		if name != nil && name.Kind == yaml.ScalarNode && name.Tag == "!!str" {
			if _, exists := used[name.Value]; !exists {
				removed++
				continue
			}
		}
		kept = append(kept, tag)
	}
	clear(tags.Content[len(kept):])
	tags.Content = kept
	return removed
}
