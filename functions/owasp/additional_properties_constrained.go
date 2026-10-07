// Copyright 2023 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/go-yaml"
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
		if schemaHasFiniteValues(value, context.SpecInfo) {
			continue
		}
		if !schemaIsObject(value) {
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
		node, valueNode := schemaObjectNodes(value)
		locatedPath, allPaths := LocateSchemaPropertyPaths(context, schema, node, valueNode)
		result := model.RuleFunctionResult{
			Message:   utils.SuppliedOrDefault(context.Rule.Message, "request maps should define `maxProperties` to limit the number of properties"),
			StartNode: node,
			EndNode:   utils.BuildEndNode(node),
			Path:      locatedPath,
			Rule:      context.Rule,
		}
		if len(allPaths) > 1 {
			result.Paths = allPaths
		}
		schema.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
		results = append(results, result)
	}
	return results
}
