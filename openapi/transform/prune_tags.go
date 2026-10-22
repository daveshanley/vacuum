// Copyright 2026 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package transform

import (
	"github.com/pb33f/go-yaml"
	"strings"
)

func unusedTags(root *yaml.Node, version string) map[*yaml.Node]bool {
	tags := resolvedMapValue(root, "tags")
	if tags == nil || tags.Kind != yaml.SequenceNode {
		return nil
	}
	used := make(map[string]struct{})
	visitOperations(root, version, func(operation *yaml.Node, _ []string) {
		if values := resolvedMapValue(operation, "tags"); values != nil && values.Kind == yaml.SequenceNode {
			for _, value := range values.Content {
				value = aliasTarget(value)
				if value.Kind == yaml.ScalarNode && value.Tag == "!!str" {
					used[value.Value] = struct{}{}
				}
			}
		}
	})
	// Documentation groups and narrative tags are uses even without operations.
	if groups := resolvedMapValue(root, "x-tagGroups"); groups != nil && groups.Kind == yaml.SequenceNode {
		for _, group := range groups.Content {
			if names := resolvedMapValue(group, "tags"); names != nil && names.Kind == yaml.SequenceNode {
				for _, name := range names.Content {
					name = aliasTarget(name)
					if name.Kind == yaml.ScalarNode && name.Tag == "!!str" {
						used[name.Value] = struct{}{}
					}
				}
			}
		}
	}
	for _, tag := range tags.Content {
		if trait := resolvedMapValue(tag, "x-traitTag"); trait != nil && trait.Tag == "!!bool" && strings.EqualFold(trait.Value, "true") {
			if name := resolvedMapValue(tag, "name"); name != nil {
				used[name.Value] = struct{}{}
			}
		}
	}
	// OpenAPI 3.2 tag parents remain necessary when a child tag is used.
	if supportsQueryAndAdditionalOperations(version) {
		parents := make(map[string]string)
		for _, tag := range tags.Content {
			name, parent := resolvedMapValue(tag, "name"), resolvedMapValue(tag, "parent")
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
	unused := make(map[*yaml.Node]bool)
	for _, tag := range tags.Content {
		name := resolvedMapValue(tag, "name")
		if name != nil && name.Kind == yaml.ScalarNode && name.Tag == "!!str" {
			if _, exists := used[name.Value]; !exists {
				unused[tag] = true
			}
		}
	}
	return unused
}

func unusedTagsOwnAnchors(unused map[*yaml.Node]bool) bool {
	var anchored func(*yaml.Node) bool
	anchored = func(node *yaml.Node) bool {
		if node.Anchor != "" {
			return true
		}
		for _, child := range node.Content {
			if anchored(child) {
				return true
			}
		}
		return false
	}
	for node := range unused {
		if anchored(node) {
			return true
		}
	}
	return false
}

func pruneUnusedTags(root *yaml.Node, unused map[*yaml.Node]bool) int {
	tags := resolvedMapValue(root, "tags")
	if len(unused) == 0 {
		return 0
	}
	kept := tags.Content[:0]
	for _, tag := range tags.Content {
		if !unused[tag] {
			kept = append(kept, tag)
		}
	}
	clear(tags.Content[len(kept):])
	tags.Content = kept
	return len(unused)
}
