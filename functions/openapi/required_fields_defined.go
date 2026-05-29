// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package openapi

import (
	"github.com/daveshanley/vacuum/functions/schemachecks"
	"github.com/daveshanley/vacuum/model"
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

	for _, schema := range context.DrDocument.Schemas {
		if !schemachecks.SchemaUsesObjectKeywords(schema) || len(schema.Value.Required) == 0 {
			continue
		}
		requiredNode := schema.Value.GoLow().Required.KeyNode
		for i, required := range schema.Value.Required {
			propertyLookup := schemachecks.LookupRequiredProperty(schema, required)
			if !propertyLookup.PropertiesFound {
				result := schemachecks.BuildResult("object contains `required` fields but no `properties`",
					schema.GenerateJSONPath(), "required", i,
					schema, requiredNode, &context)
				results = append(results, result)
				break
			}

			if !propertyLookup.PropertyDefined {
				result := schemachecks.BuildResult(model.GetStringTemplates().BuildRequiredFieldMessage(required),
					schema.GenerateJSONPath(), "required", i,
					schema, requiredNode, &context)
				results = append(results, result)
			}
		}
	}

	return results
}
