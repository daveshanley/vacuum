// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package core

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"strings"
)

// Xor is a rule that will check if one property or another has been set, but not both.
type Xor struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the Xor rule.
func (x Xor) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name:     "xor",
		Required: []string{"properties"},
		Properties: []model.RuleFunctionProperty{
			{
				Name:        "properties",
				Description: "'xor' requires two values",
			},
		},
		MinProperties: 2,
		MaxProperties: 2,
		ErrorMessage: "'xor' function has invalid options supplied. Example valid options are 'properties' = 'a, b'" +
			" or 'properties' = '1, 2'",
	}
}

// GetCategory returns the category of the Xor rule.
func (x Xor) GetCategory() string {
	return model.FunctionCategoryCore
}

// RunRule will execute the Xor rule, based on supplied context and a supplied []*yaml.Node slice.
func (x Xor) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	// check supplied properties, there can only be two
	props := utils.ConvertInterfaceIntoStringMap(context.Options)
	var properties []string

	if len(props) <= 0 {
		properties = utils.ConvertInterfaceToStringArray(context.Options)
	} else {
		properties = strings.Split(props["properties"], ",")
	}

	if len(properties) != 2 {
		return nil
	}

	pathValue := "unknown"
	if path, ok := context.Given.(string); ok {
		pathValue = path
	}

	var results []model.RuleFunctionResult
	seenCount := 0

	ruleMessage := context.Rule.Description
	message := context.Rule.Message

	for _, node := range nodes {

		// look through our properties for a match (or no match), the end result needs to be exactly 1.
		for _, v := range properties {
			fieldNode, _ := utils.FindKeyNode(strings.TrimSpace(v), node.Content)

			if fieldNode != nil && fieldNode.Value == strings.TrimSpace(v) {
				seenCount++
			}
		}

		if seenCount != 1 {
			results = append(results, model.RuleFunctionResult{
				Message: vacuumUtils.SuppliedOrDefault(message, fmt.Sprintf("%s: `%s` and `%s` must not be both defined or undefined",
					ruleMessage, properties[0], properties[1])),
				StartNode: node,
				EndNode:   vacuumUtils.BuildEndNode(node),
				Path:      pathValue,
				Rule:      context.Rule,
			})
		}
	}

	return results
}
