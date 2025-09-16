// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/libopenapi/utils"
	"go.yaml.in/yaml/v4"
)

// DuplicatedEnum will check enum values match the types provided
type DuplicatedEnum struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the DuplicatedEnum rule.
func (de DuplicatedEnum) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "duplicatedEnum",
	}
}

// GetCategory returns the category of the DuplicatedEnum rule.
func (de DuplicatedEnum) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the DuplicatedEnum rule, based on supplied context and a supplied []*yaml.Node slice.
func (de DuplicatedEnum) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if context.DrDocument == nil {
		return nil
	}

	var results []model.RuleFunctionResult

	schemas := context.DrDocument.Schemas

	for _, schema := range schemas {

		if schema.Value.Enum != nil {
			node := schema.Value.GoLow().Enum.KeyNode

			duplicates := utils.CheckEnumForDuplicates(schema.Value.Enum)

			// iterate through duplicate results and add results.
			for _, res := range duplicates {
				// Find all locations where this schema appears
				locatedPath, allPaths := vacuumUtils.LocateSchemaPropertyPaths(context, schema,
					schema.Value.GoLow().Type.KeyNode, schema.Value.GoLow().Type.ValueNode)

				result := model.RuleFunctionResult{
					Message:   fmt.Sprintf("enum contains a duplicate: `%s`", res.Value),
					StartNode: node,
					EndNode:   vacuumUtils.BuildEndNode(node),
					Path:      fmt.Sprintf("%s.%s", locatedPath, "enum"),
					Rule:      context.Rule,
				}

				// Set the Paths array if there are multiple locations
				if len(allPaths) > 1 {
					// Add .enum suffix to all paths
					enumPaths := make([]string, len(allPaths))
					for i, p := range allPaths {
						enumPaths[i] = fmt.Sprintf("%s.%s", p, "enum")
					}
					result.Paths = enumPaths
				}

				schema.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
				results = append(results, result)
			}
		}
	}

	return results
}
