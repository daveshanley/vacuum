// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"go.yaml.in/yaml/v4"
	"slices"
)

type IntegerFormat struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the DefineError rule.
func (i IntegerFormat) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "owaspIntegerFormat"}
}

// GetCategory returns the category of the IntegerFormat rule.
func (i IntegerFormat) GetCategory() string {
	return model.FunctionCategoryOWASP
}

// RunRule will execute the DefineError rule, based on supplied context and a supplied []*yaml.Node slice.
func (i IntegerFormat) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	for _, schema := range context.DrDocument.Schemas {
		if slices.Contains(schema.Value.Type, "integer") {
			if schema.Value.Format == "" ||
				(schema.Value.Format != "int32" && schema.Value.Format != "int64") {

				node := schema.Value.GoLow().Type.KeyNode
				valueNode := schema.Value.GoLow().Type.ValueNode

				// Find all locations where this schema appears
				locatedPath, allPaths := LocateSchemaPropertyPaths(context, schema, node, valueNode)

				result := model.RuleFunctionResult{
					Message: vacuumUtils.SuppliedOrDefault(context.Rule.Message,
						"schema of type `integer` must specify a format of `int32` or `int64`"),
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
				continue
			}
		}
	}
	return results
}
