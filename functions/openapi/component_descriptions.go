// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/go-yaml"
)

// ComponentDescription will check through all components and determine if they are correctly described
type ComponentDescription struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the ComponentDescription rule.
func (cd ComponentDescription) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "oasComponentDescriptions",
		Properties: []model.RuleFunctionProperty{
			{
				Name:        "minWords",
				Description: "Minimum number of words required in a description, defaults to '0'",
			},
		},
		ErrorMessage: "'oasComponentDescriptions' function has invalid options supplied. Set the 'minWords' property to a valid integer",
	}
}

// GetCategory returns the category of the ComponentDescription rule.
func (cd ComponentDescription) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the ComponentDescription rule, based on supplied context and a supplied []*yaml.Node slice.
func (cd ComponentDescription) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	// check supplied type - use cached options
	props := context.GetOptionsStringMap()

	minWordsString := props["minWords"]
	minWords, _ := strconv.Atoi(minWordsString)

	if context.DrDocument == nil {
		return results
	}

	components := context.DrDocument.V3Document.Components

	buildResult := func(message, path string, node *yaml.Node,
		component v3.AcceptsRuleResults, paths []string) model.RuleFunctionResult {
		result := model.RuleFunctionResult{
			Message:   message,
			StartNode: node,
			EndNode:   vacuumUtils.BuildEndNode(node),
			Path:      path,
			Rule:      context.Rule,
		}
		if len(paths) > 1 {
			result.Paths = paths
		}
		component.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
		return result
	}

	checkDescription := func(description string, componentName, componentType string,
		component v3.Foundational) {

		// Successful checks need neither result paths nor a split description.
		if description != "" && strings.Count(description, " ")+1 >= minWords {
			return
		}

		// all locations where this component is referenced
		node := component.GetKeyNode()
		primaryPath, allPaths := vacuumUtils.LocateComponentPaths(context, component, node, node)

		acceptsResults, _ := component.(v3.AcceptsRuleResults)

		if description == "" {
			results = append(results,
				buildResult(vacuumUtils.SuppliedOrDefault(context.Rule.Message,
					fmt.Sprintf("`%s` component `%s` is missing a description", componentType, componentName)),
					primaryPath, node, acceptsResults, allPaths))
		} else {
			results = append(results, buildResult(vacuumUtils.SuppliedOrDefault(context.Rule.Message,
				fmt.Sprintf("`%s` component `%s` description must be at least `%d` words long",
					componentType, componentName, minWords)),
				primaryPath, node, acceptsResults, allPaths))
		}
	}

	if components != nil && components.Schemas != nil {
		for key, schemaValue := range components.Schemas.FromOldest() {
			if schemaValue.Schema != nil && schemaValue.Schema.Value != nil {
				checkDescription(schemaValue.Schema.Value.Description,
					key,
					"schemas",
					schemaValue)
			}
		}
	}

	if components != nil && components.Parameters != nil {
		for key, paramValue := range components.Parameters.FromOldest() {
			checkDescription(paramValue.Value.Description,
				key,
				"parameters",
				paramValue)

		}
	}

	if components != nil && components.RequestBodies != nil {
		for key, rbValue := range components.RequestBodies.FromOldest() {
			checkDescription(rbValue.Value.Description,
				key,
				"requestBodies",
				rbValue)
		}
	}

	if components != nil && components.Responses != nil {
		for key, rValue := range components.Responses.FromOldest() {
			checkDescription(rValue.Value.Description,
				key,
				"responses",
				rValue)
		}
	}

	if components != nil && components.Examples != nil {
		for key, exampleValue := range components.Examples.FromOldest() {
			checkDescription(exampleValue.Value.Description,
				key,
				"examples",
				exampleValue)
		}
	}

	if components != nil && components.Headers != nil {
		for key, headerValue := range components.Headers.FromOldest() {
			checkDescription(headerValue.Value.Description,
				key,
				"headers",
				headerValue)
		}
	}

	if components != nil && components.Links != nil {
		for key, linkValue := range components.Links.FromOldest() {
			checkDescription(linkValue.Value.Description,
				key,
				"links",
				linkValue)

		}
	}

	if components != nil && components.SecuritySchemes != nil {
		for key, ssValue := range components.SecuritySchemes.FromOldest() {
			checkDescription(ssValue.Value.Description,
				key,
				"securitySchemes",
				ssValue)

		}
	}
	return results
}
