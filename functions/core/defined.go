// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package core

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/libopenapi/utils"
	"go.yaml.in/yaml/v4"
)

// Defined is a rule that will determine if a field has been set on a node slice.
type Defined struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the Defined rule.
func (d Defined) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name:          "defined",
		RequiresField: true,
	}
}

// GetCategory returns the category of the Defined rule.
func (d Defined) GetCategory() string {
	return model.FunctionCategoryCore
}

// RunRule will execute the Defined rule, based on supplied context and a supplied []*yaml.Node slice.
func (d Defined) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

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
	if context.Rule.Message != "" {
		ruleMessage = context.Rule.Message
	}

	for _, node := range nodes {
		fieldNode, _ := utils.FindKeyNode(context.RuleAction.Field, node.Content)
		var locatedObjects []v3.Foundational
		var allPaths []string
		var err error

		if fieldNode == nil {

			locatedPath := pathValue
			if context.DrDocument != nil {
				// Since the field is undefined, locate the parent node to be the locatedPath of infraction
				locatedObjects, err = context.DrDocument.LocateModel(node)
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
				Message: vacuumUtils.SuppliedOrDefault(message,
					fmt.Sprintf("%s: `%s` must be defined", ruleMessage, context.RuleAction.Field)),
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
