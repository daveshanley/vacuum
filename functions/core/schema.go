// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package core

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/parser"
	"github.com/mitchellh/mapstructure"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

// Schema is a rule that creates a schema check against a field value
type Schema struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the OperationParameters rule.
func (sch Schema) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "oas_schema",
	}
}

// RunRule will execute the Schema function
func (sch Schema) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	unpack := utils.ExtractValueFromInterfaceMap("unpack", context.Options)
	if _, ok := unpack.(bool); ok {
		nodes = nodes[0].Content
	}

	var results []model.RuleFunctionResult

	var schema parser.Schema
	var ok bool
	s := utils.ExtractValueFromInterfaceMap("schema", context.Options)
	if schema, ok = s.(parser.Schema); !ok {
		var p parser.Schema
		_ = mapstructure.Decode(s, &p)
		schema = p
	}

	for x, node := range nodes {
		if x%2 == 0 && len(nodes) > 1 {
			continue
		}
		// find field from rule
		_, field := utils.FindKeyNode(context.RuleAction.Field, node.Content)
		if field != nil {

			results = append(results, validateNodeAgainstSchema(schema, field, context, x)...)

		} else {
			// If the field is not found, and we're being strict, it's invalid.
			forceValidation := utils.ExtractValueFromInterfaceMap("forceValidation", context.Options)
			if _, ok := forceValidation.(bool); ok {

				r := model.BuildFunctionResultString(fmt.Sprintf("%s: %s", context.Rule.Description,
					fmt.Sprintf("`%s`, is missing and is required", context.RuleAction.Field)))
				r.StartNode = node
				r.EndNode = node.Content[len(node.Content)-1]
				r.Rule = context.Rule
				if p, ok := context.Given.(string); ok {
					r.Path = fmt.Sprintf("%s[%d]", p, x)
				}
				results = append(results, r)
			}
		}
	}
	return results
}

func validateNodeAgainstSchema(schema parser.Schema, field *yaml.Node,
	context model.RuleFunctionContext, x int) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	// validate using schema provided.
	res, _ := parser.ValidateNodeAgainstSchema(&schema, field, false)

	if res == nil {
		return results
	}

	for _, resError := range res.Errors() {

		r := model.BuildFunctionResultString(fmt.Sprintf("%s: %s", context.Rule.Description,
			resError.Description()))
		r.StartNode = field
		r.EndNode = field
		r.Rule = context.Rule
		if p, ok := context.Given.(string); ok {
			r.Path = fmt.Sprintf("%s[%d]", p, x)
		}
		results = append(results, r)
	}
	return results
}
