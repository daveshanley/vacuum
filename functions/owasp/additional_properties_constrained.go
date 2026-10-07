// Copyright 2023 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/go-yaml"
	"slices"
)

// AdditionalPropertiesConstrained checks request maps for a property count limit.
type AdditionalPropertiesConstrained struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the AdditionalPropertiesConstrained rule.
func (ad AdditionalPropertiesConstrained) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "owaspNoAdditionalPropertiesConstrained"}
}

// GetCategory returns the category of the AdditionalPropertiesConstrained rule.
func (ad AdditionalPropertiesConstrained) GetCategory() string {
	return model.FunctionCategoryOWASP
}

// RunRule requires a property count limit for open request maps.
func (ad AdditionalPropertiesConstrained) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	if context.DrDocument == nil || context.DrDocument.V3Document == nil {
		return nil
	}
	var results []model.RuleFunctionResult
	var directions map[*yaml.Node]utils.DirectionType
	for _, schema := range context.DrDocument.Schemas {
		value := schema.Value
		if (!utils.IsOAS30(context.SpecInfo) && value.Const != nil) || len(value.Enum) > 0 {
			continue
		}
		if !slices.Contains(value.Type, "object") && !(len(value.Type) == 0 && (value.AdditionalProperties != nil || value.UnevaluatedProperties != nil)) {
			continue
		}
		if value.MaxProperties != nil {
			continue
		}
		properties := value.AdditionalProperties
		if properties == nil && !utils.IsOAS30(context.SpecInfo) {
			properties = value.UnevaluatedProperties
		}
		if properties == nil || (properties.IsB() && !properties.B) {
			continue
		}
		if directions == nil {
			directions = schemaNodeDirections(context)
		}
		direction := directions[value.GoLow().RootNode]
		if direction != utils.DirectionRequest && direction != utils.DirectionBoth {
			continue
		}
		node, valueNode := value.GoLow().Type.KeyNode, value.GoLow().Type.ValueNode
		if node == nil {
			node, valueNode = value.GoLow().AdditionalProperties.KeyNode, value.GoLow().AdditionalProperties.ValueNode
		}
		if node == nil {
			node, valueNode = value.GoLow().UnevaluatedProperties.KeyNode, value.GoLow().UnevaluatedProperties.ValueNode
		}
		locatedPath, allPaths := LocateSchemaPropertyPaths(context, schema, node, valueNode)
		result := model.RuleFunctionResult{
			Message:   utils.SuppliedOrDefault(context.Rule.Message, "request maps should define `maxProperties` to limit the number of properties"),
			StartNode: node, EndNode: utils.BuildEndNode(node), Path: locatedPath, Rule: context.Rule,
		}
		if len(allPaths) > 1 {
			result.Paths = allPaths
		}
		schema.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
		results = append(results, result)
	}
	return results
}
