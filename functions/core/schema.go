// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package core

import (
	ctx "context"
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/parser"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	validationErrors "github.com/pb33f/libopenapi-validator/errors"
	highBase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/datamodel/low"
	lowBase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"strings"
)

// Schema is a rule that creates a schema check against a field value
type Schema struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the OperationParameters rule.
func (sch Schema) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name:     "schema",
		Required: []string{"schema"},
		Properties: []model.RuleFunctionProperty{
			{
				Name:        "schema",
				Description: "A valid JSON Schema object that will be used to validate",
			},
			{
				Name:        "unpack",
				Description: "Treat the parent node as a document node and unpack it (default is false)",
			},
			{
				Name:        "forceValidation",
				Description: "Force a failure if the field is not found (default is false)",
			},
			{
				Name:        "forceValidationOnCurrentNode",
				Description: "Ignore the field value of the action, and validate the current node from JSON Path (default is false)",
			},
		},
		ErrorMessage: "`schema` function needs a `schema` property to be supplied at a minimum",
	}
}

// GetCategory returns the category of the OperationParameters rule.
func (sch Schema) GetCategory() string {
	return model.FunctionCategoryCore
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

	message := context.Rule.Message

	var schema *highBase.Schema
	var ok bool

	ruleMessage := context.Rule.Description
	if context.Rule.Message != "" {
		ruleMessage = context.Rule.Message
	}

	s := utils.ExtractValueFromInterfaceMap("schema", context.Options)
	if schema, ok = s.(*highBase.Schema); !ok {

		// build schema from scratch
		var lowSchema lowBase.Schema

		// unmarshal the schema
		var on yaml.Node
		err := on.Encode(&s)

		if err != nil {
			r := model.BuildFunctionResultString(
				vacuumUtils.SuppliedOrDefault(message, fmt.Sprintf("unable to parse function options: %s", err.Error())))
			r.Rule = context.Rule
			results = append(results, r)
			return results
		}

		// first, run the model builder on the schema
		err = low.BuildModel(&on, &lowSchema)
		if err != nil {
			r := model.BuildFunctionResultString(
				vacuumUtils.SuppliedOrDefault(message,
					fmt.Sprintf("unable to build low schema from function options: %s", err.Error())))
			r.Rule = context.Rule
			results = append(results, r)
			return results
		}

		// now build out the low level schema.
		err = lowSchema.Build(ctx.Background(), &on, context.Index)
		if err != nil {
			r := model.BuildFunctionResultString(
				vacuumUtils.SuppliedOrDefault(message,
					fmt.Sprintf("unable to build high schema from function options: %s", err.Error())))
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
		schema.GoLow().Index = context.Index
		results = append(results, validateNodeAgainstSchema(&context, schema, nodes[0], context, 0)...)
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
			schema.GoLow().Index = context.Index
			results = append(results, validateNodeAgainstSchema(&context, schema, field, context, x)...)

		} else {
			// If the field is not found, and we're being strict, it's invalid.
			forceValidation := utils.ExtractValueFromInterfaceMap("forceValidation", context.Options)
			if _, ko := forceValidation.(bool); ko {

				locatedObject, err := context.DrDocument.LocateModel(node)
				locatedPath := ""
				if err == nil && locatedObject != nil {
					locatedPath = locatedObject.GenerateJSONPath()
				}

				r := model.BuildFunctionResultString(
					vacuumUtils.SuppliedOrDefault(message, fmt.Sprintf("%s: %s", ruleMessage,
						fmt.Sprintf("`%s`, is missing and is required", context.RuleAction.Field))))
				r.StartNode = node
				r.EndNode = vacuumUtils.BuildEndNode(node)
				r.Rule = context.Rule
				r.Path = locatedPath
				if p, df := context.Given.(string); df {
					r.Path = fmt.Sprintf("%s[%d]", p, x)
				}
				results = append(results, r)
			}
		}
	}
	return results
}

var bannedErrors = []string{"if-then failed", "if-else failed", "allOf failed", "oneOf failed"}

func validateNodeAgainstSchema(ctx *model.RuleFunctionContext, schema *highBase.Schema, field *yaml.Node,
	context model.RuleFunctionContext, x int) []model.RuleFunctionResult {

	ruleMessage := context.Rule.Description
	if context.Rule.Message != "" {
		ruleMessage = context.Rule.Message
	}

	var results []model.RuleFunctionResult

	// validate using schema provided.
	res, resErrors := parser.ValidateNodeAgainstSchema(ctx, schema, field, false)

	if res {
		return results
	}

	var schemaErrors []*validationErrors.SchemaValidationFailure
	for k := range resErrors {
		schemaErrors = append(schemaErrors, resErrors[k].SchemaValidationErrors...)
	}

	message := context.Rule.Message

	for c := range schemaErrors {
		locatedObject, err := context.DrDocument.LocateModel(field)
		locatedPath := ""
		if err == nil && locatedObject != nil {
			locatedPath = locatedObject.GenerateJSONPath()
		}

		r := model.BuildFunctionResultString(vacuumUtils.SuppliedOrDefault(message, fmt.Sprintf("%s: %s", ruleMessage,
			schemaErrors[c].Reason)))
		r.StartNode = field
		r.EndNode = vacuumUtils.BuildEndNode(field)
		r.Rule = context.Rule
		r.Path = locatedPath
		if p, ok := context.Given.(string); ok {
			r.Path = fmt.Sprintf("%s[%d]", p, x)
		}
		if p, ok := context.Given.([]string); ok {
			r.Path = fmt.Sprintf("%s[%d]", p[0], x)
		}

		banned := false
		for g := range bannedErrors {
			if strings.Contains(schemaErrors[c].Reason, bannedErrors[g]) {
				banned = true
				continue
			}
		}

		if !banned {
			results = append(results, r)
		}

	}
	return results
}
