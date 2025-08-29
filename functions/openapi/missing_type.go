// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package openapi

import (
	"fmt"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"gopkg.in/yaml.v3"
)

// MissingType will check that all schemas and their properties have a type defined
type MissingType struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the MissingType rule.
func (mt MissingType) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "missingType",
	}
}

// GetCategory returns the category of the MissingType rule.
func (mt MissingType) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the MissingType rule, based on supplied context and a supplied []*yaml.Node slice.
func (mt MissingType) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if context.DrDocument == nil {
		return nil
	}

	var results []model.RuleFunctionResult

	// Process all schemas in the document - DrDocument.Schemas contains ALL schemas including nested ones
	schemas := context.DrDocument.Schemas

	for _, schema := range schemas {
		// Check if the schema itself has a type
		schemaResults := mt.checkSchemaType(schema, &context)
		results = append(results, schemaResults...)

		// Check properties of this schema (no recursion needed)
		if schema.Value.Properties != nil {
			propResults := mt.checkProperties(schema, &context)
			results = append(results, propResults...)
		}
	}

	return results
}

func (mt MissingType) checkSchemaType(schema *v3.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult

	// Don't require type for polymorphic schemas themselves, but we check their nested schemas
	// in checkSchemaRecursive
	if schema.Value.AllOf != nil || schema.Value.OneOf != nil || schema.Value.AnyOf != nil {
		return results
	}

	// Skip if this is a boolean schema or has a const/enum (these don't require explicit type)
	if schema.Value.Const != nil || schema.Value.Enum != nil {
		return results
	}

	// Check if the schema has a type defined
	if len(schema.Value.Type) == 0 {
		// Check if this schema has properties, items, or additionalProperties
		// These imply a type even if not explicitly stated
		if schema.Value.Properties != nil || schema.Value.Items != nil ||
			schema.Value.AdditionalProperties != nil || schema.Value.PatternProperties != nil {
			// Schema structure implies a type, so we can skip this warning
			return results
		}

		// Find all locations where this schema appears
		node := schema.Value.GoLow().RootNode
		if node == nil {
			return results
		}

		locatedPath, allPaths := vacuumUtils.LocateSchemaPropertyPaths(*context, schema, node, node)

		result := model.RuleFunctionResult{
			Message:   vacuumUtils.SuppliedOrDefault(context.Rule.Message, "schema is missing a `type` field"),
			StartNode: node,
			EndNode:   vacuumUtils.BuildEndNode(node),
			Path:      locatedPath,
			Rule:      context.Rule,
		}

		// Always set the Paths array
		if len(allPaths) > 0 {
			result.Paths = allPaths
		} else {
			result.Paths = []string{locatedPath}
		}

		schema.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
		results = append(results, result)
	}

	return results
}

func (mt MissingType) checkProperties(schema *v3.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult

	if schema.Value.Properties == nil {
		return results
	}

	for propName, propSchemaProxy := range schema.Value.Properties.FromOldest() {
		// Get the actual schema from the proxy
		propSchema := propSchemaProxy.Schema()
		if propSchema == nil {
			continue
		}

		// Skip if this property is a reference or polymorphic schema
		if propSchema.AllOf != nil || propSchema.OneOf != nil || propSchema.AnyOf != nil {
			continue
		}

		// Skip if this property has const or enum (these don't require explicit type)
		if propSchema.Const != nil || propSchema.Enum != nil {
			continue
		}

		// Check if the property has a type defined
		if len(propSchema.Type) == 0 {
			// Check if this property has nested properties, items, or additionalProperties
			// These imply a type even if not explicitly stated
			if propSchema.Properties != nil || propSchema.Items != nil ||
				propSchema.AdditionalProperties != nil || propSchema.PatternProperties != nil {
				// Property structure implies a type, so we can skip this warning
				continue
			}

			// Get the property node
			propNode := propSchema.GoLow().RootNode
			if propNode == nil {
				continue
			}

			// Build the path to this property
			schemaPath := schema.GenerateJSONPath()
			propertiesPath := model.GetStringTemplates().BuildJSONPath(schemaPath, "properties")
			propertyPath := fmt.Sprintf("%s['%s']", propertiesPath, propName)

			// Use a simple paths array with just the property path
			finalPaths := []string{propertyPath}

			result := model.RuleFunctionResult{
				Message: vacuumUtils.SuppliedOrDefault(context.Rule.Message,
					model.GetStringTemplates().BuildFieldMessage("schema property", propName, "is missing a `type` field")),
				StartNode: propNode,
				EndNode:   vacuumUtils.BuildEndNode(propNode),
				Path:      propertyPath,
				Rule:      context.Rule,
				Paths:     finalPaths,
			}

			results = append(results, result)
		}
	}

	return results
}
