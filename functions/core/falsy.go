// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package core

import (
	"fmt"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/utils"
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

		fieldNode, fieldNodeValue := utils.FindKeyNode(context.RuleAction.Field, node.Content)
		if (fieldNode != nil && fieldNodeValue != nil) &&
			(fieldNodeValue.Value != "" && fieldNodeValue.Value != "false" && fieldNodeValue.Value != "0" || (fieldNodeValue.Value == "" && fieldNodeValue.Content != nil)) {

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
				Message:   fmt.Sprintf("%s: `%s` must be falsy", ruleMessage, context.RuleAction.Field),
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
