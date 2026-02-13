package motor

import (
	"github.com/pb33f/libopenapi/utils"
	"go.yaml.in/yaml/v4"
	"time"
)

const ignoreKey = "x-lint-ignore"

// inlineIgnoreIndex holds pre-computed inline ignore data built during
// a single tree walk. This replaces per-result JSONPath queries with O(1)
// map lookups.
type inlineIgnoreIndex struct {
	// nodeIgnores maps yaml.Node pointers to their set of ignored rule IDs.
	// Only nodes that have x-lint-ignore are present.
	nodeIgnores map[*yaml.Node]map[string]bool

	// rootIgnores holds rule IDs ignored at the document root level.
	rootIgnores map[string]bool

	// hasNonRootIgnores is true if any non-root node has x-lint-ignore.
	// When false, checkInlineIgnoreByPathIndexed can skip JSONPath queries
	// and only check root-level ignores.
	hasNonRootIgnores bool
}

// buildInlineIgnoreIndex walks the YAML tree once, building a map of
// node -> ignored rule IDs. Returns nil if no x-lint-ignore keys exist.
func buildInlineIgnoreIndex(node *yaml.Node) *inlineIgnoreIndex {
	if node == nil {
		return nil
	}

	// Resolve root mapping node
	var rootNode *yaml.Node
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		rootNode = node.Content[0]
	} else {
		rootNode = node
	}

	idx := &inlineIgnoreIndex{
		nodeIgnores: make(map[*yaml.Node]map[string]bool),
	}

	scanAndIndex(node, idx, rootNode)

	// No ignores found at all
	if len(idx.nodeIgnores) == 0 {
		return nil
	}

	// Extract root ignores for fast access
	idx.rootIgnores = idx.nodeIgnores[rootNode]

	return idx
}

// scanAndIndex recursively walks the YAML tree, recording nodes that
// have x-lint-ignore directives.
func scanAndIndex(node *yaml.Node, idx *inlineIgnoreIndex, rootNode *yaml.Node) {
	if node == nil {
		return
	}
	switch node.Kind {
	case yaml.DocumentNode:
		for _, c := range node.Content {
			scanAndIndex(c, idx, rootNode)
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			if node.Content[i].Value == ignoreKey {
				rules := extractIgnoredRules(node.Content[i+1])
				if len(rules) > 0 {
					idx.nodeIgnores[node] = rules
					if node != rootNode {
						idx.hasNonRootIgnores = true
					}
				}
			}
			scanAndIndex(node.Content[i+1], idx, rootNode)
		}
	case yaml.SequenceNode:
		for _, c := range node.Content {
			scanAndIndex(c, idx, rootNode)
		}
	}
}

// extractIgnoredRules reads the value node of an x-lint-ignore directive
// and returns the set of rule IDs.
func extractIgnoredRules(ignoreNode *yaml.Node) map[string]bool {
	if ignoreNode == nil {
		return nil
	}
	switch ignoreNode.Kind {
	case yaml.ScalarNode:
		return map[string]bool{ignoreNode.Value: true}
	case yaml.SequenceNode:
		rules := make(map[string]bool, len(ignoreNode.Content))
		for _, item := range ignoreNode.Content {
			if item.Kind == yaml.ScalarNode {
				rules[item.Value] = true
			}
		}
		if len(rules) > 0 {
			return rules
		}
	}
	return nil
}

// checkInlineIgnoreByPathIndexed checks if a rule should be ignored at the given path.
// Uses the index to avoid JSONPath queries when possible:
// - Always checks root ignores via O(1) lookup
// - Only runs JSONPath query if non-root ignores exist
func checkInlineIgnoreByPathIndexed(idx *inlineIgnoreIndex, specNode *yaml.Node, path string, ruleId string) bool {
	if idx == nil || specNode == nil || path == "" {
		return false
	}

	// Check root-level ignores first (O(1))
	if idx.rootIgnores[ruleId] {
		return true
	}

	// If no non-root ignores exist, we're done — no JSONPath query needed
	if !idx.hasNonRootIgnores {
		return false
	}

	// Non-root ignores exist — must do JSONPath lookup for this path
	nodes, err := utils.FindNodesWithoutDeserializingWithTimeout(specNode, path, time.Millisecond*500)
	if err == nil && len(nodes) > 0 {
		if rules, ok := idx.nodeIgnores[nodes[0]]; ok {
			return rules[ruleId]
		}
	}

	return false
}

// checkInlineIgnore checks if a node should be ignored for a specific rule
// by scanning the node's own Content for an x-lint-ignore key.
// This uses direct content inspection (not the pointer-based index) because
// the node may come from either the resolved or unresolved document tree.
func checkInlineIgnore(node *yaml.Node, ruleId string) bool {
	if node == nil || node.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value != ignoreKey {
			continue
		}
		ignoreNode := node.Content[i+1]
		switch ignoreNode.Kind {
		case yaml.ScalarNode:
			if ignoreNode.Value == ruleId {
				return true
			}
		case yaml.SequenceNode:
			for _, item := range ignoreNode.Content {
				if item.Kind == yaml.ScalarNode && item.Value == ruleId {
					return true
				}
			}
		}
	}
	return false
}

// filterIgnoreNodes removes ignored nodes from the slice to prevent
// them from being processed by other rules.
func filterIgnoreNodes(nodes []*yaml.Node) []*yaml.Node {
	filtered := make([]*yaml.Node, 0, len(nodes))
	skipNext := false

	for _, node := range nodes {
		if skipNext {
			skipNext = false
			continue
		}

		if isIgnoreNode(node) {
			skipNext = true // Skip the value that follows this key
			continue
		}

		filtered = append(filtered, node)
	}
	return filtered
}

// isIgnoreNode checks if a node is an ignore key.
func isIgnoreNode(node *yaml.Node) bool {
	if node == nil {
		return false
	}

	return node.Kind == yaml.ScalarNode && node.Value == ignoreKey
}
