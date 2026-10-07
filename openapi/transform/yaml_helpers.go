package transform

import (
	"fmt"
	"strings"

	"github.com/pb33f/go-yaml"
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
	clear(node.Content[len(node.Content)-2:])
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
		case "example", "default", "const", "enum":
			if !isNamedObjectMapEntry(path, i) {
				return true
			}
		case "value":
			if i >= 2 && path[i-2] == "examples" {
				return true
			}
		}
	}
	return false
}

// isArbitraryData distinguishes schema examples arrays from the named Example
// Object maps used by media types and components. A numeric key is still a name.
func isArbitraryData(node *yaml.Node, path []string) bool {
	if isWithinArbitraryExample(path) {
		return true
	}
	last := len(path) - 1
	return node != nil && node.Kind == yaml.SequenceNode && last >= 0 &&
		path[last] == "examples" && !isNamedObjectMapEntry(path, last)
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

// expandYAMLReferences materializes merge keys and, for filtering, aliases.
// Decode applies YAML merge precedence and rejects recursive or excessive
// alias expansion. Work on a new root so a failed transform leaves input intact.
func expandYAMLReferences(root *yaml.Node, aliases bool) (*yaml.Node, error) {
	needsExpansion := false
	walkYAMLNodes(root, func(node *yaml.Node) {
		if (aliases && node.Kind == yaml.AliasNode) || node.Tag == "!!merge" {
			needsExpansion = true
		}
	})
	if !needsExpansion {
		return root, nil
	}
	var value any
	if err := root.Decode(&value); err != nil {
		return nil, fmt.Errorf("cannot expand YAML aliases and merge keys: %w", err)
	}
	var expanded yaml.Node
	if err := expanded.Encode(value); err != nil {
		return nil, err
	}
	return &expanded, nil
}
