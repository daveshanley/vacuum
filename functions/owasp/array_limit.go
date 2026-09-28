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

type ArrayLimit struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the ArrayLimit rule.
func (ar ArrayLimit) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "owaspArrayLimit"}
}

// GetCategory returns the category of the ArrayLimit rule.
func (ar ArrayLimit) GetCategory() string {
	return model.FunctionCategoryOWASP
}

// RunRule checks request schemas for missing array limits.
func (ar ArrayLimit) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	// Schema names recur across properties and references within one document.
	directions := utils.GetSchemaDirections(context.DrDocument.V3Document.Document)
	for _, schema := range context.DrDocument.Schemas {
		if slices.Contains(schema.Value.Type, "array") {
			if schema.Value.MaxItems == nil {
				direction := directions[schema.Name]
				if direction != utils.DirectionRequest && direction != utils.DirectionBoth {
					continue
				}

				node := schema.Value.GoLow().Type.KeyNode
				valueNode := schema.Value.GoLow().Type.ValueNode
				locatedPath, allPaths := LocateSchemaPropertyPaths(context, schema, node, valueNode)

				result := model.RuleFunctionResult{
					Message:   utils.SuppliedOrDefault(context.Rule.Message, "schema of type `array` must specify `maxItems`"),
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
