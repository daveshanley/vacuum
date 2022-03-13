// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
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
func (cd ComponentDescription) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	// check supplied type
	props := utils.ConvertInterfaceIntoStringMap(context.Options)

	minWordsString := props["minWords"]
	minWords, _ := strconv.Atoi(minWordsString)

	components := GetComponentsFromRoot(nodes)

	var componentType, componentName string
	for i, componentNode := range components {
		if i%2 == 0 {
			componentType = componentNode.Value
			continue
		}

		for m, nameNode := range componentNode.Content {

			if m%2 == 0 {
				componentName = nameNode.Value
				continue
			}

			basePath := fmt.Sprintf("$.components.%s.%s", componentType, componentName)
			descKey, descNode := utils.FindKeyNode("description", nameNode.Content)

			if descNode == nil {

				res := createDescriptionResult(fmt.Sprintf("Component '%s' of type '%s' is missing a description",
					componentName, componentType), basePath, nameNode, nameNode)
				res.Rule = context.Rule
				results = append(results, res)
			} else {

				// check if description is above a certain length of words
				words := strings.Split(descNode.Value, " ")
				if len(words) < minWords {

					res := createDescriptionResult(fmt.Sprintf("Component '%s' of type '%s' description must be "+
						"at least %d words long, (%d is not enough)", componentName, componentType, minWords, len(words)), basePath, descKey, descNode)
					res.Rule = context.Rule
					results = append(results, res)
				}
			}
		}
	}
	return results
}
