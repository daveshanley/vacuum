package transform

import (
	"strings"

	"go.yaml.in/yaml/v4"
)

// PruneUnusedComponents removes every reusable component that is unreachable
// from retained non-component document content.
func PruneUnusedComponents(root *yaml.Node, specVersion string) (PruneStats, error) {
	stats := PruneStats{RemovedBySection: make(map[string]int), removed: make(map[ComponentID]struct{})}
	root, err := requireMappingRoot(root)
	if err != nil {
		return stats, err
	}
	graph, err := BuildComponentGraph(root, specVersion)
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

	if strings.HasPrefix(specVersion, "2.") {
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
		if !knownSections[section] {
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
	for i := 0; i+1 < len(node.Content); {
		id := ComponentID{Section: section, Name: node.Content[i].Value}
		if _, ok := remove[id]; ok {
			removeMapPair(node, i)
			continue
		}
		i += 2
	}
}
