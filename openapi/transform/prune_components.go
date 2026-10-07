// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package transform

import (
	"github.com/pb33f/go-yaml"
)

// PruneUnusedComponents removes every reusable component that is unreachable
// from retained non-component document content.
func PruneUnusedComponents(root *yaml.Node, specVersion string) (PruneStats, error) {
	_, prune, err := Transform(root, specVersion, Options{PruneUnused: true})
	return prune, err
}

func pruneUnusedComponents(root *yaml.Node, specVersion string) (PruneStats, error) {
	stats := PruneStats{RemovedBySection: make(map[string]int), removed: make(map[ComponentID]struct{})}
	graph, err := buildComponentGraph(root, specVersion)
	if err != nil {
		return stats, err
	}
	stats.ComponentsSeen = len(graph.components)
	unreachable := graph.Unreachable()
	remove := make(map[ComponentID]struct{}, len(unreachable))
	for _, id := range unreachable {
		remove[id] = struct{}{}
		stats.removed[id] = struct{}{}
		stats.RemovedBySection[id.Section]++
	}
	stats.ComponentsRemoved = len(remove)
	stats.ComponentsKept = stats.ComponentsSeen - stats.ComponentsRemoved

	if isSwagger(specVersion) {
		for _, section := range []string{"definitions", "parameters", "responses", "securityDefinitions"} {
			pruneSection(root, section, section, remove)
		}
		return stats, nil
	}
	components := mapValue(root, "components")
	if components == nil || components.Kind != yaml.MappingNode {
		return stats, nil
	}
	knownSections := openAPIComponentSectionSet(specVersion)
	for i := 0; i+1 < len(components.Content); {
		section := components.Content[i].Value
		value := components.Content[i+1]
		if _, known := knownSections[section]; !known {
			i += 2
			continue
		}
		pruneEntries(value, section, remove)
		if value.Kind == yaml.MappingNode && len(value.Content) == 0 {
			removeMapPair(components, i)
			continue
		}
		i += 2
	}
	if len(components.Content) == 0 {
		for i := 0; i+1 < len(root.Content); i += 2 {
			if root.Content[i].Value == "components" {
				removeMapPair(root, i)
				break
			}
		}
	}
	return stats, nil
}

func pruneSection(root *yaml.Node, key, section string, remove map[ComponentID]struct{}) {
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value != key {
			continue
		}
		value := root.Content[i+1]
		pruneEntries(value, section, remove)
		if value.Kind == yaml.MappingNode && len(value.Content) == 0 {
			removeMapPair(root, i)
		}
		return
	}
}

func pruneEntries(node *yaml.Node, section string, remove map[ComponentID]struct{}) {
	if node == nil || node.Kind != yaml.MappingNode {
		return
	}
	kept := node.Content[:0]
	for i := 0; i+1 < len(node.Content); i += 2 {
		id := ComponentID{Section: section, Name: node.Content[i].Value}
		if _, discard := remove[id]; !discard {
			kept = append(kept, node.Content[i], node.Content[i+1])
		}
	}
	clear(node.Content[len(kept):])
	node.Content = kept
}
