// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	v3 "github.com/pb33f/doctor/model/high/v3"
	"gopkg.in/yaml.v3"
)

type ArrayLimit struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the DefineError rule.
func (ar ArrayLimit) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "owaspArrayLimit"}
}

// GetCategory returns the category of the ArrayLimit rule.
func (ar ArrayLimit) GetCategory() string {
	return model.FunctionCategoryOWASP
}

// RunRule will execute the DefineError rule, based on supplied context and a supplied []*yaml.Node slice.
func (ar ArrayLimit) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	for _, schema := range context.DrDocument.Schemas {
		if slices.Contains(schema.Value.Type, "array") {
			if schema.Value.MaxItems == nil {
				node := schema.Value.GoLow().Type.KeyNode

				var direction = utils.GetSchemaDirection(context.DrDocument.V3Document.Document, schema.Name)

				if direction != utils.DirectionRequest && direction != utils.DirectionBoth {
					continue
				}

				result := model.RuleFunctionResult{
					Message:   utils.SuppliedOrDefault(context.Rule.Message, "schema of type `array` must specify `maxItems`"),
					StartNode: node,
					EndNode:   utils.BuildEndNode(node),
					Path:      schema.GenerateJSONPath(),
					Rule:      context.Rule,
				}
				schema.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
				results = append(results, result)
			}
		}
	}
	return results
}
