// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"go.yaml.in/yaml/v4"
)

// NoEvalInDescriptions will check if a description contains potentially malicious javascript
type NoEvalInDescriptions struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the NoEvalInDescriptions rule.
func (ne NoEvalInDescriptions) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name:          "noEvalDescription",
		Required:      []string{"pattern"},
		MinProperties: 1,
		Properties: []model.RuleFunctionProperty{
			{
				Name:        "pattern",
				Description: "Regular expression to match against the description content. ",
			},
		},
		ErrorMessage: "'noEvalDescription' function has invalid options supplied. Set the 'pattern' property to a valid regular expression",
	}
}

// GetCategory returns the category of the NoEvalInDescriptions rule.
func (ne NoEvalInDescriptions) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the NoEvalInDescriptions rule, based on supplied context and a supplied []*yaml.Node slice.
func (ne NoEvalInDescriptions) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	// check supplied type - use cached options
	props := context.GetOptionsStringMap()

	pattern := props["pattern"]

	descriptions := context.Index.GetAllDescriptions()
	compiledRegex := context.Rule.PrecompiledPattern
	if compiledRegex == nil {
		compiledRegex = model.CompileRegex(context, pattern, &results)
		if compiledRegex == nil {
			return results
		}
	}

	for _, desc := range descriptions {

		if compiledRegex.MatchString(desc.Content) {

			startNode := desc.Node
			endNode := desc.Node

			results = append(results, model.RuleFunctionResult{
				Message:   fmt.Sprintf("description contains content with `%s`, forbidden", pattern),
				StartNode: startNode,
				EndNode:   vacuumUtils.BuildEndNode(endNode),
				Path:      desc.Path,
				Rule:      context.Rule,
			})
		}
	}

	return results

}
