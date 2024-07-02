// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley // https://pb33f.io

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/dop251/goja"
	"github.com/pb33f/doctor/model/high/base"
	"gopkg.in/yaml.v3"
	"strings"
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
				result := model.RuleFunctionResult{
					Message:   fmt.Sprintf("unknown schema type: `%s`", t),
					StartNode: schema.Value.GoLow().Type.KeyNode,
					EndNode:   vacuumUtils.BuildEndNode(schema.Value.GoLow().Type.KeyNode),
					Path:      fmt.Sprintf("%s.%s", schema.GenerateJSONPath(), "type"),
					Rule:      context.Rule,
				}
				schema.AddRuleFunctionResult(base.ConvertRuleResult(&result))
				results = append(results, result)
			}
		}
	}

	return results
}

func (st SchemaTypeCheck) validateNumber(schema *base.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult

	if schema.Value.MultipleOf != nil {
		if *schema.Value.MultipleOf <= 0 {
			result := st.buildResult("`multipleOf` should be a number greater than `0`",
				fmt.Sprintf("%s.%s", schema.GenerateJSONPath(), "multipleOf"),
				schema, schema.Value.GoLow().MultipleOf.KeyNode, context)
			results = append(results, result)
		}
	}

	if schema.Value.Maximum != nil {
		if schema.Value.Minimum != nil {
			if *schema.Value.Maximum < *schema.Value.Minimum {
				result := st.buildResult("`maximum` should be a number greater than or equal to `minimum`",
					fmt.Sprintf("%s.%s", schema.GenerateJSONPath(), "maximum"),
					schema, schema.Value.GoLow().Maximum.KeyNode, context)
				results = append(results, result)
			}
		}
	}

	if schema.Value.ExclusiveMaximum != nil {
		if schema.Value.ExclusiveMinimum != nil {
			if schema.Value.ExclusiveMinimum.IsB() && schema.Value.ExclusiveMinimum.B > schema.Value.ExclusiveMaximum.B {
				result := st.buildResult("`exclusiveMaximum` should be greater than or equal to `exclusiveMinimum`",
					fmt.Sprintf("%s.%s", schema.GenerateJSONPath(), "exclusiveMaximum"),
					schema, schema.Value.GoLow().ExclusiveMaximum.KeyNode, context)
				results = append(results, result)
			}
		}
	}

	return results
}

func (st SchemaTypeCheck) validateString(schema *base.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult

	if schema.Value.MinLength != nil {
		if *schema.Value.MinLength < 0 {
			result := st.buildResult("`minLength` should be a non-negative number",
				fmt.Sprintf("%s.%s", schema.GenerateJSONPath(), "minLength"),
				schema, schema.Value.GoLow().MinLength.KeyNode, context)
			results = append(results, result)
		}
	}

	if schema.Value.MaxLength != nil {
		if *schema.Value.MaxLength < 0 {
			result := st.buildResult("`maxLength` should be a non-negative number",
				fmt.Sprintf("%s.%s", schema.GenerateJSONPath(), "maxLength"),
				schema, schema.Value.GoLow().MaxLength.KeyNode, context)
			results = append(results, result)
		}
		if schema.Value.MinLength != nil {
			if *schema.Value.MinLength > *schema.Value.MaxLength {
				result := st.buildResult("`maxLength` should be greater than or equal to `minLength`",
					fmt.Sprintf("%s.%s", schema.GenerateJSONPath(), "maxLength"),
					schema, schema.Value.GoLow().MaxLength.KeyNode, context)
				results = append(results, result)
			}
		}
	}

	if schema.Value.Pattern != "" {
		vm := goja.New()
		script := strings.Replace("const regex = new RegExp('{pattern}');", "{pattern}", schema.Value.Pattern, 1)
		_, err := vm.RunString(script)
		if err != nil {
			result := st.buildResult("schema `pattern` should be a ECMA-262 regular expression dialect",
				fmt.Sprintf("%s.%s", schema.GenerateJSONPath(), "pattern"),
				schema, schema.Value.GoLow().Pattern.KeyNode, context)
			results = append(results, result)
		}
	}
	return results
}

func (st SchemaTypeCheck) validateArray(schema *base.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult

	if schema.Value.MinItems != nil {
		if *schema.Value.MinItems < 0 {
			result := st.buildResult("`minItems` should be a non-negative number",
				fmt.Sprintf("%s.%s", schema.GenerateJSONPath(), "minItems"),
				schema, schema.Value.GoLow().MinItems.KeyNode, context)
			results = append(results, result)
		}
	}

	if schema.Value.MaxItems != nil {
		if *schema.Value.MaxItems < 0 {
			result := st.buildResult("`maxItems` should be a non-negative number",
				fmt.Sprintf("%s.%s", schema.GenerateJSONPath(), "maxItems"),
				schema, schema.Value.GoLow().MaxItems.KeyNode, context)
			results = append(results, result)
		}
		if schema.Value.MinItems != nil {
			if *schema.Value.MinItems > *schema.Value.MaxItems {
				result := st.buildResult("`maxItems` should be greater than or equal to `minItems`",
					fmt.Sprintf("%s.%s", schema.GenerateJSONPath(), "maxItems"),
					schema, schema.Value.GoLow().MaxItems.KeyNode, context)
				results = append(results, result)
			}
		}
	}

	if schema.Value.MinContains != nil {
		if *schema.Value.MinContains < 0 {
			result := st.buildResult("`minContains` should be a non-negative number",
				fmt.Sprintf("%s.%s", schema.GenerateJSONPath(), "minContains"),
				schema, schema.Value.GoLow().MinContains.KeyNode, context)
			results = append(results, result)
		}
	}

	if schema.Value.MaxContains != nil {
		if *schema.Value.MaxContains < 0 {
			result := st.buildResult("`maxContains` should be a non-negative number",
				fmt.Sprintf("%s.%s", schema.GenerateJSONPath(), "maxContains"),
				schema, schema.Value.GoLow().MaxContains.KeyNode, context)
			results = append(results, result)
		}
		if schema.Value.MinContains != nil {
			if *schema.Value.MinContains > *schema.Value.MaxContains {
				result := st.buildResult("`maxContains` should be greater than or equal to `minContains`",
					fmt.Sprintf("%s.%s", schema.GenerateJSONPath(), "maxContains"),
					schema, schema.Value.GoLow().MaxContains.KeyNode, context)
				results = append(results, result)
			}
		}
	}

	return results
}

func (st SchemaTypeCheck) buildResult(message, path string, schema *base.Schema, node *yaml.Node, context *model.RuleFunctionContext) model.RuleFunctionResult {
	result := model.RuleFunctionResult{
		Message:   message,
		StartNode: node,
		EndNode:   vacuumUtils.BuildEndNode(node),
		Path:      path,
		Rule:      context.Rule,
	}
	schema.AddRuleFunctionResult(base.ConvertRuleResult(&result))
	return result
}

func (st SchemaTypeCheck) validateObject(schema *base.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult

	if schema.Value.MinProperties != nil {
		if *schema.Value.MinProperties < 0 {
			result := st.buildResult("`minProperties` should be a non-negative number",
				fmt.Sprintf("%s.%s", schema.GenerateJSONPath(), "minProperties"),
				schema, schema.Value.GoLow().MinProperties.KeyNode, context)
			results = append(results, result)
		}
	}

	if schema.Value.MaxProperties != nil {
		if *schema.Value.MaxProperties < 0 {
			result := st.buildResult("`maxProperties` should be a non-negative number",
				fmt.Sprintf("%s.%s", schema.GenerateJSONPath(), "maxProperties"),
				schema, schema.Value.GoLow().MaxProperties.KeyNode, context)
			results = append(results, result)
		}
		if schema.Value.MinProperties != nil {
			if *schema.Value.MinProperties > *schema.Value.MaxProperties {
				result := st.buildResult("`maxProperties` should be greater than or equal to `minProperties`",
					fmt.Sprintf("%s.%s", schema.GenerateJSONPath(), "maxProperties"),
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
						if anyOf.Schema().Properties.Len() >= 0 {
							polyFound = true
						}
						if anyOf.Schema().Properties.GetOrZero(required) != nil {
							polyDefined = true
						}
					}
				}
				if schema.Value.OneOf != nil {
					for _, oneOf := range schema.Value.OneOf {
						if oneOf.Schema().Properties.Len() >= 0 {
							polyFound = true
						}
						if oneOf.Schema().Properties.GetOrZero(required) != nil {
							polyDefined = true
						}
					}
				}
				if schema.Value.AllOf != nil {
					for _, allOf := range schema.Value.AllOf {
						if allOf.Schema().Properties.Len() >= 0 {
							polyFound = true
						}
						if allOf.Schema().Properties.GetOrZero(required) != nil {
							polyDefined = true
						}
					}
				}
			}
			if schema.Value.Properties == nil && !polyFound {
				result := st.buildResult("object contains `required` fields but no `properties`",
					fmt.Sprintf("%s.%s[%d]", schema.GenerateJSONPath(), "required", i),
					schema, schema.Value.GoLow().Required.KeyNode, context)
				results = append(results, result)
				break
			}

			if schema.Value.Properties != nil && schema.Value.Properties.GetOrZero(required) == nil && !polyDefined {
				result := st.buildResult(fmt.Sprintf("`required` field `%s` is not defined in `properties`", required),
					fmt.Sprintf("%s.%s[%d]", schema.GenerateJSONPath(), "required", i),
					schema, schema.Value.GoLow().Required.KeyNode, context)
				results = append(results, result)
			}
		}
	}

	// TODO: DependentRequired
	return results
}
