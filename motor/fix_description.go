package motor

import (
	"strconv"
	"strings"

	"github.com/pb33f/go-yaml"
)

// Seed the set once so generated prose cannot duplicate authored descriptions,
// summaries, or another suggestion in the same review.
func fixDescriptionValues(root *yaml.Node) map[string]bool {
	used := make(map[string]bool)
	var walk func(*yaml.Node)
	walk = func(node *yaml.Node) {
		if node == nil {
			return
		}
		if node.Kind == yaml.MappingNode {
			for i := 0; i+1 < len(node.Content); i += 2 {
				key, value := node.Content[i], node.Content[i+1]
				if (key.Value == "description" || key.Value == "summary") && value.Kind == yaml.ScalarNode && value.Tag == "!!str" {
					used[value.Value] = true
				}
				walk(value)
			}
			return
		}
		for _, child := range node.Content {
			walk(child)
		}
	}
	walk(root)
	return used
}

func uniqueFixDescription(value, field string, used map[string]bool) string {
	if value == "" {
		return value
	}
	if used[value] {
		label := "Description of "
		if field == "summary" {
			label = "Summary of "
		}
		base := label + strings.TrimSuffix(value, ".")
		value = base + "."
		for suffix := 2; used[value]; suffix++ {
			value = base + " (" + strconv.Itoa(suffix) + ")."
		}
	}
	used[value] = true
	return value
}
