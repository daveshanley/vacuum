package transform

import (
	"fmt"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v4"
)

var namedObjectMapKeys = map[string]struct{}{
	"callbacks":           {},
	"content":             {},
	"definitions":         {},
	"dependentSchemas":    {},
	"encoding":            {},
	"examples":            {},
	"headers":             {},
	"links":               {},
	"mediaTypes":          {},
	"parameters":          {},
	"pathItems":           {},
	"paths":               {},
	"patternProperties":   {},
	"properties":          {},
	"requestBodies":       {},
	"responses":           {},
	"schemas":             {},
	"securityDefinitions": {},
	"securitySchemes":     {},
	"webhooks":            {},
	"$defs":               {},
}

func documentRoot(node *yaml.Node) *yaml.Node {
	if node != nil && node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return node.Content[0]
	}
	return node
}

func mapValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func removeMapPair(node *yaml.Node, index int) {
	copy(node.Content[index:], node.Content[index+2:])
	node.Content = node.Content[:len(node.Content)-2]
}

func jsonPath(parts []string) string {
	var b strings.Builder
	b.WriteByte('$')
	for _, part := range parts {
		if part == "" {
			continue
		}
		b.WriteString("['")
		b.WriteString(strings.ReplaceAll(part, "'", "\\'"))
		b.WriteString("']")
	}
	return b.String()
}

func requireMappingRoot(root *yaml.Node) (*yaml.Node, error) {
	root = documentRoot(root)
	if root == nil || root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("OpenAPI document root must be a mapping")
	}
	return root, nil
}

// isWithinArbitraryExample reports whether path is inside example data rather
// than an OpenAPI object. Example values may contain keys such as $ref that are
// ordinary payload data and must not participate in document reference checks.
func isWithinArbitraryExample(path []string) bool {
	for i, part := range path {
		switch part {
		case "example":
			if !isNamedObjectMapEntry(path, i) {
				return true
			}
		case "examples":
			if i+1 < len(path) {
				if _, err := strconv.Atoi(path[i+1]); err == nil {
					return true
				}
			}
		case "value":
			if i >= 2 && path[i-2] == "examples" {
				return true
			}
		}
	}
	return false
}

// isWithinExtension reports whether path is inside an OpenAPI specification
// extension. Extension values are arbitrary, but local component references in
// them are still followed conservatively by the pruning graph.
func isWithinExtension(path []string) bool {
	for i, part := range path {
		if strings.HasPrefix(part, "x-") && !isNamedObjectMapEntry(path, i) {
			return true
		}
	}
	return false
}

func isNamedObjectMapEntry(path []string, index int) bool {
	if index == 0 {
		return false
	}
	_, ok := namedObjectMapKeys[path[index-1]]
	return ok
}
