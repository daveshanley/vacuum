package motor

import (
	"go.yaml.in/yaml/v4"
)

const ignoreKey = "x-lint-ignore"

// checkInlineIgnore checks if a node should be ignored for a specific rule
// by looking for the ignore key ignore in the node itself.
func checkInlineIgnore(node *yaml.Node, ruleId string) bool {
	if node == nil || node.Kind != yaml.MappingNode {
		return false
	}

	// Look for ignore key
	// Use i+1 < len to ensure we have both key and value before accessing
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value != ignoreKey {
			continue
		}

		if isRuleIgnored(node.Content[i+1], ruleId) {
			return true
		}
	}

	return false
}

// filterIgnoreNodes removes ignored nodes from the slice to prevent
// them from being processed by other rules.
func filterIgnoreNodes(nodes []*yaml.Node) []*yaml.Node {
	var filtered []*yaml.Node
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

// isRuleIgnored checks if a rule ID is in the ignore value.
func isRuleIgnored(ignoreNode *yaml.Node, ruleId string) bool {
	switch ignoreNode.Kind {
	case yaml.ScalarNode:
		return ignoreNode.Value == ruleId

	case yaml.SequenceNode:
		for _, item := range ignoreNode.Content {
			if item.Kind == yaml.ScalarNode && item.Value == ruleId {
				return true
			}
		}
	}

	return false
}
