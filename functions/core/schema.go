// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package core

import (
	"fmt"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/parser"
	validationErrors "github.com/pb33f/libopenapi-validator/errors"
	highBase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/datamodel/low"
	lowBase "github.com/pb33f/libopenapi/datamodel/low/base"
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

	var schema *highBase.Schema
	var ok bool
	s := utils.ExtractValueFromInterfaceMap("schema", context.Options)
	if schema, ok = s.(*highBase.Schema); !ok {

		// build schema from scratch
		var lowSchema lowBase.Schema

		// unmarshal the schema
		var on yaml.Node
		err := on.Encode(&s)

		if err != nil {
			r := model.BuildFunctionResultString(fmt.Sprintf("unable to parse function options: %s", err.Error()))
			r.Rule = context.Rule
			results = append(results, r)
			return results
		}

		// first, run the model builder on the schema
		err = low.BuildModel(&on, &lowSchema)
		if err != nil {
			r := model.BuildFunctionResultString(fmt.Sprintf("unable to build low schema from function options: %s", err.Error()))
			r.Rule = context.Rule
			results = append(results, r)
			return results
		}

		// now build out the low level schema.
		err = lowSchema.Build(&on, context.Index)
		if err != nil {
			r := model.BuildFunctionResultString(fmt.Sprintf("unable to build high schema from function options: %s", err.Error()))
			r.Rule = context.Rule
			results = append(results, r)
			return results
		}

		// now, build the high level schema
		schema = highBase.NewSchema(&lowSchema)
	}

	// use the current node to validate (field not needed)
	forceValidationOnCurrentNode := utils.ExtractValueFromInterfaceMap("forceValidationOnCurrentNode", context.Options)
	if _, ok := forceValidationOnCurrentNode.(bool); ok && len(nodes) > 0 {
		results = append(results, validateNodeAgainstSchema(schema, nodes[0], context, 0)...)
		return results
	}

	for x, node := range nodes {
		if x%2 == 0 && len(nodes) > 1 {
			continue
		}
		// if the node is a document node, skip down one level
		var no []*yaml.Node
		if node.Kind == yaml.DocumentNode {
			no = node.Content[0].Content
		} else {
			no = node.Content
		}

		_, field := utils.FindKeyNodeTop(context.RuleAction.Field, no)
		if field != nil {
			results = append(results, validateNodeAgainstSchema(schema, field, context, x)...)

		} else {
			// If the field is not found, and we're being strict, it's invalid.

			forceValidation := utils.ExtractValueFromInterfaceMap("forceValidation", context.Options)
			if _, ko := forceValidation.(bool); ko {

				r := model.BuildFunctionResultString(fmt.Sprintf("%s: %s", context.Rule.Description,
					fmt.Sprintf("`%s`, is missing and is required", context.RuleAction.Field)))
				r.StartNode = node
				r.EndNode = node.Content[len(node.Content)-1]
				r.Rule = context.Rule
				if p, df := context.Given.(string); df {
					r.Path = fmt.Sprintf("%s[%d]", p, x)
				}
				results = append(results, r)
			}
		}
	}
	return results
}

func validateNodeAgainstSchema(schema *highBase.Schema, field *yaml.Node,
	context model.RuleFunctionContext, x int) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	// validate using schema provided.
	res, resErrors := parser.ValidateNodeAgainstSchema(schema, field, false)

	if res {
		return results
	}

	var schemaErrors []*validationErrors.SchemaValidationFailure
	for k := range resErrors {
		schemaErrors = append(schemaErrors, resErrors[k].SchemaValidationErrors...)
	}

	for c := range schemaErrors {

		r := model.BuildFunctionResultString(fmt.Sprintf("%s: %s", context.Rule.Description,
			schemaErrors[c].Reason))
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
