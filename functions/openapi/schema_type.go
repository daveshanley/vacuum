// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley // https://pb33f.io

package openapi

import (
	"fmt"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/dop251/goja"
	"github.com/pb33f/doctor/model/high/base"
	"gopkg.in/yaml.v3"
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

func (st SchemaTypeCheck) validateString(schema *base.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
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

func (st SchemaTypeCheck) validateArray(schema *base.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
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

func (st SchemaTypeCheck) buildResult(message, path, violationProperty string, segment int, schema *base.Schema, node *yaml.Node, context *model.RuleFunctionContext) model.RuleFunctionResult {

	// locate all paths that this model is referenced by
	var allPaths []string
	var modelByLine []base.Foundational
	var modelErr error
	if context.DrDocument != nil {
		modelByLine, modelErr = context.DrDocument.LocateModelByLine(node.Line + 1)
		if modelErr == nil {
			if modelByLine != nil && len(modelByLine) >= 1 {
				for j := 0; j < len(modelByLine); j++ {
					p := modelByLine[j].GenerateJSONPath()
					allPaths = append(allPaths, p)
					if violationProperty != "" {
						if segment >= 0 {
							p = fmt.Sprintf("%s.%s[%d]", p, violationProperty, segment)
							path = p
						} else {
							p = fmt.Sprintf("%s.%s", p, violationProperty)
							path = p
						}
						allPaths = append(allPaths, p)
					}
				}
			}
		} else {
			if violationProperty != "" {
				if segment >= 0 {
					path = fmt.Sprintf("%s.%s[%d]", path, violationProperty, segment)
				} else {
					path = fmt.Sprintf("%s.%s", path, violationProperty)
				}
			}
		}
	}
	result := model.RuleFunctionResult{
		Message:   message,
		StartNode: node,
		EndNode:   vacuumUtils.BuildEndNode(node),
		Path:      path,
		Rule:      context.Rule,
	}
	if len(allPaths) > 1 {
		result.Paths = allPaths
	}
	schema.AddRuleFunctionResult(base.ConvertRuleResult(&result))
	return result
}

func (st SchemaTypeCheck) validateObject(schema *base.Schema, context *model.RuleFunctionContext) []model.RuleFunctionResult {
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

			if schema.Value.Properties != nil && schema.Value.Properties.GetOrZero(required) == nil && !polyDefined {
				result := st.buildResult(fmt.Sprintf("`required` field `%s` is not defined in `properties`", required),
					schema.GenerateJSONPath(), "required", i,
					schema, schema.Value.GoLow().Required.KeyNode, context)
				results = append(results, result)
			}
		}
	}

	// TODO: DependentRequired
	return results
}
