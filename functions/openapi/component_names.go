package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/go-yaml"
	"strings"
)

// The OpenAPI 3.0 metaschema permits unmatched patternProperties keys. Enforce
// the Components Object's naming requirement without changing schema validation.
func checkComponentNames(root *yaml.Node, rule *model.Rule) []model.RuleFunctionResult {
	components := mappingValueNode(documentContentNode(root), "components")
	var results []model.RuleFunctionResult
	for _, category := range []string{"schemas", "responses", "parameters", "examples", "requestBodies", "headers", "securitySchemes", "links", "callbacks"} {
		entries := mappingValueNode(components, category)
		if entries == nil || entries.Kind != yaml.MappingNode {
			continue
		}
		for i := 0; i+1 < len(entries.Content); i += 2 {
			key := entries.Content[i]
			if key.Tag != "!!str" || validComponentName(key.Value) {
				continue
			}
			reason := fmt.Sprintf("component name %q does not match pattern `^[a-zA-Z0-9._-]+$`", key.Value)
			results = append(results, propertyNameResult(root, []string{"components", category, key.Value}, reason, rule))
		}
	}
	return results
}

func validComponentName(name string) bool {
	if name == "" {
		return false
	}
	for _, c := range name {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

func propertyNameResult(root *yaml.Node, path []string, reason string, rule *model.Rule) model.RuleFunctionResult {
	var pointer strings.Builder
	node := documentContentNode(root)
	var keyNode *yaml.Node
	for _, segment := range path {
		pointer.WriteByte('/')
		pointer.WriteString(strings.ReplaceAll(strings.ReplaceAll(segment, "~", "~0"), "/", "~1"))
		keyNode = nil
		if node != nil && node.Kind == yaml.MappingNode {
			for i := 0; i+1 < len(node.Content); i += 2 {
				if node.Content[i].Value == segment {
					keyNode = node.Content[i]
					break
				}
			}
		}
		node = mappingValueNode(node, segment)
	}
	if keyNode == nil {
		keyNode = &yaml.Node{Line: 1, Column: 1}
	}
	location := jsonPointerToJSONPath(pointer.String(), root)
	return model.RuleFunctionResult{Message: "schema invalid: " + reason, StartNode: keyNode, EndNode: vacuumUtils.BuildEndNode(keyNode), Path: location, Paths: []string{location}, Rule: rule}
}
