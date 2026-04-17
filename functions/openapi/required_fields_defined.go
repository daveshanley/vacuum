// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/doctor/model/high/v3"
	"go.yaml.in/yaml/v4"
)

// RequiredFieldsDefined checks that explicitly required fields are declared in schema properties.
type RequiredFieldsDefined struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the RequiredFieldsDefined rule.
func (rfd RequiredFieldsDefined) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "requiredFieldsDefined",
	}
}

// GetCategory returns the category of the RequiredFieldsDefined rule.
func (rfd RequiredFieldsDefined) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the RequiredFieldsDefined rule using the walked schema graph.
func (rfd RequiredFieldsDefined) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	if context.DrDocument == nil {
		return nil
	}

	var results []model.RuleFunctionResult
	st := SchemaTypeCheck{}

	for _, schema := range context.DrDocument.Schemas {
		if !schemaUsesObjectKeywords(schema) || len(schema.Value.Required) == 0 {
			continue
		}
		requiredNode := schema.Value.GoLow().Required.KeyNode
		for i, required := range schema.Value.Required {
			propertyLookup := st.lookupRequiredProperty(schema, required)
			if !propertyLookup.propertiesFound {
				result := st.buildResult("object contains `required` fields but no `properties`",
					schema.GenerateJSONPath(), "required", i,
					schema, requiredNode, &context)
				results = append(results, result)
				break
			}

			if !propertyLookup.propertyDefined {
				result := st.buildResult(model.GetStringTemplates().BuildRequiredFieldMessage(required),
					schema.GenerateJSONPath(), "required", i,
					schema, requiredNode, &context)
				results = append(results, result)
			}
		}
	}

	return results
}

func schemaUsesObjectKeywords(schema *v3.Schema) bool {
	if schema == nil || schema.Value == nil {
		return false
	}

	for _, t := range schema.Value.Type {
		if t == "object" {
			return true
		}
	}

	return schema.Value.Properties != nil ||
		len(schema.Value.Required) > 0 ||
		len(schema.AllOf) > 0 ||
		len(schema.AnyOf) > 0 ||
		len(schema.OneOf) > 0
}
