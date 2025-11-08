// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley // https://pb33f.io

package openapi

import (
	"fmt"
	"strings"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/dop251/goja"
	"github.com/pb33f/doctor/model/high/v3"
	highBase "github.com/pb33f/libopenapi/datamodel/high/base"
	lowBase "github.com/pb33f/libopenapi/datamodel/low/base"
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
				errs := st.validateBoolean(schema, &context)
				results = append(results, errs...)
			case "array":
				errs := st.validateArray(schema, &context)
				results = append(results, errs...)
			case "object":
				errs := st.validateObject(schema, &context)
				results = append(results, errs...)
			case "null":
			errs := st.validateNull(schema, &context)
				results = append(results, errs...)
			default:
				// find all locations where this schema appears
				locatedPath, allPaths := vacuumUtils.LocateSchemaPropertyPaths(context, schema,
					schema.Value.GoLow().Type.KeyNode, schema.Value.GoLow().Type.ValueNode)

				result := model.RuleFunctionResult{
					Message:   model.GetStringTemplates().BuildUnknownSchemaTypeMessage(t),
					StartNode: schema.Value.GoLow().Type.KeyNode,
					EndNode:   vacuumUtils.BuildEndNode(schema.Value.GoLow().Type.KeyNode),
					Path:      model.GetStringTemplates().BuildJSONPath(locatedPath, "type"),
					Rule:      context.Rule,
				}

				// set the Paths array if there are multiple locations
				if len(allPaths) > 1 {
					// add .type suffix to all paths
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

		// validate const value matches declared types
		if len(schemaType) > 0 {
			constErrs := st.validateConst(schema, &context)
			results = append(results, constErrs...)
		}

		// validate enum and const are not conflicting
		enumConstErrs := st.validateEnumConst(schema, &context)
		results = append(results, enumConstErrs...)

		// validate discriminator property existence
		discriminatorErrs := st.validateDiscriminator(schema, &context)
		results = append(results, discriminatorErrs...)
	}

	return results
}

func (st SchemaTypeCheck) validateNumber(schema *v3.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult

	typeMismatchResults := st.checkTypeMismatchedConstraints(schema, "number", context)
	results = append(results, typeMismatchResults...)

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

	typeMismatchResults := st.checkTypeMismatchedConstraints(schema, "string", context)
	results = append(results, typeMismatchResults...)

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

	typeMismatchResults := st.checkTypeMismatchedConstraints(schema, "array", context)
	results = append(results, typeMismatchResults...)

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

	// find all locations where this schema appears
	locatedPath, allPaths := vacuumUtils.LocateSchemaPropertyPaths(*context, schema, node, node)

	// build the complete path with the violation property
	if violationProperty != "" {
		if segment >= 0 {
			locatedPath = model.GetStringTemplates().BuildPropertyArrayPath(locatedPath, violationProperty, segment)
		} else {
			locatedPath = model.GetStringTemplates().BuildJSONPath(locatedPath, violationProperty)
		}

		// update all paths with the violation property
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

	typeMismatchResults := st.checkTypeMismatchedConstraints(schema, "object", context)
	results = append(results, typeMismatchResults...)

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

			// check if the required field is defined in properties (direct or polymorphic)
			propertyExists := false
			if schema.Value.Properties != nil && schema.Value.Properties.GetOrZero(required) != nil {
				propertyExists = true
			}

			// if not in direct properties, check if it was found in polymorphic schemas
			if !propertyExists && polyDefined {
				propertyExists = true
			}

			// report error if property is not defined anywhere
			if !propertyExists {
				result := st.buildResult(model.GetStringTemplates().BuildRequiredFieldMessage(required),
					schema.GenerateJSONPath(), "required", i,
					schema, schema.Value.GoLow().Required.KeyNode, context)
				results = append(results, result)
			}
		}
	}

	// validate DependentRequired
	dependentRequiredResults := st.validateDependentRequired(schema, context)
	results = append(results, dependentRequiredResults...)

	return results
}

// constraintInfo holds information about a constraint that doesn't match the schema type
type constraintInfo struct {
	name     string
	node     *yaml.Node
	validFor string
}

// constraintChecker defines a function that checks if a constraint exists on a schema
type constraintChecker func(*highBase.Schema) *yaml.Node

// schemaConstraint defines a constraint with its name, checker function, and valid types
type schemaConstraint struct {
	name     string
	checker  constraintChecker
	validFor string
}

// buildConstraintCheckers returns all defined schema constraints organized by category
func buildConstraintCheckers(lowSchema *lowBase.Schema) map[string][]schemaConstraint {
	return map[string][]schemaConstraint{
		"string": {
			{"pattern", func(s *highBase.Schema) *yaml.Node {
				if s.Pattern != "" {
					return lowSchema.Pattern.KeyNode
				}
				return nil
			}, "string"},
			{"minLength", func(s *highBase.Schema) *yaml.Node {
				if s.MinLength != nil {
					return lowSchema.MinLength.KeyNode
				}
				return nil
			}, "string"},
			{"maxLength", func(s *highBase.Schema) *yaml.Node {
				if s.MaxLength != nil {
					return lowSchema.MaxLength.KeyNode
				}
				return nil
			}, "string"},
		},
		"number": {
			{"minimum", func(s *highBase.Schema) *yaml.Node {
				if s.Minimum != nil {
					return lowSchema.Minimum.KeyNode
				}
				return nil
			}, "number/integer"},
			{"maximum", func(s *highBase.Schema) *yaml.Node {
				if s.Maximum != nil {
					return lowSchema.Maximum.KeyNode
				}
				return nil
			}, "number/integer"},
			{"multipleOf", func(s *highBase.Schema) *yaml.Node {
				if s.MultipleOf != nil {
					return lowSchema.MultipleOf.KeyNode
				}
				return nil
			}, "number/integer"},
			{"exclusiveMinimum", func(s *highBase.Schema) *yaml.Node {
				if s.ExclusiveMinimum != nil {
					return lowSchema.ExclusiveMinimum.KeyNode
				}
				return nil
			}, "number/integer"},
			{"exclusiveMaximum", func(s *highBase.Schema) *yaml.Node {
				if s.ExclusiveMaximum != nil {
					return lowSchema.ExclusiveMaximum.KeyNode
				}
				return nil
			}, "number/integer"},
		},
		"array": {
			{"minItems", func(s *highBase.Schema) *yaml.Node {
				if s.MinItems != nil {
					return lowSchema.MinItems.KeyNode
				}
				return nil
			}, "array"},
			{"maxItems", func(s *highBase.Schema) *yaml.Node {
				if s.MaxItems != nil {
					return lowSchema.MaxItems.KeyNode
				}
				return nil
			}, "array"},
			{"uniqueItems", func(s *highBase.Schema) *yaml.Node {
				if s.UniqueItems != nil {
					return lowSchema.UniqueItems.KeyNode
				}
				return nil
			}, "array"},
			{"minContains", func(s *highBase.Schema) *yaml.Node {
				if s.MinContains != nil {
					return lowSchema.MinContains.KeyNode
				}
				return nil
			}, "array"},
			{"maxContains", func(s *highBase.Schema) *yaml.Node {
				if s.MaxContains != nil {
					return lowSchema.MaxContains.KeyNode
				}
				return nil
			}, "array"},
		},
		"object": {
			{"minProperties", func(s *highBase.Schema) *yaml.Node {
				if s.MinProperties != nil {
					return lowSchema.MinProperties.KeyNode
				}
				return nil
			}, "object"},
			{"maxProperties", func(s *highBase.Schema) *yaml.Node {
				if s.MaxProperties != nil {
					return lowSchema.MaxProperties.KeyNode
				}
				return nil
			}, "object"},
		},
	}
}

// checkTypeMismatchedConstraints validates that a schema only uses constraints appropriate for its type.
// This ensures JSON Schema compliance by preventing semantically incorrect constraint usage.
func (st SchemaTypeCheck) checkTypeMismatchedConstraints(schema *v3.Schema, schemaType string, context *model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult
	mismatches := make([]constraintInfo, 0, 15)

	lowSchema := schema.Value.GoLow()
	allConstraints := buildConstraintCheckers(lowSchema)

	// determine which constraint categories are invalid for this type
	var invalidConstraintTypes []string
	switch schemaType {
	case "string":
		invalidConstraintTypes = []string{"number", "array", "object"}
	case "number", "integer":
		invalidConstraintTypes = []string{"string", "array", "object"}
	case "array":
		invalidConstraintTypes = []string{"string", "number", "object"}
	case "object":
		invalidConstraintTypes = []string{"string", "number", "array"}
	case "boolean", "null":
		invalidConstraintTypes = []string{"string", "number", "array", "object"}
	}

	// check for any invalid constraints on this schema
	for _, constraintType := range invalidConstraintTypes {
		for _, constraint := range allConstraints[constraintType] {
			if node := constraint.checker(schema.Value); node != nil {
				mismatches = append(mismatches, constraintInfo{
					name:     constraint.name,
					node:     node,
					validFor: constraint.validFor,
				})
			}
		}
	}

	// build results for all mismatched constraints
	for _, mismatch := range mismatches {
		message := fmt.Sprintf("`%s` constraint is only applicable to %s types, not `%s`",
			mismatch.name, mismatch.validFor, schemaType)
		result := st.buildResult(message, schema.GenerateJSONPath(), mismatch.name, -1,
			schema, mismatch.node, context)
		results = append(results, result)
	}

	return results
}

func (st SchemaTypeCheck) validateBoolean(schema *v3.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
	return st.checkTypeMismatchedConstraints(schema, "boolean", context)
}

func (st SchemaTypeCheck) validateNull(schema *v3.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
	return st.checkTypeMismatchedConstraints(schema, "null", context)
}

func (st SchemaTypeCheck) validateDependentRequired(schema *v3.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult

	// check if DependentRequired is present
	if schema.Value.DependentRequired == nil {
		return results
	}

	// iterate through all dependent required entries
	for pair := schema.Value.DependentRequired.First(); pair != nil; pair = pair.Next() {
		triggerProp := pair.Key()
		requiredProps := pair.Value()

		// validate that trigger property exists in schema properties if properties are defined
		if schema.Value.Properties != nil && schema.Value.Properties.GetOrZero(triggerProp) == nil {
			// check if the property exists in polymorphic schemas (anyOf, oneOf, allOf)
			polyDefined := st.checkPolymorphicProperty(schema, triggerProp)
			if !polyDefined {
				result := st.buildResult(
					fmt.Sprintf("property `%s` referenced in `dependentRequired` does not exist in schema `properties`", triggerProp),
					schema.GenerateJSONPath(), "dependentRequired", -1,
					schema, schema.Value.GoLow().DependentRequired.KeyNode, context)
				results = append(results, result)
			}
		}

		// validate that all dependent properties exist in schema
		for _, reqProp := range requiredProps {
			if schema.Value.Properties != nil && schema.Value.Properties.GetOrZero(reqProp) == nil {
				// check if the property exists in polymorphic schemas
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

		// validate no self-referential dependencies
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
	// check in AnyOf schemas
	if schema.Value.AnyOf != nil {
		for _, anyOfSchema := range schema.Value.AnyOf {
			if anyOfSchema.Schema() != nil && anyOfSchema.Schema().Properties != nil &&
				anyOfSchema.Schema().Properties.GetOrZero(propertyName) != nil {
				return true
			}
		}
	}

	// check in OneOf schemas
	if schema.Value.OneOf != nil {
		for _, oneOfSchema := range schema.Value.OneOf {
			if oneOfSchema.Schema() != nil && oneOfSchema.Schema().Properties != nil &&
				oneOfSchema.Schema().Properties.GetOrZero(propertyName) != nil {
				return true
			}
		}
	}

	// check in AllOf schemas
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

func (st SchemaTypeCheck) validateConst(schema *v3.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult

	// check if const is present
	if schema.Value.Const == nil {
		return results
	}

	constValueNode := schema.Value.GoLow().Const.ValueNode
	schemaTypes := schema.Value.Type

	// if no types declared, cannot validate const
	if len(schemaTypes) == 0 {
		return results
	}

	// check if const value matches any of the declared types
	isValid := false
	for _, schemaType := range schemaTypes {
		if st.isConstNodeValidForType(constValueNode, schemaType) {
			isValid = true
			break
		}
	}

	if !isValid {
		typeList := fmt.Sprintf("[%s]", strings.Join(schemaTypes, ", "))
		message := fmt.Sprintf("`const` value type does not match schema type %s", typeList)

		result := st.buildResult(message,
			schema.GenerateJSONPath(), "const", -1,
			schema, schema.Value.GoLow().Const.KeyNode, context)
		results = append(results, result)
	}

	return results
}

func (st SchemaTypeCheck) validateEnumConst(schema *v3.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult

	// check if both enum and const are present
	if schema.Value.Enum == nil || schema.Value.Const == nil {
		return results
	}

	constValue := schema.Value.Const.Value
	enumValues := schema.Value.Enum

	// check if const value exists in enum values
	constInEnum := false
	for _, enumValue := range enumValues {
		if enumValue.Value == constValue {
			constInEnum = true
			break
		}
	}

	if !constInEnum {
		message := fmt.Sprintf("`const` value `%s` is not present in `enum` values", constValue)

		result := st.buildResult(message,
			schema.GenerateJSONPath(), "const", -1,
			schema, schema.Value.GoLow().Const.KeyNode, context)
		results = append(results, result)

	} else {

		// both enum and const are present and compatible - flag as potentially redundant
		if len(enumValues) == 1 {
			message := "schema uses both `enum` with single value and `const` - consider using only `const`"
			result := st.buildResult(message,
				schema.GenerateJSONPath(), "enum", -1,
				schema, schema.Value.GoLow().Enum.KeyNode, context)
			results = append(results, result)
		} else {
			message := "schema uses both `enum` and `const` - this is likely an oversight as `const` restricts to a single value"
			result := st.buildResult(message,
				schema.GenerateJSONPath(), "enum", -1,
				schema, schema.Value.GoLow().Enum.KeyNode, context)
			results = append(results, result)
		}
	}

	return results
}

func (st SchemaTypeCheck) isConstNodeValidForType(node *yaml.Node, schemaType string) bool {
	switch schemaType {
	case "string":
		return node.Tag == "!!str"
	case "integer":
		if node.Tag == "!!int" {
			return true
		}
		// allow float values that have no fractional part (e.g., 42.0)
		if node.Tag == "!!float" {
			return st.isFloatWhole(node.Value)
		}
		return false
	case "number":
		return node.Tag == "!!int" || node.Tag == "!!float"
	case "boolean":
		return node.Tag == "!!bool"
	case "null":
		return node.Tag == "!!null"
	case "array":
		return node.Tag == "!!seq"
	case "object":
		return node.Tag == "!!map"
	}
	return false
}

func (st SchemaTypeCheck) isFloatWhole(value string) bool {
	// check if a float string represents a whole number (e.g., "42.0" -> true, "42.5" -> false)
	if !strings.Contains(value, ".") {
		return true
	}
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return false
	}
	// check if fractional part is all zeros
	fractional := parts[1]
	for _, char := range fractional {
		if char != '0' {
			return false
		}
	}
	return true
}

func (st SchemaTypeCheck) isConstValueValidForType(value interface{}, schemaType string) bool {
	switch schemaType {
	case "string":
		_, ok := value.(string)
		return ok
	case "integer":
		// integers can be int, int64, or float64 with no fractional part
		switch v := value.(type) {
		case int, int64:
			return true
		case float64:
			return v == float64(int64(v))
		}
		return false
	case "number":
		// numbers can be int, int64, or float64
		switch value.(type) {
		case int, int64, float64:
			return true
		}
		return false
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "null":
		return value == nil
	case "array":
		// arrays are represented as []interface{}
		_, ok := value.([]interface{})
		return ok
	case "object":
		// objects are represented as map[string]interface{}
		_, ok := value.(map[string]interface{})
		return ok
	}
	return false
}

// validateDiscriminator checks that discriminator.propertyName exists in schema properties or polymorphic compositions.
// note: discriminator properties are not required to be in the required array per OpenAPI spec.
func (st SchemaTypeCheck) validateDiscriminator(schema *v3.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult

	if schema.Discriminator == nil {
		return results
	}

	discriminator := schema.Discriminator
	propertyName := discriminator.Value.PropertyName

	// propertyName is required per OpenAPI 3.x spec
	if propertyName == "" {
		result := st.buildResult(
			"discriminator object is missing required `propertyName` field",
			schema.GenerateJSONPath(), "discriminator", -1,
			schema, discriminator.KeyNode, context)
		results = append(results, result)
		return results
	}

	propertyExists := false
	if schema.Value.Properties != nil && schema.Value.Properties.GetOrZero(propertyName) != nil {
		propertyExists = true
	}

	if !propertyExists {
		propertyExists = st.checkPolymorphicProperty(schema, propertyName)
	}

	if !propertyExists {
		result := st.buildResult(
			fmt.Sprintf("discriminator property `%s` is not defined in schema properties", propertyName),
			schema.GenerateJSONPath(), "discriminator", -1,
			schema, discriminator.KeyNode, context)
		results = append(results, result)
	}

	return results
}
