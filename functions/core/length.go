// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package core

import (
    "github.com/daveshanley/vacuum/model"
    vacuumUtils "github.com/daveshanley/vacuum/utils"
    "github.com/pb33f/doctor/model/high/v3"
    "github.com/pb33f/libopenapi/utils"
    "gopkg.in/yaml.v3"
    "strconv"
)

// TODO: come back and reduce the amount of code in here, it's not very efficient.

// Length is a rule that will determine if nodes meet a 'min' or 'max' size. It checks arrays, strings and maps.
type Length struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the Length rule.
func (l Length) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "length",
		Properties: []model.RuleFunctionProperty{
			{
				Name:        "min",
				Description: "'length' requires minimum value to check against",
			},
			{
				Name:        "max",
				Description: "'length' needs a maximum value to check against",
			},
		},
		MinProperties: 1,
		MaxProperties: 2,
		ErrorMessage:  "'length' needs 'min' or 'max' (or both) function options being set to operate",
	}
}

// GetCategory returns the category of the Length rule.
func (l Length) GetCategory() string {
	return model.FunctionCategoryCore
}

// RunRule will execute the Length rule, based on supplied context and a supplied []*yaml.Node slice.
func (l Length) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult
	if len(nodes) <= 0 {
		return nil
	}

	var minVal int
	var maxVal int

	// check if there are min and max values
	if context.Options == nil {
		return results
	}
	if opts := utils.ConvertInterfaceIntoStringMap(context.Options); opts != nil {
		if v, ok := opts["min"]; ok {
			minVal, _ = strconv.Atoi(v)
		}
		if v, ok := opts["max"]; ok {
			maxVal, _ = strconv.Atoi(v)
		}
	}

	ruleMessage := context.Rule.Description
	if context.Rule.Message != "" {
		ruleMessage = context.Rule.Message
	}

	if minVal == 0 && maxVal == 0 {
		return results
	}

	// run through nodes
	for _, node := range nodes {

		var p, q *yaml.Node

		// check field type, is it a map? is it an array?
		if context.RuleAction.Field != "" {
			q, p = utils.FindFirstKeyNode(context.RuleAction.Field, []*yaml.Node{node}, 0)

			// no luck? try again.
			if p == nil {
				q, p = utils.FindKeyNode(context.RuleAction.Field, []*yaml.Node{node})
			}
		} else {
			p = node
		}

		if p == nil {
			continue
		}

		pathValue := "unknown"
		if path, ok := context.Given.(string); ok {
			pathValue = path
		}

		var locatedObjects []v3.Foundational
		var allPaths []string
		var err error
		locatedPath := pathValue
		if context.DrDocument != nil {
			locatedObjects, err = context.DrDocument.LocateModelsByKeyAndValue(q, p)
			if err == nil && locatedObjects != nil {
				for x, obj := range locatedObjects {
					if x == 0 {
						locatedPath = obj.GenerateJSONPath()
					}
					allPaths = append(allPaths, obj.GenerateJSONPath())
				}
			}
		}

		// check for value lengths.
		if utils.IsNodeStringValue(p) || utils.IsNodeIntValue(p) || utils.IsNodeFloatValue(p) {

			var valueCheck int
			if utils.IsNodeStringValue(p) {
				valueCheck = len(p.Value)
			}
			if utils.IsNodeIntValue(p) {
				valueCheck, _ = strconv.Atoi(p.Value)
			}

			// floats can't be boiled nicely into an int, so this logic branches a little.
			// there is no value is trying boil off accuracy, so lets keep it accurate.
			if utils.IsNodeFloatValue(p) {
				fValue, _ := strconv.ParseFloat(p.Value, 64)
				if float64(minVal) > 0 && fValue < float64(minVal) {
					res := createMinError(ruleMessage, p.Value, minVal)
					res.StartNode = node
					res.EndNode = vacuumUtils.BuildEndNode(node)
					res.Path = locatedPath
					res.Rule = context.Rule
					if len(allPaths) > 1 {
						res.Paths = allPaths
					}
					results = append(results, res)
					if len(locatedObjects) > 0 {
						if arr, ok := locatedObjects[0].(v3.AcceptsRuleResults); ok {
							arr.AddRuleFunctionResult(v3.ConvertRuleResult(&res))
						}
					}
					continue
				}
				if float64(maxVal) > 0 && fValue > float64(maxVal) {
					res := createMaxError(ruleMessage, p.Value, maxVal)
					res.StartNode = node
					res.EndNode = vacuumUtils.BuildEndNode(node)
					res.Path = locatedPath
					res.Rule = context.Rule
					if len(allPaths) > 1 {
						res.Paths = allPaths
					}
					results = append(results, res)
					if len(locatedObjects) > 0 {
						if arr, ok := locatedObjects[0].(v3.AcceptsRuleResults); ok {
							arr.AddRuleFunctionResult(v3.ConvertRuleResult(&res))
						}
					}
					continue
				}
			}

			if minVal > 0 && valueCheck < minVal {
				res := createMinError(ruleMessage, p.Value, minVal)
				res.StartNode = node
				res.EndNode = vacuumUtils.BuildEndNode(node)
				res.Path = locatedPath
				res.Rule = context.Rule
				if len(allPaths) > 1 {
					res.Paths = allPaths
				}
				results = append(results, res)
				continue
			}
			if maxVal > 0 && valueCheck > maxVal {
				res := createMaxError(ruleMessage, p.Value, maxVal)
				res.StartNode = node
				res.EndNode = vacuumUtils.BuildEndNode(node)
				res.Path = locatedPath
				res.Rule = context.Rule
				if len(allPaths) > 1 {
					res.Paths = allPaths
				}
				results = append(results, res)
				if len(locatedObjects) > 0 {
					if arr, ok := locatedObjects[0].(v3.AcceptsRuleResults); ok {
						arr.AddRuleFunctionResult(v3.ConvertRuleResult(&res))
					}
				}
				continue
			}
		} else {

			nodeCount := 0

			if utils.IsNodeMap(p) {
				// AST uses sequential ordering for maps, doubling the size essentially.
				nodeCount = len(p.Content) / 2
			}

			if utils.IsNodeArray(p) {
				nodeCount = len(p.Content)
			}

			// check for structure sizes (maps and arrays)
			if minVal > 0 && nodeCount < minVal {
				var fv string
				if context.RuleAction.Field != "" {
					fv = context.RuleAction.Field
				} else {
					fv = context.Rule.Given.(string)
				}
				res := createMinError(ruleMessage, fv, minVal)
				res.StartNode = node
				res.EndNode = vacuumUtils.BuildEndNode(node)
				res.Path = locatedPath
				res.Rule = context.Rule
				if len(allPaths) > 1 {
					res.Paths = allPaths
				}
				results = append(results, res)
				if len(locatedObjects) > 0 {
					if arr, ok := locatedObjects[0].(v3.AcceptsRuleResults); ok {
						arr.AddRuleFunctionResult(v3.ConvertRuleResult(&res))
					}
				}
				results = model.MapPathAndNodesToResults(pathValue, p, p, results)
				continue
			}

			if maxVal > 0 && nodeCount > maxVal {
				var fv string
				if context.RuleAction.Field != "" {
					fv = context.RuleAction.Field
				} else {
					fv = context.Rule.Given.(string)
				}
				res := createMaxError(ruleMessage, fv, maxVal)
				res.StartNode = node
				res.EndNode = vacuumUtils.BuildEndNode(node)
				res.Path = locatedPath
				res.Rule = context.Rule
				if len(allPaths) > 1 {
					res.Paths = allPaths
				}
				results = append(results, res)
				if len(locatedObjects) > 0 {
					if arr, ok := locatedObjects[0].(v3.AcceptsRuleResults); ok {
						arr.AddRuleFunctionResult(v3.ConvertRuleResult(&res))
					}
				}
				//results = model.MapPathAndNodesToResults(pathValue, p, p, results)
				continue
			}
		}

	}

	return results
}

func createMaxError(desc, field string, max int) model.RuleFunctionResult {
	return model.BuildFunctionResultWithDescription(desc, field, "must not be longer/greater than", max)
}

func createMinError(desc, field string, min int) model.RuleFunctionResult {
	return model.BuildFunctionResultWithDescription(desc, field, "must be longer/greater than", min)
}
