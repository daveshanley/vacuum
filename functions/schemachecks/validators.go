// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package schemachecks

import (
	"fmt"
	"strings"
	"sync"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/dop251/goja"
	drV3 "github.com/pb33f/doctor/model/high/v3"
	"go.yaml.in/yaml/v4"
)

var ecma262PatternCache sync.Map

// RunTypeChecks runs the full schema type check set for OpenAPI and compatible schema hosts.
func RunTypeChecks(schemas []*drV3.Schema, context model.RuleFunctionContext, options TypeCheckOptions) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult
	for _, schema := range schemas {
		results = append(results, runSchemaTypeChecks(schema, &context, options)...)
	}
	return results
}

// RunSchemaSanityCheck runs one JSON Schema sanity check against a Doctor schema.
func RunSchemaSanityCheck(schema *drV3.Schema, root *yaml.Node, context *model.RuleFunctionContext,
	check string,
) []model.RuleFunctionResult {
	if schema == nil || schema.Value == nil || schema.Value.GoLow() == nil {
		return nil
	}

	switch check {
	case SanityCheckType:
		return runStructuralTypeChecks(schema, context, TypeCheckOptions{})
	case SanityCheckRequired:
		return CheckRequiredFields(schema, context)
	case SanityCheckDependent:
		return validateDependentRequired(schema, context)
	case SanityCheckEnumConst:
		var results []model.RuleFunctionResult
		results = append(results, validateEnumDuplicates(schema, context)...)
		results = append(results, validateConst(schema, context)...)
		results = append(results, validateEnumTypes(schema, context, false)...)
		results = append(results, validateEnumConst(schema, context, false)...)
		return results
	case SanityCheckPatterns:
		return validatePatternKeywords(schema, context)
	case SanityCheckComposition:
		return checkShallowComposition(schema, context)
	case SanityCheckQuality:
		return checkQualityRules(root, schema, context)
	case SanityCheckExamples:
		return checkExamplesAndDefault(schema, context)
	default:
		return nil
	}
}

func runSchemaTypeChecks(schema *drV3.Schema, context *model.RuleFunctionContext,
	options TypeCheckOptions,
) []model.RuleFunctionResult {
	results := runStructuralTypeChecks(schema, context, options)
	if schema == nil || schema.Value == nil {
		return results
	}

	if options.ValidateValueCompatibility && len(schema.Value.Type) > 0 {
		results = append(results, validateConst(schema, context)...)
		results = append(results, validateEnumTypes(schema, context, options.AllowOAS30Nullable)...)
	}

	if options.ValidateValueCompatibility {
		results = append(results, validateEnumConst(schema, context, options.ValidateEnumConstRedundancy)...)
	}

	if options.ValidateDiscriminator {
		results = append(results, validateDiscriminator(schema, context)...)
	}

	return results
}

func runStructuralTypeChecks(schema *drV3.Schema, context *model.RuleFunctionContext,
	options TypeCheckOptions,
) []model.RuleFunctionResult {
	if schema == nil || schema.Value == nil || schema.Value.GoLow() == nil {
		return nil
	}

	var results []model.RuleFunctionResult
	schemaTypes := schema.Value.Type
	results = append(results, CheckTypeMismatchedConstraints(schema, context)...)

	var validatedString bool
	var validatedNumber bool
	var validatedBoolean bool
	var validatedArray bool
	var validatedObject bool
	var validatedNull bool
	for _, schemaType := range schemaTypes {
		switch schemaType {
		case "string":
			if !validatedString {
				results = append(results, validateString(schema, context, options.ValidatePatterns)...)
				validatedString = true
			}
		case "integer", "number":
			if !validatedNumber {
				results = append(results, validateNumber(schema, context)...)
				validatedNumber = true
			}
		case "boolean":
			if !validatedBoolean {
				validatedBoolean = true
			}
		case "array":
			if !validatedArray {
				results = append(results, validateArray(schema, context)...)
				validatedArray = true
			}
		case "object":
			if !validatedObject {
				results = append(results, validateObject(schema, context, options.ValidateDependentRequired)...)
				validatedObject = true
			}
		case "null":
			if !validatedNull {
				validatedNull = true
			}
		default:
			result := BuildResult(model.GetStringTemplates().BuildUnknownSchemaTypeMessage(schemaType),
				schema.GenerateJSONPath(), "type", -1, schema, schema.Value.GoLow().Type.KeyNode, context)
			results = append(results, result)
		}
	}

	return results
}

func validateNumber(schema *drV3.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult

	if schema.Value.MultipleOf != nil && *schema.Value.MultipleOf <= 0 {
		result := BuildResult("`multipleOf` should be a number greater than `0`",
			schema.GenerateJSONPath(), "multipleOf", -1, schema, schema.Value.GoLow().MultipleOf.KeyNode, context)
		results = append(results, result)
	}

	if schema.Value.Maximum != nil && schema.Value.Minimum != nil && *schema.Value.Maximum < *schema.Value.Minimum {
		result := BuildResult("`maximum` should be a number greater than or equal to `minimum`",
			schema.GenerateJSONPath(), "maximum", -1, schema, schema.Value.GoLow().Maximum.KeyNode, context)
		results = append(results, result)
	}

	if schema.Value.ExclusiveMaximum != nil && schema.Value.ExclusiveMinimum != nil &&
		schema.Value.ExclusiveMinimum.IsB() && schema.Value.ExclusiveMinimum.B > schema.Value.ExclusiveMaximum.B {
		result := BuildResult("`exclusiveMaximum` should be greater than or equal to `exclusiveMinimum`",
			schema.GenerateJSONPath(), "exclusiveMaximum", -1, schema, schema.Value.GoLow().ExclusiveMaximum.KeyNode, context)
		results = append(results, result)
	}

	return results
}

func validateString(schema *drV3.Schema, context *model.RuleFunctionContext, validatePattern bool) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult

	if schema.Value.MinLength != nil && *schema.Value.MinLength < 0 {
		result := BuildResult("`minLength` should be a non-negative number",
			schema.GenerateJSONPath(), "minLength", -1, schema, schema.Value.GoLow().MinLength.KeyNode, context)
		results = append(results, result)
	}

	if schema.Value.MaxLength != nil {
		if *schema.Value.MaxLength < 0 {
			result := BuildResult("`maxLength` should be a non-negative number",
				schema.GenerateJSONPath(), "maxLength", -1, schema, schema.Value.GoLow().MaxLength.KeyNode, context)
			results = append(results, result)
		}
		if schema.Value.MinLength != nil && *schema.Value.MinLength > *schema.Value.MaxLength {
			result := BuildResult("`maxLength` should be greater than or equal to `minLength`",
				schema.GenerateJSONPath(), "maxLength", -1, schema, schema.Value.GoLow().MaxLength.KeyNode, context)
			results = append(results, result)
		}
	}

	if validatePattern && schema.Value.Pattern != "" && !ecma262PatternValid(schema.Value.Pattern) {
		result := BuildResult("schema `pattern` should be a ECMA-262 regular expression dialect",
			schema.GenerateJSONPath(), "pattern", -1, schema, schema.Value.GoLow().Pattern.KeyNode, context)
		results = append(results, result)
	}

	return results
}

func validateArray(schema *drV3.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult

	if schema.Value.MinItems != nil && *schema.Value.MinItems < 0 {
		result := BuildResult("`minItems` should be a non-negative number",
			schema.GenerateJSONPath(), "minItems", -1, schema, schema.Value.GoLow().MinItems.KeyNode, context)
		results = append(results, result)
	}

	if schema.Value.MaxItems != nil {
		if *schema.Value.MaxItems < 0 {
			result := BuildResult("`maxItems` should be a non-negative number",
				schema.GenerateJSONPath(), "maxItems", -1, schema, schema.Value.GoLow().MaxItems.KeyNode, context)
			results = append(results, result)
		}
		if schema.Value.MinItems != nil && *schema.Value.MinItems > *schema.Value.MaxItems {
			result := BuildResult("`maxItems` should be greater than or equal to `minItems`",
				schema.GenerateJSONPath(), "maxItems", -1, schema, schema.Value.GoLow().MaxItems.KeyNode, context)
			results = append(results, result)
		}
	}

	if schema.Value.MinContains != nil && *schema.Value.MinContains < 0 {
		result := BuildResult("`minContains` should be a non-negative number",
			schema.GenerateJSONPath(), "minContains", -1, schema, schema.Value.GoLow().MinContains.KeyNode, context)
		results = append(results, result)
	}

	if schema.Value.MaxContains != nil {
		if *schema.Value.MaxContains < 0 {
			result := BuildResult("`maxContains` should be a non-negative number",
				schema.GenerateJSONPath(), "maxContains", -1, schema, schema.Value.GoLow().MaxContains.KeyNode, context)
			results = append(results, result)
		}
		if schema.Value.MinContains != nil && *schema.Value.MinContains > *schema.Value.MaxContains {
			result := BuildResult("`maxContains` should be greater than or equal to `minContains`",
				schema.GenerateJSONPath(), "maxContains", -1, schema, schema.Value.GoLow().MaxContains.KeyNode, context)
			results = append(results, result)
		}
	}

	return results
}

func validateObject(schema *drV3.Schema, context *model.RuleFunctionContext,
	validateDependencies bool,
) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult

	if schema.Value.MinProperties != nil && *schema.Value.MinProperties < 0 {
		result := BuildResult("`minProperties` should be a non-negative number",
			schema.GenerateJSONPath(), "minProperties", -1, schema, schema.Value.GoLow().MinProperties.KeyNode, context)
		results = append(results, result)
	}

	if schema.Value.MaxProperties != nil {
		if *schema.Value.MaxProperties < 0 {
			result := BuildResult("`maxProperties` should be a non-negative number",
				schema.GenerateJSONPath(), "maxProperties", -1, schema, schema.Value.GoLow().MaxProperties.KeyNode, context)
			results = append(results, result)
		}
		if schema.Value.MinProperties != nil && *schema.Value.MinProperties > *schema.Value.MaxProperties {
			result := BuildResult("`maxProperties` should be greater than or equal to `minProperties`",
				schema.GenerateJSONPath(), "maxProperties", -1, schema, schema.Value.GoLow().MaxProperties.KeyNode, context)
			results = append(results, result)
		}
	}

	if validateDependencies {
		results = append(results, validateDependentRequired(schema, context)...)
	}

	return results
}

func validateDependentRequired(schema *drV3.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult
	if schema.Value.DependentRequired == nil {
		return results
	}

	for pair := schema.Value.DependentRequired.First(); pair != nil; pair = pair.Next() {
		triggerProp := pair.Key()
		requiredProps := pair.Value()

		if schema.Value.Properties != nil && schema.Value.Properties.GetOrZero(triggerProp) == nil &&
			!checkPolymorphicProperty(schema, triggerProp) {
			result := BuildResult(
				fmt.Sprintf("property `%s` referenced in `dependentRequired` does not exist in schema `properties`", triggerProp),
				schema.GenerateJSONPath(), "dependentRequired", -1,
				schema, schema.Value.GoLow().DependentRequired.KeyNode, context)
			results = append(results, result)
		}

		for _, reqProp := range requiredProps {
			if schema.Value.Properties != nil && schema.Value.Properties.GetOrZero(reqProp) == nil &&
				!checkPolymorphicProperty(schema, reqProp) {
				result := BuildResult(
					fmt.Sprintf("property `%s` referenced in `dependentRequired` does not exist in schema `properties`", reqProp),
					schema.GenerateJSONPath(), "dependentRequired", -1,
					schema, schema.Value.GoLow().DependentRequired.KeyNode, context)
				results = append(results, result)
			}
		}

		for _, reqProp := range requiredProps {
			if reqProp == triggerProp {
				result := BuildResult(
					fmt.Sprintf("circular dependency detected: property `%s` requires itself in `dependentRequired`", triggerProp),
					schema.GenerateJSONPath(), "dependentRequired", -1,
					schema, schema.Value.GoLow().DependentRequired.KeyNode, context)
				results = append(results, result)
			}
		}
	}

	return results
}

func validateConst(schema *drV3.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
	if schema.Value.Const == nil || len(schema.Value.Type) == 0 {
		return nil
	}

	constValueNode := schema.Value.GoLow().Const.ValueNode
	for _, schemaType := range schema.Value.Type {
		if constNodeValidForType(constValueNode, schemaType) {
			return nil
		}
	}

	typeList := fmt.Sprintf("[%s]", strings.Join(schema.Value.Type, ", "))
	message := fmt.Sprintf("`const` value type does not match schema type %s", typeList)
	result := BuildResult(message, schema.GenerateJSONPath(), "const", -1,
		schema, schema.Value.GoLow().Const.KeyNode, context)
	return []model.RuleFunctionResult{result}
}

func validateEnumTypes(schema *drV3.Schema, context *model.RuleFunctionContext,
	allowOAS30Nullable bool,
) []model.RuleFunctionResult {
	if len(schema.Value.Enum) == 0 {
		return nil
	}

	var results []model.RuleFunctionResult
	nullAllowed := isNullAllowedForSchema(schema, context, allowOAS30Nullable)

	for i, node := range schema.Value.Enum {
		if node == nil {
			continue
		}
		if node.Tag == "!!null" && nullAllowed {
			continue
		}

		matched := false
		for _, schemaType := range schema.Value.Type {
			if constNodeValidForType(node, schemaType) {
				matched = true
				break
			}
		}
		if matched {
			continue
		}

		typeList := formatSchemaTypesForMessage(schema.Value.Type)
		message := fmt.Sprintf("`enum` value `%s` does not match schema type `%s`", node.Value, typeList)
		result := BuildResult(message, schema.GenerateJSONPath(), "enum", i,
			schema, schema.Value.GoLow().Enum.KeyNode, context)
		results = append(results, result)
	}

	return results
}

func validateEnumConst(schema *drV3.Schema, context *model.RuleFunctionContext,
	includeRedundancy bool,
) []model.RuleFunctionResult {
	if schema.Value.Enum == nil || schema.Value.Const == nil {
		return nil
	}

	enumNode := schema.Value.GoLow().Enum.ValueNode
	constNode := schema.Value.GoLow().Const.ValueNode
	if enumNode != nil && enumNode.Kind == yaml.SequenceNode && !sequenceContainsEquivalent(enumNode, constNode) {
		message := fmt.Sprintf("`const` value `%s` is not present in `enum` values", schema.Value.Const.Value)
		result := BuildResult(message, schema.GenerateJSONPath(), "const", -1,
			schema, schema.Value.GoLow().Const.KeyNode, context)
		return []model.RuleFunctionResult{result}
	}

	if !includeRedundancy {
		return nil
	}

	if len(schema.Value.Enum) == 1 {
		message := "schema uses both `enum` with single value and `const` - consider using only `const`"
		result := BuildResult(message, schema.GenerateJSONPath(), "enum", -1,
			schema, schema.Value.GoLow().Enum.KeyNode, context)
		return []model.RuleFunctionResult{result}
	}

	message := "schema uses both `enum` and `const` - this is likely an oversight as `const` restricts to a single value"
	result := BuildResult(message, schema.GenerateJSONPath(), "enum", -1,
		schema, schema.Value.GoLow().Enum.KeyNode, context)
	return []model.RuleFunctionResult{result}
}

func validateEnumDuplicates(schema *drV3.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
	enumNode := schema.Value.GoLow().Enum.ValueNode
	if enumNode == nil || enumNode.Kind != yaml.SequenceNode {
		return nil
	}

	var results []model.RuleFunctionResult
	seen := make(map[string]int, len(enumNode.Content))
	for i, enumValue := range enumNode.Content {
		key := stableNodeValue(enumValue)
		if first, ok := seen[key]; ok {
			result := BuildResult(fmt.Sprintf("enum value duplicates enum[%d]", first),
				schema.GenerateJSONPath(), "enum", i, schema, enumValue, context)
			results = append(results, result)
			continue
		}
		seen[key] = i
	}
	return results
}

func isNullAllowedForSchema(schema *drV3.Schema, context *model.RuleFunctionContext, allowOAS30Nullable bool) bool {
	for _, schemaType := range schema.Value.Type {
		if schemaType == "null" {
			return true
		}
	}

	if !allowOAS30Nullable || schema.Value.Nullable == nil || !*schema.Value.Nullable {
		return false
	}

	specInfo := context.SpecInfo
	if specInfo == nil && context.Document != nil {
		specInfo = context.Document.GetSpecInfo()
	}

	return vacuumUtils.IsOAS30(specInfo)
}

func ecma262PatternValid(pattern string) bool {
	if cached, ok := ecma262PatternCache.Load(pattern); ok {
		return cached.(bool)
	}
	vm := goja.New()
	script := fmt.Sprintf("const regex = new RegExp(%q)", pattern)
	_, err := vm.RunString(script)
	valid := err == nil
	ecma262PatternCache.Store(pattern, valid)
	return valid
}
