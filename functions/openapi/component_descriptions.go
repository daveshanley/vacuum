// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/base"
	"github.com/pb33f/libopenapi/datamodel/low"
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

// GetCategory returns the category of the ComponentDescription rule.
func (cd ComponentDescription) GetCategory() string {
	return model.FunctionCategoryOpenAPI
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
			EndNode:   vacuumUtils.BuildEndNode(node),
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

			k, _ := low.FindItemInOrderedMapWithKey(key, components.Value.GoLow().Schemas.Value)
			if schemaValue.Schema != nil && schemaValue.Schema.Value != nil {
				checkDescription(schemaValue.Schema.Value.Description,
					key,
					"schemas",
					schemaValue.GenerateJSONPath(),
					k.GetKeyNode(),
					schemaValue)
			}
		}
	}

	if components != nil && components.Parameters != nil {
		for paramPairs := components.Parameters.First(); paramPairs != nil; paramPairs = paramPairs.Next() {
			paramValue := paramPairs.Value()
			key := paramPairs.Key()
			k, _ := low.FindItemInOrderedMapWithKey(key, components.Value.GoLow().Parameters.Value)
			checkDescription(paramValue.Value.Description,
				key,
				"parameters",
				paramValue.GenerateJSONPath(),
				k.GetKeyNode(),
				paramValue)

		}
	}

	if components != nil && components.RequestBodies != nil {
		for requestBodyPairs := components.RequestBodies.First(); requestBodyPairs != nil; requestBodyPairs = requestBodyPairs.Next() {
			rbValue := requestBodyPairs.Value()
			key := requestBodyPairs.Key()
			k, _ := low.FindItemInOrderedMapWithKey(key, components.Value.GoLow().RequestBodies.Value)
			checkDescription(rbValue.Value.Description,
				key,
				"requestBodies",
				rbValue.GenerateJSONPath(),
				k.GetKeyNode(),
				rbValue)
		}
	}

	if components != nil && components.Responses != nil {
		for responsePairs := components.Responses.First(); responsePairs != nil; responsePairs = responsePairs.Next() {
			rValue := responsePairs.Value()
			key := responsePairs.Key()
			k, _ := low.FindItemInOrderedMapWithKey(key, components.Value.GoLow().Responses.Value)
			checkDescription(rValue.Value.Description,
				key,
				"responses",
				rValue.GenerateJSONPath(),
				k.GetKeyNode(),
				rValue)
		}
	}

	if components != nil && components.Examples != nil {
		for examplePairs := components.Examples.First(); examplePairs != nil; examplePairs = examplePairs.Next() {
			exampleValue := examplePairs.Value()
			key := examplePairs.Key()
			k, _ := low.FindItemInOrderedMapWithKey(key, components.Value.GoLow().Examples.Value)
			checkDescription(exampleValue.Value.Description,
				key,
				"examples",
				exampleValue.GenerateJSONPath(),
				k.GetKeyNode(),
				exampleValue)
		}
	}

	if components != nil && components.Headers != nil {
		for headerPair := components.Headers.First(); headerPair != nil; headerPair = headerPair.Next() {
			headerValue := headerPair.Value()
			key := headerPair.Key()
			k, _ := low.FindItemInOrderedMapWithKey(key, components.Value.GoLow().Headers.Value)
			checkDescription(headerValue.Value.Description,
				key,
				"headers",
				headerValue.GenerateJSONPath(),
				k.GetKeyNode(),
				headerValue)
		}
	}

	if components != nil && components.Links != nil {
		for linkPair := components.Links.First(); linkPair != nil; linkPair = linkPair.Next() {
			linkValue := linkPair.Value()
			key := linkPair.Key()
			k, _ := low.FindItemInOrderedMapWithKey(key, components.Value.GoLow().Links.Value)
			checkDescription(linkValue.Value.Description,
				key,
				"links",
				linkValue.GenerateJSONPath(),
				k.GetKeyNode(),
				linkValue)

		}
	}

	if components != nil && components.SecuritySchemes != nil {
		for securitySchemePair := components.SecuritySchemes.First(); securitySchemePair != nil; securitySchemePair = securitySchemePair.Next() {
			ssValue := securitySchemePair.Value()
			key := securitySchemePair.Key()
			k, _ := low.FindItemInOrderedMapWithKey(key, components.Value.GoLow().SecuritySchemes.Value)
			checkDescription(ssValue.Value.Description,
				key,
				"securitySchemes",
				ssValue.GenerateJSONPath(),
				k.GetKeyNode(),
				ssValue)

		}
	}
	return results
}
