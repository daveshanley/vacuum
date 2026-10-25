// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package core

import (
	ctx "context"
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/parser"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi-validator/config"
	validationErrors "github.com/pb33f/libopenapi-validator/errors"
	"github.com/pb33f/libopenapi-validator/schema_validation"
	highBase "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/datamodel/low"
	lowBase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
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
				Name: "allErrors", Description: "Report all schema failures (default is false)",
			},
			{Name: "dialect", Description: "JSON Schema dialect (default is draft7)"},
			{
				Name:        "unpack",
				Description: "Treat the parent node as a document node and unpack it (default is false)",
			},
			{
				Name:        "forceValidation",
				Description: "Fail if the field is not found (default is true)",
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

// ValidateOptions checks the types and dialect names accepted by schema.
func (sch Schema) ValidateOptions(options any) error {
	if value := utils.ExtractValueFromInterfaceMap("allErrors", options); value != nil {
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("schema allErrors must be a boolean")
		}
	}
	if value := utils.ExtractValueFromInterfaceMap("dialect", options); value != nil {
		dialect, ok := value.(string)
		if !ok {
			return fmt.Errorf("schema dialect must be a string")
		}
		switch dialect {
		case "draft4", "draft6", "draft7", "draft2019-09", "draft2020-12":
		default:
			return fmt.Errorf("unsupported schema dialect %q", dialect)
		}
	}
	return nil
}

// RunRule will execute the Schema function
func (sch Schema) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	if err := sch.ValidateOptions(context.Options); err != nil {
		if context.ReportError != nil {
			context.ReportError(err)
		}
		return []model.RuleFunctionResult{{Message: err.Error(), Rule: context.Rule, StartNode: nodes[0], EndNode: nodes[0]}}
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

		// Prefer a root reference's target in the options. Keep the input index
		// for existing document-relative and external reference rules.
		if isRef, _, ref := utils.IsNodeRefValue(&on); isRef && strings.HasPrefix(ref, "#/") &&
			index.FindComponent(ctx.Background(), &on, ref, "", nil) != nil {
			schemaIndex = index.NewSpecIndexWithConfig(&on, index.CreateClosedAPIIndexConfig())
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

	// Each rule owns its validator so format checks do not change other functions.
	validator := schema_validation.NewSchemaValidatorWithLogger(context.Logger, config.WithFormatAssertions())
	defer validator.Release()
	context.SchemaValidator = validator

	// use the current node to validate (field not needed)

	schema = cloneSchemaForCoreValidation(schema, schemaIndex)
	// Copy the high-level wrapper before setting the dialect on caller-owned schemas.
	ownedSchema := *schema
	schema = &ownedSchema
	if schema.SchemaTypeRef == "" {
		dialect, _ := utils.ExtractValueFromInterfaceMap("dialect", context.Options).(string)
		switch dialect {
		case "", "draft7":
			schema.SchemaTypeRef = "http://json-schema.org/draft-07/schema#"
		case "draft4":
			schema.SchemaTypeRef = "http://json-schema.org/draft-04/schema#"
		case "draft6":
			schema.SchemaTypeRef = "http://json-schema.org/draft-06/schema#"
		case "draft2019-09":
			schema.SchemaTypeRef = "https://json-schema.org/draft/2019-09/schema"
		case "draft2020-12":
			schema.SchemaTypeRef = "https://json-schema.org/draft/2020-12/schema"
		default:
			r := model.BuildFunctionResultString(fmt.Sprintf("unsupported schema dialect %q", dialect))
			r.Rule = context.Rule
			return []model.RuleFunctionResult{r}
		}
	}
	forceValidationOnCurrentNode :=
		utils.ExtractValueFromInterfaceMap("forceValidationOnCurrentNode", context.Options)

	// Without a field, the selector has already identified the value to validate.
	if forceBool, _ := forceValidationOnCurrentNode.(bool); forceBool || context.RuleAction.Field == "" {
		for x, node := range nodes {
			results = append(results, validateNodeAgainstSchema(&context, schema, nil, node, context, x)...)
		}
		return results
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

		result := vacuumUtils.FindFieldPath(context.RuleAction.Field, searchNode.Content, fieldLookupOptions(context, false))
		fieldNode, fieldNodeValue := result.KeyNode, result.ValueNode
		if fieldNodeValue != nil {
			results = append(results, validateNodeAgainstSchema(&context, schema, fieldNode, fieldNodeValue, context, x)...)

		} else {
			// Missing values fail unless the rule explicitly makes this field optional.
			forceValidation := utils.ExtractValueFromInterfaceMap("forceValidation", context.Options)
			if enabled, specified := forceValidation.(bool); !specified || enabled {

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
				r.Path, r.Paths, locatedObjects = locateNodePaths(&context, node)
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

func cloneSchemaForCoreValidation(schema *highBase.Schema, schemaIndex *index.SpecIndex) *highBase.Schema {
	if schema == nil || schema.GoLow() == nil || schema.GoLow().GetRootNode() == nil {
		return schema
	}
	if schemaIndex == nil {
		return schema
	}
	if schema.GoLow().GetIndex() == schemaIndex {
		return schema
	}
	// Core rule schemas are expected to be self-contained. This clone prevents
	// validation from retargeting a shared schema's index.
	clonedRoot := utils.CloneYAMLNode(schema.GoLow().GetRootNode())
	if clonedRoot == nil {
		return schema
	}
	clonedSchema, err := parser.ConvertNodeIntoJSONSchema(clonedRoot, schemaIndex)
	if err != nil || clonedSchema == nil {
		return schema
	}
	return clonedSchema
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
		// Compilation and decoding failures must not become a clean result.
		if len(resErrors[k].SchemaValidationErrors) == 0 {
			reason := resErrors[k].Reason
			if reason == "" {
				reason = resErrors[k].Message
			}
			schemaErrors = append(schemaErrors, &validationErrors.SchemaValidationFailure{Reason: reason})
		}

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
			if context.DrDocument != nil {
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
		if context.RuleAction.Field == "" {
			r.Path, r.Paths, locatedObjects = locateNodePaths(&context, fieldNodeValue)
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
			if allErrors, _ := utils.ExtractValueFromInterfaceMap("allErrors", context.Options).(bool); !allErrors {
				break
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
