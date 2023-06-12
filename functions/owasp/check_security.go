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
	return model.RuleFunctionSchema{Name: "define_error"}
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

	var results []model.RuleFunctionResult
	_, valueOfPathNode := utils.FindFirstKeyNode("paths", nodes, 0)
	for i := 1; i < len(valueOfPathNode.Content); i += 2 {
		for j := 0; j < len(valueOfPathNode.Content[i].Content); j += 2 {
			if slices.Contains([]string{
				"get",
				"head",
				"post",
				"put",
				"patch",
				"delete",
				"options",
				"trace",
			}, valueOfPathNode.Content[i].Content[j].Value) && slices.Contains(methods, valueOfPathNode.Content[i].Content[j].Value) && len(valueOfPathNode.Content[i].Content) > j+1 {
				operation := valueOfPathNode.Content[i].Content[j+1]
				_, valueOfSecurityNode := utils.FindFirstKeyNode("security", operation.Content, 0)
				if valueOfSecurityNode == nil {
					results = append(results, model.RuleFunctionResult{
						Message:   "", // TODO
						StartNode: nodes[0],
						EndNode:   utils.FindLastChildNodeWithLevel(nodes[0], 0),
						Path:      fmt.Sprintf("$.paths.%s.%s", valueOfPathNode.Content[i-1].Value, valueOfPathNode.Content[i].Content[j].Value), // TODO
						Rule:      context.Rule,
					})
					continue
				}
				if len(valueOfSecurityNode.Content) == 0 {
					results = append(results, model.RuleFunctionResult{
						Message:   "", // TODO
						StartNode: nodes[0],
						EndNode:   utils.FindLastChildNodeWithLevel(nodes[0], 0),
						Path:      fmt.Sprintf("$.paths.%s.%s.security", valueOfPathNode.Content[i-1].Value, valueOfPathNode.Content[i].Content[j].Value), // TODO
						Rule:      context.Rule,
					})
					continue
				}
				if valueOfSecurityNode.Kind == yaml.SequenceNode {
					for k := 0; k < len(valueOfSecurityNode.Content); k++ {
						if valueOfSecurityNode.Content[k].Kind != yaml.MappingNode {
							continue
						}
						if len(valueOfSecurityNode.Content[k].Content) == 0 && !nullable {
							results = append(results, model.RuleFunctionResult{
								Message:   "", // TODO
								StartNode: nodes[0],
								EndNode:   utils.FindLastChildNodeWithLevel(nodes[0], 0),
								Path:      fmt.Sprintf("$.paths.%s.%s", valueOfPathNode.Content[i-1].Value, valueOfPathNode.Content[i].Content[j].Value), // TODO
								Rule:      context.Rule,
							})
						}
					}
				}
			}
		}
	}

	return results

	// []model.RuleFunctionResult{
	// 	{
	// 		Message:   "", // TODO
	// 		StartNode: nodes[0],
	// 		EndNode:   utils.FindLastChildNodeWithLevel(nodes[0], 0),
	// 		Path:      "", // TODO
	// 		Rule:      context.Rule,
	// 	},
	// }
}
