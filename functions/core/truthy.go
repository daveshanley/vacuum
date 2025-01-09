// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package core

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/base"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
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

		fieldNode, fieldNodeValue := utils.FindKeyNodeTop(context.RuleAction.Field, node.Content)
		if fieldNode == nil && fieldNodeValue == nil || fieldNodeValue.Value == "false" ||
			fieldNodeValue.Value == "0" || fieldNodeValue.Value == "" {

			if isArray {
				pathValue = fmt.Sprintf("%s[%d]", pathValue, x)
			}

			if !utils.IsNodeMap(fieldNode) && !utils.IsNodeArray(fieldNodeValue) && !utils.IsNodeMap(fieldNodeValue) {
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

						np := nm[origin.Line-1][keys[0]]

						if np != nil {
							node = np
						}
					}
				}

				var locatedObjects []base.Foundational
				var allPaths []string
				var err error
				locatedPath := pathValue
				if context.DrDocument != nil {
					if fieldNode == nil {
						locatedObjects, err = context.DrDocument.LocateModel(node)
					} else {
						locatedObjects, err = context.DrDocument.LocateModelsByKeyAndValue(fieldNode, fieldNodeValue)
					}
					if err == nil && locatedObjects != nil {
						for x, obj := range locatedObjects {
							p := fmt.Sprintf("%s.%s", obj.GenerateJSONPath(), context.RuleAction.Field)
							if x == 0 {
								locatedPath = p
							}
							allPaths = append(allPaths, p)
						}
					}
				}
				result := model.RuleFunctionResult{
					Message: vacuumUtils.SuppliedOrDefault(message,
						fmt.Sprintf("%s: `%s` must be set", ruleMessage, context.RuleAction.Field)),
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
					if arr, ok := locatedObjects[0].(base.AcceptsRuleResults); ok {
						arr.AddRuleFunctionResult(base.ConvertRuleResult(&result))
					}
				}
			}
		}
	}
	return results
}
