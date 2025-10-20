package motor

import (
	"go.yaml.in/yaml/v4"
)

// checkInlineIgnore checks if a node should be ignored for a specific rule
// by looking for x-vacuum-ignore in the node itself.
func checkInlineIgnore(node *yaml.Node, ruleId string) bool {
	if node == nil || node.Kind != yaml.MappingNode {
		return false
	}

	// Look for x-vacuum-ignore key
	// Use i+1 < len to ensure we have both key and value before accessing
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value != "x-vacuum-ignore" {
			continue
		}

		if isRuleIgnored(node.Content[i+1], ruleId) {
			return true
		}
	}

	return false
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
