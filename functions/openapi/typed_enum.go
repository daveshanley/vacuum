// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"strconv"
)

// TypedEnum will check enum values match the types provided
type TypedEnum struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the TypedEnum rule.
func (te TypedEnum) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "typed_enum",
	}
}

// RunRule will execute the TypedEnum rule, based on supplied context and a supplied []*yaml.Node slice.
func (te TypedEnum) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	enums := context.Index.GetAllEnums()

	for _, enum := range enums {

		enumType := enum.Type.Value
		enumDataNode := enum.Node

		// extract types into an array and have them checked against the spec.
		var typeArray []interface{}
		for _, dn := range enumDataNode.Content {
			if utils.IsNodeStringValue(dn) {
				typeArray = append(typeArray, dn.Value)
			}
			if utils.IsNodeIntValue(dn) {
				i, _ := strconv.ParseInt(dn.Value, 10, 64)
				typeArray = append(typeArray, i)
			}
			if utils.IsNodeBoolValue(dn) {
				b, _ := strconv.ParseBool(dn.Value)
				typeArray = append(typeArray, b)
			}
			if utils.IsNodeFloatValue(dn) {
				f, _ := strconv.ParseFloat(dn.Value, 64)
				typeArray = append(typeArray, f)
			}
		}

		typeResults := datamodel.AreValuesCorrectlyTyped(enumType, typeArray)

		startNode := enum.Node
		endNode := enum.Node

		// iterate through type results and add to rule output.
		for _, res := range typeResults {
			results = append(results, model.RuleFunctionResult{
				Message:   fmt.Sprintf("enum type mismatch: %s", res),
				StartNode: startNode,
				EndNode:   endNode,
				Path:      fmt.Sprintf("%v", context.Given),
				Rule:      context.Rule,
			})
		}

	}

	return results
}
