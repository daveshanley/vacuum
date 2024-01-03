// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/base"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"strconv"
	"strings"
)

// ComponentDescription will check through all components and determine if they are correctly described
type ComponentDescription struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the ComponentDescription rule.
func (cd ComponentDescription) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "component_description"}
}

// RunRule will execute the ComponentDescription rule, based on supplied context and a supplied []*yaml.Node slice.
func (cd ComponentDescription) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	// check supplied type
	props := utils.ConvertInterfaceIntoStringMap(context.Options)

	minWordsString := props["minWords"]
	minWords, _ := strconv.Atoi(minWordsString)

	if context.DrDocument == nil {
		return results
	}

	components := context.DrDocument.V3Document.Components

	buildResult := func(message, path string, node *yaml.Node, component base.AcceptsRuleResults) model.RuleFunctionResult {
		result := model.RuleFunctionResult{
			Message:   message,
			StartNode: node,
			EndNode:   node,
			Path:      path,
			Rule:      context.Rule,
		}
		component.AddRuleFunctionResult(base.ConvertRuleResult(&result))
		return result
	}

	checkDescription := func(description string, componentName, componentType, path string, node *yaml.Node, component base.AcceptsRuleResults) {
		if description == "" {
			results = append(results,
				buildResult(vacuumUtils.SuppliedOrDefault(context.Rule.Message,
					fmt.Sprintf("`%s` component `%s` is missing a description", componentType, componentName)),
					path, node, component))
		} else {
			words := strings.Split(description, " ")
			if len(words) < minWords {
				results = append(results, buildResult(vacuumUtils.SuppliedOrDefault(context.Rule.Message,
					fmt.Sprintf("`%s` component `%s` description must be at least `%d` words long", componentType,
						componentName, minWords)),
					path, node, component))
			}
		}
	}

	if components != nil && components.Schemas != nil {
		for schemaPairs := components.Schemas.First(); schemaPairs != nil; schemaPairs = schemaPairs.Next() {
			schemaValue := schemaPairs.Value()
			key := schemaPairs.Key()

			checkDescription(schemaValue.Schema.Value.Description,
				key,
				"schemas",
				schemaValue.GenerateJSONPath(),
				schemaValue.Value.GetSchemaKeyNode(),
				schemaValue)
		}
	}

	if components != nil && components.Parameters != nil {
		for paramPairs := components.Parameters.First(); paramPairs != nil; paramPairs = paramPairs.Next() {
			paramValue := paramPairs.Value()
			key := paramPairs.Key()

			checkDescription(paramValue.Value.Description,
				key,
				"parameters",
				paramValue.GenerateJSONPath(),
				paramValue.Value.GoLow().RootNode,
				paramValue)

		}
	}

	if components != nil && components.RequestBodies != nil {
		for requestBodyPairs := components.RequestBodies.First(); requestBodyPairs != nil; requestBodyPairs = requestBodyPairs.Next() {
			rbValue := requestBodyPairs.Value()
			key := requestBodyPairs.Key()

			checkDescription(rbValue.Value.Description,
				key,
				"requestBodies",
				rbValue.GenerateJSONPath(),
				rbValue.Value.GoLow().RootNode,
				rbValue)
		}
	}

	if components != nil && components.Responses != nil {
		for responsePairs := components.Responses.First(); responsePairs != nil; responsePairs = responsePairs.Next() {
			rValue := responsePairs.Value()
			key := responsePairs.Key()

			checkDescription(rValue.Value.Description,
				key,
				"responses",
				rValue.GenerateJSONPath(),
				rValue.Value.GoLow().RootNode,
				rValue)
		}
	}

	if components != nil && components.Examples != nil {
		for examplePairs := components.Examples.First(); examplePairs != nil; examplePairs = examplePairs.Next() {
			exampleValue := examplePairs.Value()
			key := examplePairs.Key()

			checkDescription(exampleValue.Value.Description,
				key,
				"examples",
				exampleValue.GenerateJSONPath(),
				exampleValue.Value.GoLow().RootNode,
				exampleValue)
		}
	}

	if components != nil && components.Callbacks != nil {
		for callbackPairs := components.Examples.First(); callbackPairs != nil; callbackPairs = callbackPairs.Next() {
			callbackValue := callbackPairs.Value()
			key := callbackPairs.Key()

			checkDescription(callbackValue.Value.Description,
				key,
				"callbacks",
				callbackValue.GenerateJSONPath(),
				callbackValue.Value.GoLow().RootNode,
				callbackValue)
		}
	}

	if components != nil && components.Headers != nil {
		for headerPair := components.Examples.First(); headerPair != nil; headerPair = headerPair.Next() {
			headerValue := headerPair.Value()
			key := headerPair.Key()

			checkDescription(headerValue.Value.Description,
				key,
				"headers",
				headerValue.GenerateJSONPath(),
				headerValue.Value.GoLow().RootNode,
				headerValue)
		}
	}

	if components != nil && components.Links != nil {
		for linkPair := components.Examples.First(); linkPair != nil; linkPair = linkPair.Next() {
			linkValue := linkPair.Value()
			key := linkPair.Key()

			checkDescription(linkValue.Value.Description,
				key,
				"links",
				linkValue.GenerateJSONPath(),
				linkValue.Value.GoLow().RootNode,
				linkValue)

		}
	}

	if components != nil && components.SecuritySchemes != nil {
		for securitySchemePair := components.Examples.First(); securitySchemePair != nil; securitySchemePair = securitySchemePair.Next() {
			ssValue := securitySchemePair.Value()
			key := securitySchemePair.Key()

			checkDescription(ssValue.Value.Description,
				key,
				"securitySchemes",
				ssValue.GenerateJSONPath(),
				ssValue.Value.GoLow().RootNode,
				ssValue)

		}
	}
	return results
}
