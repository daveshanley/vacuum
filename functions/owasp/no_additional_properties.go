// Copyright 2023 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/go-yaml"
)

// NoAdditionalProperties checks request objects for unrestricted extra fields.
type NoAdditionalProperties struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the NoAdditionalProperties rule.
func (na NoAdditionalProperties) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "owaspNoAdditionalProperties"}
}

// GetCategory returns the category of the NoAdditionalProperties rule.
func (na NoAdditionalProperties) GetCategory() string {
	return model.FunctionCategoryOWASP
}

// RunRule checks request object schemas for unrestricted additional properties.
func (na NoAdditionalProperties) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	if context.DrDocument == nil || context.DrDocument.V3Document == nil {
		return nil
	}
	composedOnly := composedOnlySchemas(context)
	var results []model.RuleFunctionResult
	var directions map[*yaml.Node]vacuumUtils.DirectionType
	for _, schema := range context.DrDocument.Schemas {
		value := schema.Value
		if schemaHasFiniteValues(value, context.SpecInfo) {
			continue
		}
		if !schemaIsObject(value) {
			continue
		}
		properties := value.AdditionalProperties
		if properties != nil && (properties.IsA() || !properties.B) {
			continue
		}
		if properties == nil {
			if composedOnly[value.GoLow().RootNode] {
				continue
			}
			// An explicit unevaluatedProperties constraint can close an OAS 3.1 object.
			if !vacuumUtils.IsOAS30(context.SpecInfo) && value.UnevaluatedProperties != nil && (value.UnevaluatedProperties.IsA() || !value.UnevaluatedProperties.B) {
				continue
			}
			// Composition branches constrain the same instance. Missing closure here
			// does not prove the complete object is open; do not require closure per arm.
			if len(value.AllOf) > 0 || len(value.AnyOf) > 0 || len(value.OneOf) > 0 {
				continue
			}
			if proxy, ok := schema.GetParent().(*v3.SchemaProxy); ok {
				switch proxy.GetPathSegment() {
				case "allOf", "anyOf", "oneOf", "if", "then", "else", "not", "dependentSchemas":
					continue
				}
			}
		}
		if directions == nil {
			directions = schemaNodeDirections(context)
		}
		direction := directions[value.GoLow().RootNode]
		if direction != vacuumUtils.DirectionRequest && direction != vacuumUtils.DirectionBoth {
			continue
		}
		node, valueNode := schemaObjectNodes(value)
		locatedPath, allPaths := LocateSchemaPropertyPaths(context, schema, node, valueNode)
		result := model.RuleFunctionResult{
			Message:   vacuumUtils.SuppliedOrDefault(context.Rule.Message, "request objects should set `additionalProperties` to `false` or define a schema for additional values"),
			StartNode: node,
			EndNode:   vacuumUtils.BuildEndNode(node),
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
