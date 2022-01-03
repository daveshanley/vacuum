// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package core

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"gopkg.in/yaml.v3"
	"strconv"
)

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
		ErrorMessage:  "'length' needs 'min' or 'max' (or both) properties being set to operate",
	}
}

// RunRule will execute the Length rule, based on supplied context and a supplied []*yaml.Node slice.
func (l Length) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult
	if len(nodes) <= 0 {
		return nil
	}

	var min int
	var max int

	// check if there are min and max values
	if context.Options == nil {
		return results
	}
	if opts := utils.ConvertInterfaceIntoStringMap(context.Options); opts != nil {
		if v, ok := opts["min"]; ok {
			min, _ = strconv.Atoi(v)
		}
		if v, ok := opts["max"]; ok {
			max, _ = strconv.Atoi(v)
		}
	} else {
		return results // can't do much without a min or a max.
	}

	// run through nodes
	for _, node := range nodes {

		var p *yaml.Node

		// check field type, is it a map? is it an array?
		if context.RuleAction.Field != "" {
			_, p = utils.FindFirstKeyNode(context.RuleAction.Field, []*yaml.Node{node}, 0)

			// no luck? try again.
			if p == nil {
				_, p = utils.FindKeyNode(context.RuleAction.Field, []*yaml.Node{node})
			}
		} else {

			if p == nil {
				p = node
			}
		}

		if p == nil {
			continue
		}

		pathValue := "unknown"
		if path, ok := context.Given.(string); ok {
			pathValue = path
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
				if float64(min) > 0 && fValue < float64(min) {
					res := createMinError(p.Value, min)
					res.StartNode = node
					res.EndNode = node
					res.Path = pathValue
					results = append(results, res)
					continue
				}
				if float64(max) > 0 && fValue > float64(max) {
					res := createMaxError(p.Value, max)
					res.StartNode = node
					res.EndNode = node
					res.Path = pathValue
					results = append(results, res)
					continue
				}
			}

			if min > 0 && valueCheck < min {
				res := createMinError(p.Value, min)
				res.StartNode = node
				res.EndNode = node
				res.Path = pathValue
				results = append(results, res)
				continue
			}
			if max > 0 && valueCheck > max {
				res := createMaxError(p.Value, max)
				res.StartNode = node
				res.EndNode = node
				res.Path = pathValue
				results = append(results, res)
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
			if min > 0 && nodeCount < min {
				if context.RuleAction.Field != "" {
					res := createMinError(context.RuleAction.Field, min)
					res.StartNode = node
					res.EndNode = node
					res.Path = pathValue
					results = append(results, res)
				} else {
					res := createMinError(context.Rule.Given.(string), min)
					res.StartNode = node
					res.EndNode = node
					res.Path = pathValue
					results = append(results, res)
				}
				results = model.MapPathAndNodesToResults(pathValue, p, p, results)
				continue
			}

			if max > 0 && nodeCount > max {
				if context.RuleAction.Field != "" {
					res := createMaxError(context.RuleAction.Field, max)
					res.StartNode = node
					res.EndNode = node
					res.Path = pathValue
					results = append(results, res)
				} else {
					res := createMaxError(context.Rule.Given.(string), max)
					res.StartNode = node
					res.EndNode = node
					res.Path = pathValue
					results = append(results, res)

				}

				continue
			}
		}

	}

	return results
}

func createMaxError(field string, max int) model.RuleFunctionResult {
	return model.BuildFunctionResult(field, "must not be longer/greater than", max)
}

func createMinError(field string, min int) model.RuleFunctionResult {
	return model.BuildFunctionResult(field, "must be longer/greater than", min)
}
