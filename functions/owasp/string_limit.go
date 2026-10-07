// Copyright 2023 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	v3 "github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/go-yaml"
)

type StringLimit struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the StringLimit rule.
func (st StringLimit) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "owaspStringLimit"}
}

// GetCategory returns the category of the StringLimit rule.
func (st StringLimit) GetCategory() string {
	return model.FunctionCategoryOWASP
}

// RunRule checks request schemas for missing string limits.
func (st StringLimit) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil || context.DrDocument.V3Document == nil {
		return results
	}

	// Schema names recur across properties and references within one document.
	directions := utils.GetSchemaNodeDirections(context.DrDocument.V3Document.Document)
	for _, schema := range context.DrDocument.Schemas {
		if slices.Contains(schema.Value.Type, "string") {
			if schema.Value.MaxLength == nil && schema.Value.Const == nil && schema.Value.Enum == nil {
				direction := directions[schema.Value.GoLow().RootNode]
				if direction != utils.DirectionRequest && direction != utils.DirectionBoth {
					continue
				}

				node := schema.Value.GoLow().Type.KeyNode
				valueNode := schema.Value.GoLow().Type.ValueNode
				locatedPath, allPaths := LocateSchemaPropertyPaths(context, schema, node, valueNode)

				result := model.RuleFunctionResult{
					Message: utils.SuppliedOrDefault(context.Rule.Message,
						"schema of type `string` must specify `maxLength`, `const` or `enum`"),
					StartNode: node,
					EndNode:   utils.BuildEndNode(node),
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
