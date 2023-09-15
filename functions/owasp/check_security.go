package owasp

import (
	"fmt"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/utils"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

type CheckSecurity struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the CheckSecurity rule.
func (cd CheckSecurity) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "check_security"}
}

// RunRule will execute the CheckSecurity rule, based on supplied context and a supplied []*yaml.Node slice.
func (cd CheckSecurity) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	if len(nodes) <= 0 {
		return nil
	}

	var nullable bool
	nullableMap := utils.ExtractValueFromInterfaceMap("nullable", context.Options)
	if castedNullable, ok := nullableMap.(bool); ok {
		nullable = castedNullable
	}

	var methods []string
	methodsMap := utils.ExtractValueFromInterfaceMap("methods", context.Options)
	if castedMethods, ok := methodsMap.([]string); ok {
		methods = castedMethods
	}

	// security at the global level replaces if not defined at the operation level
	_, valueOfSecurityGlobalNode := findGlobalSecurityNode(nodes)

	var results []model.RuleFunctionResult
	_, valueOfPathNode := utils.FindFirstKeyNode("paths", nodes, 0)
	if valueOfPathNode == nil {
		return nil
	}

	for i := 1; i < len(valueOfPathNode.Content); i += 2 {
		for j := 0; j < len(valueOfPathNode.Content[i].Content); j += 2 {
			if slices.Contains(
				[]string{"get", "head", "post", "put", "patch", "delete", "options", "trace"},
				valueOfPathNode.Content[i].Content[j].Value,
			) && slices.Contains(
				methods,
				valueOfPathNode.Content[i].Content[j].Value,
			) && len(valueOfPathNode.Content[i].Content) > j+1 {
				operation := valueOfPathNode.Content[i].Content[j+1]
				results = append(results, checkSecurityRule(operation, valueOfSecurityGlobalNode, nullable, valueOfPathNode.Content[i-1].Value, valueOfPathNode.Content[i].Content[j].Value, context)...)
			}
		}
	}

	return results
}

func findGlobalSecurityNode(nodes []*yaml.Node) (keyNode *yaml.Node, valueNode *yaml.Node) {
	// Find the first document node. There should be only one, so just take the first
	var documentNode *yaml.Node
	for _, node := range nodes {
		if node.Kind != yaml.DocumentNode {
			continue
		}
		documentNode = node
		break
	}
	if documentNode == nil {
		return nil, nil
	}

	// Find the document's mapping node. There should be only one, so just take the first
	var mappingNode *yaml.Node
	for _, node := range documentNode.Content {
		if node.Kind != yaml.MappingNode {
			continue
		}
		mappingNode = node
		continue
	}
	if mappingNode == nil {
		return nil, nil
	}

	return utils.FindKeyNodeTop("security", mappingNode.Content)
}

func checkSecurityRule(operation *yaml.Node, valueOfSecurityGlobalNode *yaml.Node, nullable bool, pathPrefix, method string, context model.RuleFunctionContext) []model.RuleFunctionResult {
	_, valueOfSecurityNode := utils.FindFirstKeyNode("security", operation.Content, 0)
	if valueOfSecurityNode == nil { // if not defined at the operation level, use global
		valueOfSecurityNode = valueOfSecurityGlobalNode
	}
	if valueOfSecurityNode == nil {
		return []model.RuleFunctionResult{
			{
				Message:   fmt.Sprintf("'security' was not defined: for path %q in method %q.", pathPrefix, method),
				StartNode: operation,
				EndNode:   operation,
				Path:      fmt.Sprintf("$.paths.%s.%s", pathPrefix, method),
				Rule:      context.Rule,
			},
		}
	}
	if len(valueOfSecurityNode.Content) == 0 {
		return []model.RuleFunctionResult{
			{
				Message:   fmt.Sprintf("'security' is empty: for path %q in method %q.", pathPrefix, method),
				StartNode: valueOfSecurityNode,
				EndNode:   valueOfSecurityNode,
				Path:      fmt.Sprintf("$.paths.%s.%s.security", pathPrefix, method),
				Rule:      context.Rule,
			},
		}
	}
	if valueOfSecurityNode.Kind == yaml.SequenceNode {
		var results []model.RuleFunctionResult
		for k := 0; k < len(valueOfSecurityNode.Content); k++ {
			if valueOfSecurityNode.Content[k].Kind != yaml.MappingNode {
				continue
			}
			if len(valueOfSecurityNode.Content[k].Content) == 0 && !nullable {
				results = append(results, model.RuleFunctionResult{
					Message:   fmt.Sprintf("'security' has null elements: for path %q in method %q with element.", pathPrefix, method),
					StartNode: valueOfSecurityNode.Content[k],
					EndNode:   utils.FindLastChildNodeWithLevel(valueOfSecurityNode.Content[k], 0),
					Path:      fmt.Sprintf("$.paths.%s.%s.security", pathPrefix, method),
					Rule:      context.Rule,
				})
			}
		}
		return results
	}

	return nil
}
