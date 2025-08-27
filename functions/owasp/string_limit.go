// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
    "github.com/daveshanley/vacuum/model"
    vacuumUtils "github.com/daveshanley/vacuum/utils"
    "github.com/pb33f/doctor/model/high/v3"
    "gopkg.in/yaml.v3"
    "slices"
)

type StringLimit struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the DefineError rule.
func (st StringLimit) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "owaspStringLimit"}
}

// GetCategory returns the category of the StringLimit rule.
func (st StringLimit) GetCategory() string {
	return model.FunctionCategoryOWASP
}

// RunRule will execute the DefineError rule, based on supplied context and a supplied []*yaml.Node slice.
func (st StringLimit) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	for _, schema := range context.DrDocument.Schemas {
		if slices.Contains(schema.Value.Type, "string") {
			if schema.Value.MaxLength == nil && schema.Value.Const == nil && schema.Value.Enum == nil {
				node := schema.Value.GoLow().Type.KeyNode
				valueNode := schema.Value.GoLow().Type.ValueNode
				
				// Find all locations where this schema appears
				locatedPath, allPaths := LocateSchemaPropertyPaths(context, schema, node, valueNode)
				
				result := model.RuleFunctionResult{
					Message: vacuumUtils.SuppliedOrDefault(context.Rule.Message,
						"schema of type `string` must specify `maxLength`, `const` or `enum`"),
					StartNode: node,
					EndNode:   vacuumUtils.BuildEndNode(node),
					Path:      locatedPath,
					Rule:      context.Rule,
				}
				
				// Set the Paths array if there are multiple locations
				if len(allPaths) > 1 {
					result.Paths = allPaths
				}
				schema.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
				results = append(results, result)
			}
		}
	}
	return results
}
