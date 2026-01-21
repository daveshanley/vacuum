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

// Undefined is a rule that will check if a field has not been defined.
type Undefined struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the Undefined rule.
func (u Undefined) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "undefined",
	}
}

// GetCategory returns the category of the Undefined rule.
func (u Undefined) GetCategory() string {
	return model.FunctionCategoryCore
}

// RunRule will execute the Undefined rule, based on supplied context and a supplied []*yaml.Node slice.
func (u Undefined) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	message := context.Rule.Message

	pathValue := "unknown"
	if path, ok := context.Given.(string); ok {
		pathValue = path
	}

	ruleMessage := context.Rule.Description

	for _, node := range nodes {

		result := vacuumUtils.FindFieldPath(context.RuleAction.Field, node.Content, vacuumUtils.FieldPathOptions{RecursiveFirstSegment: true})
		fieldNode, fieldNodeValue := result.KeyNode, result.ValueNode
		if fieldNode != nil {
			var val = ""
			if context.RuleAction.Field != "" {
				val = fmt.Sprintf("'%s' ", context.RuleAction.Field)
			}

			var locatedObjects []v3.Foundational
			var allPaths []string
			var err error
			locatedPath := pathValue
			if context.DrDocument != nil {
				locatedObjects, err = context.DrDocument.LocateModelsByKeyAndValue(fieldNode, fieldNodeValue)
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
				Message: vacuumUtils.SuppliedOrDefault(message, fmt.Sprintf("%s: `%s` must be undefined",
					ruleMessage, val)),
				StartNode: fieldNode,
				EndNode:   vacuumUtils.BuildEndNode(fieldNode),
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
