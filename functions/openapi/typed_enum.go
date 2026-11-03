// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"strconv"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
)

// TypedEnum will check enum values match the types provided
type TypedEnum struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the TypedEnum rule.
func (te TypedEnum) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "typedEnum",
	}
}

// GetCategory returns the category of the TypedEnum rule.
func (te TypedEnum) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the TypedEnum rule, based on supplied context and a supplied []*yaml.Node slice.
func (te TypedEnum) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	// the root index only holds enums from the root document, enums defined in files reached
	// through a $ref live in the rolodex indexes.
	results = te.checkIndex(context.Index, context, results)
	if rolodex := context.Index.GetRolodex(); rolodex != nil {
		for _, idx := range rolodex.GetIndexes() {
			results = te.checkIndex(idx, context, results)
		}
	}
	return results
}

func (te TypedEnum) checkIndex(idx *index.SpecIndex, context model.RuleFunctionContext, results []model.RuleFunctionResult) []model.RuleFunctionResult {
	for _, enum := range idx.GetAllEnums() {
		var enumType string
		if enum.Type.Value != "" {
			enumType = enum.Type.Value
		} else if len(enum.Type.Content) == 2 {
			for _, n := range enum.Type.Content {
				if n.Value != "null" {
					enumType = n.Value
				}
			}
		}

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

		typeResults := utils.AreValuesCorrectlyTyped(enumType, typeArray)

		startNode := enum.Node
		endNode := vacuumUtils.BuildEndNode(enum.Node)

		// iterate through type results and add to rule output.
		var origin *index.NodeOrigin
		if idx != context.Index && len(typeResults) > 0 {
			origin = idx.FindNodeOrigin(startNode)
		}
		for _, res := range typeResults {
			results = append(results, model.RuleFunctionResult{
				Message:   fmt.Sprintf("enum type mismatch: %s", res),
				StartNode: startNode,
				EndNode:   endNode,
				Path:      fmt.Sprintf("%v", context.Given),
				Rule:      context.Rule,
				Origin:    origin,
			})
		}

	}

	return results
}
