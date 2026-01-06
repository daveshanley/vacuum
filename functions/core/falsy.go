// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package core

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"go.yaml.in/yaml/v4"
)

// Falsy is a rule that will determine if something is seen as 'false' (could be a 0 or missing, or actually 'false')
type Falsy struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the Falsy rule.
func (f Falsy) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "falsy",
	}
}

// GetCategory returns the category of the Falsy rule.
func (f Falsy) GetCategory() string {
	return model.FunctionCategoryCore
}

// RunRule will execute the Falsy rule, based on supplied context and a supplied []*yaml.Node slice.
// If no field is specified, the function checks if the matched node itself is truthy (and reports if so).
// If a field is specified, the function checks if that field within the matched node is truthy.
func (f Falsy) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	pathValue := "unknown"
	if path, ok := context.Given.(string); ok {
		pathValue = path
	}

	ruleMessage := context.Rule.Description
	if context.Rule.Message != "" {
		ruleMessage = context.Rule.Message
	}

	for _, node := range nodes {
		// handle document nodes by unwrapping
		if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
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
			result := vacuumUtils.FindFieldPath(context.RuleAction.Field, node.Content, vacuumUtils.FieldPathOptions{RecursiveFirstSegment: true})
			fieldNode, targetNode = result.KeyNode, result.ValueNode
			fieldName = context.RuleAction.Field
		}

		// check if the target is truthy (which means falsy check fails)
		if targetNode != nil && isTruthyNode(targetNode) {
			var locatedObjects []v3.Foundational
			var allPaths []string
			var err error
			locatedPath := pathValue
			if context.DrDocument != nil {
				if fieldNode != nil {
					locatedObjects, err = context.DrDocument.LocateModelsByKeyAndValue(fieldNode, targetNode)
				} else {
					locatedObjects, err = context.DrDocument.LocateModel(node)
				}
				if err == nil && locatedObjects != nil {
					for x, obj := range locatedObjects {
						if x == 0 {
							locatedPath = obj.GenerateJSONPath()
						}
						allPaths = append(allPaths, obj.GenerateJSONPath())
					}
				}
			}
			result := model.RuleFunctionResult{
				Message:   fmt.Sprintf("%s: `%s` must be falsy", ruleMessage, fieldName),
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

// isTruthyNode checks if a YAML node represents a truthy value.
// A node is truthy if it has content, or has a non-empty/non-false/non-zero value.
func isTruthyNode(node *yaml.Node) bool {
	if node == nil {
		return false
	}
	// node with content (map or array) is truthy
	if len(node.Content) > 0 {
		return true
	}
	// scalar values: check for falsy values
	if node.Value == "" || node.Value == "false" || node.Value == "0" {
		return false
	}
	return true
}
