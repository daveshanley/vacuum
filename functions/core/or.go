// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package core

import (
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/libopenapi/utils"
	"go.yaml.in/yaml/v4"
	"strings"
)

// Or is a rule that will check if one property or another has been set, but not both.
type Or struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the Or rule.
func (x Or) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name:     "or",
		Required: []string{"properties"},
		Properties: []model.RuleFunctionProperty{
			{
				Name: "properties",
				Description: "'or' requires at least two values, examples of valid options are 'a, b'" +
					" or '1, 2, 3' etc (do not include quotes)",
			},
		},
		MinProperties: 2,
		ErrorMessage: "'or' function has invalid options supplied. Example valid options are 'properties' = 'a, b'" +
			" or 'properties' = '1, 2, 3'",
	}
}

// GetCategory returns the category of the Or rule.
func (x Or) GetCategory() string {
	return model.FunctionCategoryCore
}

// RunRule will execute the Or rule, based on supplied context and a supplied []*yaml.Node slice.
func (x Or) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	// check supplied properties, there can only be two - use cached options
	props := context.GetOptionsStringMap()
	var properties []string

	if len(props) <= 0 {
		properties = utils.ConvertInterfaceToStringArray(context.Options)
	} else {
		properties = strings.Split(props["properties"], ",")
	}

	if len(properties) < 2 {
		return nil
	}

	pathValue := "unknown"
	if path, ok := context.Given.(string); ok {
		pathValue = path
	}

	var results []model.RuleFunctionResult

	ruleMessage := context.Rule.Description
	message := context.Rule.Message

	for _, node := range nodes {

		seenCount := 0

		// look through our properties for a match (or no match), the end result needs to be at least one.
		for _, v := range properties {
			fieldNode, _ := utils.FindKeyNode(strings.TrimSpace(v), node.Content)

			if fieldNode != nil && fieldNode.Value == strings.TrimSpace(v) {
				seenCount++
			}
		}

		if seenCount == 0 {
			locatedPath, allPaths, locatedObjects := locateModelPaths(context, node, pathValue)
			result := model.RuleFunctionResult{
				Message: vacuumUtils.SuppliedOrDefault(message,
					model.GetStringTemplates().BuildNoneDefinedMessage(ruleMessage, properties)),
				StartNode: node,
				EndNode:   vacuumUtils.BuildEndNode(node),
				Path:      locatedPath,
				Rule:      context.Rule,
			}
			if len(allPaths) > 1 {
				result.Paths = allPaths
			}
			results = append(results, result)
			addResultToLocatedModel(locatedObjects, &result)
		}
	}

	return results
}
