package core

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"gopkg.in/yaml.v3"
	"strconv"
)

type Length struct{}

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
	} else {
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
	}

	// run through nodes
	for _, node := range nodes {

		var p *yaml.Node

		// check field type, is it a map? is it an array?
		if context.RuleAction.Field != "" {
			_, p = utils.FindFirstKeyNode(context.RuleAction.Field, []*yaml.Node{node})

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
					results = append(results, createMinError(p.Value, min))
					continue
				}
				if float64(max) > 0 && fValue > float64(max) {
					results = append(results, createMaxError(p.Value, max))
					continue
				}
			}

			if min > 0 && valueCheck < min {
				results = append(results, createMinError(p.Value, min))
				continue
			}
			if max > 0 && valueCheck > max {
				results = append(results, createMaxError(p.Value, max))
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
					results = append(results, createMinError(context.RuleAction.Field, min))
				} else {

					//results = append(results, createMinError(context.Rule.Given, min))
					results = append(results, createMinError("chicken", min))
				}
				continue
			}

			if max > 0 && nodeCount > max {
				if context.RuleAction.Field != "" {
					results = append(results, createMaxError(context.RuleAction.Field, max))
				} else {
					//results = append(results, createMaxError(context.Rule.Given, max))
					results = append(results, createMaxError("chops", max))
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
