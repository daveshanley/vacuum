// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package core

import (
	ctx "context"
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/parser"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	validationErrors "github.com/pb33f/libopenapi-validator/errors"
	highBase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/datamodel/low"
	lowBase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"go.yaml.in/yaml/v4"
	"strings"
)

const (
	schemaErrorPlaceholder           = "{{error}}"
	interpolateErrorMessageOptionKey = "interpolateErrorMessage"
)

var buildSchemaLowModel = low.BuildModel

var buildSchemaLowDocument = func(node *yaml.Node, schema *lowBase.Schema, schemaIndex *index.SpecIndex) error {
	return schema.Build(ctx.Background(), node, schemaIndex)
}

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
			{
				Name:        interpolateErrorMessageOptionKey,
				Description: "Replace `{{error}}` in the rule message with schema error text (default is false)",
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
	interpolateErrorMessage := shouldInterpolateSchemaErrorMessage(context.Options)

	var schema *highBase.Schema
	var ok bool
	schemaIndex := context.Index

	ruleMessage := context.Rule.Description
	if context.Rule.Message != "" {
		ruleMessage = context.Rule.Message
	}

	s := utils.ExtractValueFromInterfaceMap("schema", context.Options)
	if schema, ok = s.(*highBase.Schema); !ok {

		// build schema from scratch
		var lowSchema lowBase.Schema

		// Prepare the schema from raw function options as a YAML node.
		// Using Node.Encode preserves the schema root node instead of wrapping it in a document node.
		var on yaml.Node
		err := encodeSchemaOptions(&on, s)
		if err != nil {
			defaultMessage := fmt.Sprintf("unable to parse function options: %s", err.Error())
			r := model.BuildFunctionResultString(
				buildSchemaResultMessage(message, defaultMessage, err.Error(), interpolateErrorMessage))
			r.Rule = context.Rule
			results = append(results, r)
			return results
		}

		// first, run the model builder on the schema
		err = buildSchemaLowModel(&on, &lowSchema)
		if err != nil {
			defaultMessage := fmt.Sprintf("unable to build low schema from function options: %s", err.Error())
			r := model.BuildFunctionResultString(
				buildSchemaResultMessage(message, defaultMessage, err.Error(), interpolateErrorMessage))
			r.Rule = context.Rule
			results = append(results, r)
			return results
		}

		// now build out the low level schema.
		err = buildSchemaLowDocument(&on, &lowSchema, schemaIndex)
		if err != nil {
			defaultMessage := fmt.Sprintf("unable to build high schema from function options: %s", err.Error())
			r := model.BuildFunctionResultString(
				buildSchemaResultMessage(message, defaultMessage, err.Error(), interpolateErrorMessage))
			r.Rule = context.Rule
			results = append(results, r)
			return results
		}

		// now, build the high level schema
		schema = highBase.NewSchema(&lowSchema)
	}

	// use the current node to validate (field not needed)
	forceValidationOnCurrentNode := utils.ExtractValueFromInterfaceMap("forceValidationOnCurrentNode", context.Options)

	// Auto-enable forceValidationOnCurrentNode when:
	// 1. The given path is "$" (root)
	// 2. No field is specified in the rule action
	// This makes the behavior consistent with Spectral for root-level schema validation
	autoForce := false
	if context.RuleAction.Field == "" {
		// Check if we're at the root level
		if givenStr, ok := context.Given.(string); ok && givenStr == "$" {
			autoForce = true
		} else if givenArr, ok := context.Given.([]string); ok && len(givenArr) > 0 && givenArr[0] == "$" {
			autoForce = true
		}
	}

	if forceBool, ok := forceValidationOnCurrentNode.(bool); (ok && forceBool) || autoForce {
		if len(nodes) > 0 {
			schema.GoLow().Index = schemaIndex
			results = append(results, validateNodeAgainstSchema(&context, schema, nil, nodes[0], context, 0)...)
			return results
		}
	}

	for x, node := range nodes {
		if x%2 == 0 && len(nodes) > 1 {
			continue
		}

		// if the node is a document node, skip down one level
		searchNode := node
		if searchNode.Kind == yaml.DocumentNode && len(searchNode.Content) > 0 {
			searchNode = searchNode.Content[0]
		}

		result := vacuumUtils.FindFieldPath(context.RuleAction.Field, searchNode.Content, vacuumUtils.FieldPathOptions{})
		fieldNode, fieldNodeValue := result.KeyNode, result.ValueNode
		if fieldNodeValue != nil {
			schema.GoLow().Index = schemaIndex
			results = append(results, validateNodeAgainstSchema(&context, schema, fieldNode, fieldNodeValue, context, x)...)

		} else {
			// If the field is not found, and we're being strict, it's invalid.
			forceValidation := utils.ExtractValueFromInterfaceMap("forceValidation", context.Options)
			if _, ko := forceValidation.(bool); ko {

				var locatedObjects []v3.Foundational
				var allPaths []string
				var err error
				locatedPath := ""

				if context.DrDocument != nil {
					locatedObjects, err = context.DrDocument.LocateModelsByKeyAndValue(fieldNode, fieldNodeValue)
					if err == nil && locatedObjects != nil {
						for q, obj := range locatedObjects {
							if q == 0 {
								locatedPath = obj.GenerateJSONPath()
							}
							allPaths = append(allPaths, obj.GenerateJSONPath())
						}
					}
				}

				errorText := fmt.Sprintf("`%s`, is missing and is required", context.RuleAction.Field)
				defaultMessage := fmt.Sprintf("%s: %s", ruleMessage, errorText)
				r := model.BuildFunctionResultString(
					buildSchemaResultMessage(message, defaultMessage, errorText, interpolateErrorMessage))
				r.StartNode = node
				r.EndNode = vacuumUtils.BuildEndNode(node)
				r.Rule = context.Rule
				r.Path = locatedPath
				if len(allPaths) > 1 {
					r.Paths = allPaths
				}
				if p, df := context.Given.(string); df {
					r.Path = fmt.Sprintf("%s[%d]", p, x)
				}
				results = append(results, r)
				if len(locatedObjects) > 0 {
					if arr, kk := locatedObjects[0].(v3.AcceptsRuleResults); kk {
						arr.AddRuleFunctionResult(v3.ConvertRuleResult(&r))
					}
				}
			}
		}
	}
	return results
}

var bannedErrors = []string{"if-then failed", "if-else failed", "allOf failed", "oneOf failed"}

func validateNodeAgainstSchema(ctx *model.RuleFunctionContext, schema *highBase.Schema, fieldNode, fieldNodeValue *yaml.Node,
	context model.RuleFunctionContext, x int) []model.RuleFunctionResult {

	ruleMessage := context.Rule.Description
	if context.Rule.Message != "" {
		ruleMessage = context.Rule.Message
	}

	var results []model.RuleFunctionResult
	message := context.Rule.Message
	interpolateErrorMessage := shouldInterpolateSchemaErrorMessage(context.Options)

	// validate using schema provided.
	res, resErrors := parser.ValidateNodeAgainstSchema(ctx, schema, fieldNodeValue, false)

	if res {
		return results
	}

	var schemaErrors []*validationErrors.SchemaValidationFailure
	for k := range resErrors {
		schemaErrors = append(schemaErrors, resErrors[k].SchemaValidationErrors...)
	}

	for c := range schemaErrors {

		var locatedObjects []v3.Foundational
		var allPaths []string
		var err error
		locatedPath := ""
		if fieldNode != nil {
			if context.DrDocument != nil {
				locatedObjects, err = context.DrDocument.LocateModelsByKeyAndValue(fieldNode, fieldNodeValue)
				if err == nil && locatedObjects != nil {
					for x, obj := range locatedObjects {
						if x == 0 {
							locatedPath = obj.GenerateJSONPath()
						}
						allPaths = append(allPaths, obj.GenerateJSONPath())
					}
				}
			}
		} else {
			locatedObjects, err = context.DrDocument.LocateModel(fieldNodeValue)
			if err == nil && locatedObjects != nil {
				for s, obj := range locatedObjects {
					if s == 0 {
						locatedPath = obj.GenerateJSONPath()
					}
					allPaths = append(allPaths, obj.GenerateJSONPath())
				}
			}
		}

		defaultMessage := fmt.Sprintf("%s: %s", ruleMessage, schemaErrors[c].Reason)
		r := model.BuildFunctionResultString(buildSchemaResultMessage(message, defaultMessage, schemaErrors[c].Reason, interpolateErrorMessage))
		r.StartNode = fieldNodeValue
		r.EndNode = vacuumUtils.BuildEndNode(fieldNodeValue)
		r.Rule = context.Rule
		r.Path = locatedPath
		if len(allPaths) > 1 {
			r.Paths = allPaths
		}
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
			if len(locatedObjects) > 0 {
				if arr, kk := locatedObjects[0].(v3.AcceptsRuleResults); kk {
					arr.AddRuleFunctionResult(v3.ConvertRuleResult(&r))
				}
			}
		}

	}
	return results
}

func encodeSchemaOptions(node *yaml.Node, schema interface{}) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("%v", recovered)
		}
	}()
	return node.Encode(schema)
}

func shouldInterpolateSchemaErrorMessage(options interface{}) bool {
	interpolate := utils.ExtractValueFromInterfaceMap(interpolateErrorMessageOptionKey, options)
	enabled, _ := interpolate.(bool)
	return enabled
}

func buildSchemaResultMessage(messageTemplate, defaultMessage, errorText string, interpolate bool) string {
	if messageTemplate == "" {
		return defaultMessage
	}
	if !interpolate || !strings.Contains(messageTemplate, schemaErrorPlaceholder) {
		return messageTemplate
	}
	return strings.ReplaceAll(messageTemplate, schemaErrorPlaceholder, errorText)
}
