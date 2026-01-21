// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package core

import (
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/libopenapi/utils"
	"go.yaml.in/yaml/v4"
	"sort"
)

// Truthy is a rule that will determine if something is seen as 'true' (could be a 1 or "pizza", or actually 'true')
type Truthy struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the Truthy rule.
func (t Truthy) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "truthy",
	}
}

// GetCategory returns the category of the Truthy rule.
func (t Truthy) GetCategory() string {
	return model.FunctionCategoryCore
}

// RunRule will execute the Truthy rule, based on supplied context and a supplied []*yaml.Node slice.
// If no field is specified, the function checks if the matched node itself is falsy (and reports if so).
// If a field is specified, the function checks if that field within the matched node is falsy.
func (t *Truthy) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult
	pathValue := "unknown"
	if path, ok := context.Given.(string); ok {
		pathValue = path
	}

	isArray := false
	if len(nodes) == 1 && utils.IsNodeArray(nodes[0]) {
		nodes = nodes[0].Content
		isArray = true
	}

	ruleMessage := context.Rule.Description
	message := context.Rule.Message

	for x, node := range nodes {

		if node.Kind == yaml.DocumentNode {
			node = node.Content[0]
		}

		var targetNode *yaml.Node
		var fieldNode *yaml.Node
		var fieldName string

		if context.RuleAction.Field == "" {
			// no field specified - check the matched node itself
			targetNode = node
			fieldName = "value"
		} else {
			// field specified - find it within the node (supports nested paths like "properties.data")
			result := vacuumUtils.FindFieldPath(context.RuleAction.Field, node.Content, vacuumUtils.FieldPathOptions{})
			fieldNode, targetNode = result.KeyNode, result.ValueNode
			fieldName = context.RuleAction.Field
		}

		// check if the target is falsy (which means truthy check fails)
		if isFalsyNode(targetNode) {
			if isArray {
				pathValue = model.GetStringTemplates().BuildArrayPath(pathValue, x)
			}

			// skip if target is a complex type (map or array with content)
			if targetNode != nil && (utils.IsNodeMap(targetNode) || utils.IsNodeArray(targetNode)) && len(targetNode.Content) > 0 {
				continue
			}

			if context.Index != nil {
				origin := context.Index.FindNodeOrigin(node)

				if origin != nil && origin.Line > 1 {
					nm := context.Index.GetNodeMap()
					var keys []int
					for k := range nm {
						keys = append(keys, k)
					}

					// Sort the keys slice.
					sort.Ints(keys)

					if len(keys) > 0 {
						np := nm[origin.Line-1][keys[0]]
						if np != nil {
							node = np
						}
					}
				}
			}

			var locatedObjects []v3.Foundational
			var allPaths []string
			var err error
			locatedPath := pathValue
			if context.DrDocument != nil {
				if fieldNode == nil {
					locatedObjects, err = context.DrDocument.LocateModel(node)
				} else {
					locatedObjects, err = context.DrDocument.LocateModelsByKeyAndValue(fieldNode, targetNode)
				}
				if err == nil && locatedObjects != nil {
					for i, obj := range locatedObjects {
						p := model.GetStringTemplates().BuildJSONPath(obj.GenerateJSONPath(), context.RuleAction.Field)
						if i == 0 {
							locatedPath = p
						}
						allPaths = append(allPaths, p)
					}
				}
			}
			result := model.RuleFunctionResult{
				Message: vacuumUtils.SuppliedOrDefault(message,
					model.GetStringTemplates().BuildFieldValidationMessage(ruleMessage, fieldName, "set")),
				StartNode: node,
				EndNode:   vacuumUtils.BuildEndNode(node),
				Path:      locatedPath,
				Rule:      context.Rule,
			}
			if len(allPaths) > 1 {
				result.Paths = allPaths
			}
			results = append(results, result)
			if len(locatedObjects) > 0 {
				if arr, ok := locatedObjects[0].(v3.AcceptsRuleResults); ok {
					arr.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
				}
			}
		}
	}
	return results
}

// isFalsyNode checks if a YAML node represents a falsy value.
// A node is falsy if it's nil, has no content, or has an empty/false/0 value.
func isFalsyNode(node *yaml.Node) bool {
	if node == nil {
		return true
	}
	// node with content (map or array) is truthy, not falsy
	if len(node.Content) > 0 {
		return false
	}
	// scalar values: check for falsy values
	if node.Value == "" || node.Value == "false" || node.Value == "0" {
		return true
	}
	return false
}
