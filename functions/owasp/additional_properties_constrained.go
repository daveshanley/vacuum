// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
    "github.com/daveshanley/vacuum/model"
    "github.com/daveshanley/vacuum/utils"
    "github.com/pb33f/doctor/model/high/v3"
    "gopkg.in/yaml.v3"
    "slices"
)

type AdditionalPropertiesConstrained struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the DefineError rule.
func (ad AdditionalPropertiesConstrained) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "owaspNoAdditionalPropertiesConstrained"}
}

// GetCategory returns the category of the DefineError rule.
func (ad AdditionalPropertiesConstrained) GetCategory() string {
	return model.FunctionCategoryOWASP
}

// RunRule will execute the DefineError rule, based on supplied context and a supplied []*yaml.Node slice.
func (ad AdditionalPropertiesConstrained) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	for _, schema := range context.DrDocument.Schemas {
		if slices.Contains(schema.Value.Type, "object") {
			if schema.Value.AdditionalProperties != nil {

				node := schema.Value.GoLow().Type.KeyNode
				valueNode := schema.Value.GoLow().Type.ValueNode
				
				// Find all locations where this schema appears
				locatedPath, allPaths := LocateSchemaPropertyPaths(context, schema, node, valueNode)
				
				result := model.RuleFunctionResult{
					Message: utils.SuppliedOrDefault(context.Rule.Message,
						"schema should also define `maxProperties` when `additionalProperties` is an object"),
					StartNode: node,
					EndNode:   utils.BuildEndNode(node),
					Path:      locatedPath,
					Rule:      context.Rule,
				}
				
				// Set the Paths array if there are multiple locations
				if len(allPaths) > 1 {
					result.Paths = allPaths
				}

				if schema.Value.AdditionalProperties.IsA() {
					if schema.Value.MaxProperties == nil {

						schema.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
						results = append(results, result)
						continue
					}
				}
				if schema.Value.AdditionalProperties.IsB() && schema.Value.AdditionalProperties.B {

					if schema.Value.MaxProperties == nil {

						schema.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
						results = append(results, result)
						continue
					}
				}

			}
		}
	}
	return results
}
