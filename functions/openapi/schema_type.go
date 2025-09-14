// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley // https://pb33f.io

package openapi

import (
	"fmt"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/dop251/goja"
	"github.com/pb33f/doctor/model/high/v3"
	"go.yaml.in/yaml/v4"
)

// SchemaTypeCheck will determine if document schemas contain the correct type
type SchemaTypeCheck struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the DuplicatedEnum rule.
func (st SchemaTypeCheck) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "schemaTypeCheck",
	}
}

// GetCategory returns the category of the DuplicatedEnum rule.
func (st SchemaTypeCheck) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the DuplicatedEnum rule, based on supplied context and a supplied []*yaml.Node slice.
func (st SchemaTypeCheck) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if context.DrDocument == nil {
		return nil
	}

	var results []model.RuleFunctionResult

	schemas := context.DrDocument.Schemas

	for _, schema := range schemas {

		schemaType := schema.Value.Type

		for _, t := range schemaType {
			switch t {
			case "string":
				errs := st.validateString(schema, &context)
				results = append(results, errs...)
			case "integer", "number":
				errs := st.validateNumber(schema, &context)
				results = append(results, errs...)
			case "boolean":
				break
			case "array":
				errs := st.validateArray(schema, &context)
				results = append(results, errs...)
			case "object":
				errs := st.validateObject(schema, &context)
				results = append(results, errs...)
			case "null":
			default:
				// Find all locations where this schema appears
				locatedPath, allPaths := vacuumUtils.LocateSchemaPropertyPaths(context, schema,
					schema.Value.GoLow().Type.KeyNode, schema.Value.GoLow().Type.ValueNode)

				result := model.RuleFunctionResult{
					Message:   model.GetStringTemplates().BuildUnknownSchemaTypeMessage(t),
					StartNode: schema.Value.GoLow().Type.KeyNode,
					EndNode:   vacuumUtils.BuildEndNode(schema.Value.GoLow().Type.KeyNode),
					Path:      model.GetStringTemplates().BuildJSONPath(locatedPath, "type"),
					Rule:      context.Rule,
				}

				// Set the Paths array if there are multiple locations
				if len(allPaths) > 1 {
					// Add .type suffix to all paths
					typePaths := make([]string, len(allPaths))
					for i, p := range allPaths {
						typePaths[i] = model.GetStringTemplates().BuildJSONPath(p, "type")
					}
					result.Paths = typePaths
				}

				schema.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
				results = append(results, result)
			}
		}
	}

	return results
}

func (st SchemaTypeCheck) validateNumber(schema *v3.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult

	if schema.Value.MultipleOf != nil {
		if *schema.Value.MultipleOf <= 0 {
			result := st.buildResult("`multipleOf` should be a number greater than `0`",
				schema.GenerateJSONPath(), "multipleOf", -1,
				schema, schema.Value.GoLow().MultipleOf.KeyNode, context)
			results = append(results, result)
		}
	}

	if schema.Value.Maximum != nil {
		if schema.Value.Minimum != nil {
			if *schema.Value.Maximum < *schema.Value.Minimum {
				result := st.buildResult("`maximum` should be a number greater than or equal to `minimum`",
					schema.GenerateJSONPath(), "maximum", -1,
					schema, schema.Value.GoLow().Maximum.KeyNode, context)
				results = append(results, result)
			}
		}
	}

	if schema.Value.ExclusiveMaximum != nil {
		if schema.Value.ExclusiveMinimum != nil {
			if schema.Value.ExclusiveMinimum.IsB() && schema.Value.ExclusiveMinimum.B > schema.Value.ExclusiveMaximum.B {
				result := st.buildResult("`exclusiveMaximum` should be greater than or equal to `exclusiveMinimum`",
					schema.GenerateJSONPath(), "exclusiveMaximum", -1,
					schema, schema.Value.GoLow().ExclusiveMaximum.KeyNode, context)
				results = append(results, result)
			}
		}
	}

	return results
}

func (st SchemaTypeCheck) validateString(schema *v3.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult

	if schema.Value.MinLength != nil {
		if *schema.Value.MinLength < 0 {
			result := st.buildResult("`minLength` should be a non-negative number",
				schema.GenerateJSONPath(), "minLength", -1,
				schema, schema.Value.GoLow().MinLength.KeyNode, context)
			results = append(results, result)
		}
	}

	if schema.Value.MaxLength != nil {
		if *schema.Value.MaxLength < 0 {
			result := st.buildResult("`maxLength` should be a non-negative number",
				schema.GenerateJSONPath(), "maxLength", -1,
				schema, schema.Value.GoLow().MaxLength.KeyNode, context)
			results = append(results, result)
		}
		if schema.Value.MinLength != nil {
			if *schema.Value.MinLength > *schema.Value.MaxLength {
				result := st.buildResult("`maxLength` should be greater than or equal to `minLength`",
					schema.GenerateJSONPath(), "maxLength", -1,
					schema, schema.Value.GoLow().MaxLength.KeyNode, context)
				results = append(results, result)
			}
		}
	}

	if schema.Value.Pattern != "" {
		vm := goja.New()
		script := fmt.Sprintf("const regex = new RegExp(%q)", schema.Value.Pattern)
		_, err := vm.RunString(script)
		if err != nil {
			result := st.buildResult("schema `pattern` should be a ECMA-262 regular expression dialect",
				schema.GenerateJSONPath(), "pattern", -1,
				schema, schema.Value.GoLow().Pattern.KeyNode, context)
			results = append(results, result)
		}
	}
	return results
}

func (st SchemaTypeCheck) validateArray(schema *v3.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult

	if schema.Value.MinItems != nil {
		if *schema.Value.MinItems < 0 {
			result := st.buildResult("`minItems` should be a non-negative number",
				schema.GenerateJSONPath(), "minItems", -1,
				schema, schema.Value.GoLow().MinItems.KeyNode, context)
			results = append(results, result)
		}
	}

	if schema.Value.MaxItems != nil {
		if *schema.Value.MaxItems < 0 {
			result := st.buildResult("`maxItems` should be a non-negative number",
				schema.GenerateJSONPath(), "maxItems", -1,
				schema, schema.Value.GoLow().MaxItems.KeyNode, context)
			results = append(results, result)
		}
		if schema.Value.MinItems != nil {
			if *schema.Value.MinItems > *schema.Value.MaxItems {
				result := st.buildResult("`maxItems` should be greater than or equal to `minItems`",
					schema.GenerateJSONPath(), "maxItems", -1,
					schema, schema.Value.GoLow().MaxItems.KeyNode, context)
				results = append(results, result)
			}
		}
	}

	if schema.Value.MinContains != nil {
		if *schema.Value.MinContains < 0 {
			result := st.buildResult("`minContains` should be a non-negative number",
				schema.GenerateJSONPath(), "minContains", -1,
				schema, schema.Value.GoLow().MinContains.KeyNode, context)
			results = append(results, result)
		}
	}

	if schema.Value.MaxContains != nil {
		if *schema.Value.MaxContains < 0 {
			result := st.buildResult("`maxContains` should be a non-negative number",
				schema.GenerateJSONPath(), "maxContains", -1,
				schema, schema.Value.GoLow().MaxContains.KeyNode, context)
			results = append(results, result)
		}
		if schema.Value.MinContains != nil {
			if *schema.Value.MinContains > *schema.Value.MaxContains {
				result := st.buildResult("`maxContains` should be greater than or equal to `minContains`",
					schema.GenerateJSONPath(), "maxContains", -1,
					schema, schema.Value.GoLow().MaxContains.KeyNode, context)
				results = append(results, result)
			}
		}
	}

	return results
}

func (st SchemaTypeCheck) buildResult(message, path, violationProperty string, segment int, schema *v3.Schema, node *yaml.Node, context *model.RuleFunctionContext) model.RuleFunctionResult {

	// Find all locations where this schema appears
	locatedPath, allPaths := vacuumUtils.LocateSchemaPropertyPaths(*context, schema, node, node)

	// Build the complete path with the violation property
	if violationProperty != "" {
		if segment >= 0 {
			locatedPath = model.GetStringTemplates().BuildPropertyArrayPath(locatedPath, violationProperty, segment)
		} else {
			locatedPath = model.GetStringTemplates().BuildJSONPath(locatedPath, violationProperty)
		}

		// Update all paths with the violation property
		if len(allPaths) > 1 {
			updatedPaths := make([]string, len(allPaths))
			for i, p := range allPaths {
				if segment >= 0 {
					updatedPaths[i] = model.GetStringTemplates().BuildPropertyArrayPath(p, violationProperty, segment)
				} else {
					updatedPaths[i] = model.GetStringTemplates().BuildJSONPath(p, violationProperty)
				}
			}
			allPaths = updatedPaths
		}
	}

	result := model.RuleFunctionResult{
		Message:   message,
		StartNode: node,
		EndNode:   vacuumUtils.BuildEndNode(node),
		Path:      locatedPath,
		Rule:      context.Rule,
	}
	if len(allPaths) > 1 {
		result.Paths = allPaths
	}
	schema.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
	return result
}

func (st SchemaTypeCheck) validateObject(schema *v3.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult

	if schema.Value.MinProperties != nil {
		if *schema.Value.MinProperties < 0 {
			result := st.buildResult("`minProperties` should be a non-negative number",
				schema.GenerateJSONPath(), "minProperties", -1,
				schema, schema.Value.GoLow().MinProperties.KeyNode, context)
			results = append(results, result)
		}
	}

	if schema.Value.MaxProperties != nil {
		if *schema.Value.MaxProperties < 0 {
			result := st.buildResult("`maxProperties` should be a non-negative number",
				schema.GenerateJSONPath(), "maxProperties", -1,
				schema, schema.Value.GoLow().MaxProperties.KeyNode, context)
			results = append(results, result)
		}
		if schema.Value.MinProperties != nil {
			if *schema.Value.MinProperties > *schema.Value.MaxProperties {
				result := st.buildResult("`maxProperties` should be greater than or equal to `minProperties`",
					schema.GenerateJSONPath(), "maxProperties", -1,
					schema, schema.Value.GoLow().MaxProperties.KeyNode, context)
				results = append(results, result)
			}
		}
	}

	if len(schema.Value.Required) > 0 {
		for i, required := range schema.Value.Required {

			// check for polymorphic schema props
			// https://github.com/daveshanley/vacuum/issues/510
			polyFound := false
			polyDefined := false
			if schema.Value.AnyOf != nil || schema.Value.OneOf != nil || schema.Value.AllOf != nil {
				if schema.Value.AnyOf != nil {
					for _, anyOf := range schema.Value.AnyOf {
						if anyOf.Schema() != nil && anyOf.Schema().Properties != nil && anyOf.Schema().Properties.Len() >= 0 {
							polyFound = true
						}
						if anyOf.Schema() != nil && anyOf.Schema().Properties != nil && anyOf.Schema().Properties.GetOrZero(required) != nil {
							polyDefined = true
						}
					}
				}
				if schema.Value.OneOf != nil {
					for _, oneOf := range schema.Value.OneOf {
						if oneOf.Schema() != nil && oneOf.Schema().Properties != nil && oneOf.Schema().Properties.Len() >= 0 {
							polyFound = true
						}
						if oneOf.Schema() != nil && oneOf.Schema().Properties != nil && oneOf.Schema().Properties.GetOrZero(required) != nil {
							polyDefined = true
						}
					}
				}
				if schema.Value.AllOf != nil {
					for _, allOf := range schema.Value.AllOf {
						if allOf.Schema() != nil && allOf.Schema().Properties != nil && allOf.Schema().Properties.Len() >= 0 {
							polyFound = true
						}
						if allOf.Schema() != nil && allOf.Schema().Properties != nil && allOf.Schema().Properties.GetOrZero(required) != nil {
							polyDefined = true
						}
					}
				}
			}
			if schema.Value.Properties == nil && !polyFound {
				result := st.buildResult("object contains `required` fields but no `properties`",
					schema.GenerateJSONPath(), "required", i,
					schema, schema.Value.GoLow().Required.KeyNode, context)
				results = append(results, result)
				break
			}

			// Check if the required field is defined in properties (direct or polymorphic)
			propertyExists := false
			if schema.Value.Properties != nil && schema.Value.Properties.GetOrZero(required) != nil {
				propertyExists = true
			}

			// If not in direct properties, check if it was found in polymorphic schemas
			if !propertyExists && polyDefined {
				propertyExists = true
			}

			// Report error if property is not defined anywhere
			if !propertyExists {
				result := st.buildResult(model.GetStringTemplates().BuildRequiredFieldMessage(required),
					schema.GenerateJSONPath(), "required", i,
					schema, schema.Value.GoLow().Required.KeyNode, context)
				results = append(results, result)
			}
		}
	}

	// Validate DependentRequired
	dependentRequiredResults := st.validateDependentRequired(schema, context)
	results = append(results, dependentRequiredResults...)

	return results
}

func (st SchemaTypeCheck) validateDependentRequired(schema *v3.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult

	// Check if DependentRequired is present
	if schema.Value.DependentRequired == nil {
		return results
	}

	// Iterate through all dependent required entries
	for pair := schema.Value.DependentRequired.First(); pair != nil; pair = pair.Next() {
		triggerProp := pair.Key()
		requiredProps := pair.Value()

		// Validate that trigger property exists in schema properties if properties are defined
		if schema.Value.Properties != nil && schema.Value.Properties.GetOrZero(triggerProp) == nil {
			// Check if the property exists in polymorphic schemas (anyOf, oneOf, allOf)
			polyDefined := st.checkPolymorphicProperty(schema, triggerProp)
			if !polyDefined {
				result := st.buildResult(
					fmt.Sprintf("property `%s` referenced in `dependentRequired` does not exist in schema `properties`", triggerProp),
					schema.GenerateJSONPath(), "dependentRequired", -1,
					schema, schema.Value.GoLow().DependentRequired.KeyNode, context)
				results = append(results, result)
			}
		}

		// Validate that all dependent properties exist in schema
		for _, reqProp := range requiredProps {
			if schema.Value.Properties != nil && schema.Value.Properties.GetOrZero(reqProp) == nil {
				// Check if the property exists in polymorphic schemas
				polyDefined := st.checkPolymorphicProperty(schema, reqProp)
				if !polyDefined {
					result := st.buildResult(
						fmt.Sprintf("property `%s` referenced in `dependentRequired` does not exist in schema `properties`", reqProp),
						schema.GenerateJSONPath(), "dependentRequired", -1,
						schema, schema.Value.GoLow().DependentRequired.KeyNode, context)
					results = append(results, result)
				}
			}
		}

		// Validate no self-referential dependencies
		for _, reqProp := range requiredProps {
			if reqProp == triggerProp {
				result := st.buildResult(
					fmt.Sprintf("circular dependency detected: property `%s` requires itself in `dependentRequired`", triggerProp),
					schema.GenerateJSONPath(), "dependentRequired", -1,
					schema, schema.Value.GoLow().DependentRequired.KeyNode, context)
				results = append(results, result)
			}
		}
	}

	return results
}

// checkPolymorphicProperty checks if a property is defined in anyOf, oneOf, or allOf schemas
func (st SchemaTypeCheck) checkPolymorphicProperty(schema *v3.Schema, propertyName string) bool {
	// Check in AnyOf schemas
	if schema.Value.AnyOf != nil {
		for _, anyOfSchema := range schema.Value.AnyOf {
			if anyOfSchema.Schema() != nil && anyOfSchema.Schema().Properties != nil &&
				anyOfSchema.Schema().Properties.GetOrZero(propertyName) != nil {
				return true
			}
		}
	}

	// Check in OneOf schemas
	if schema.Value.OneOf != nil {
		for _, oneOfSchema := range schema.Value.OneOf {
			if oneOfSchema.Schema() != nil && oneOfSchema.Schema().Properties != nil &&
				oneOfSchema.Schema().Properties.GetOrZero(propertyName) != nil {
				return true
			}
		}
	}

	// Check in AllOf schemas
	if schema.Value.AllOf != nil {
		for _, allOfSchema := range schema.Value.AllOf {
			if allOfSchema.Schema() != nil && allOfSchema.Schema().Properties != nil &&
				allOfSchema.Schema().Properties.GetOrZero(propertyName) != nil {
				return true
			}
		}
	}

	return false
}
