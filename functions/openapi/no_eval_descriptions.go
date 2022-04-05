// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"gopkg.in/yaml.v3"
)

// NoEvalInDescriptions will check if a description contains potentially malicious javascript
type NoEvalInDescriptions struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the NoEvalInDescriptions rule.
func (ne NoEvalInDescriptions) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "no_eval_descriptions"}
}

// RunRule will execute the NoEvalInDescriptions rule, based on supplied context and a supplied []*yaml.Node slice.
func (ne NoEvalInDescriptions) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	descriptions := context.Index.GetAllDescriptions()
	compiledRegex := context.Rule.PrecomiledPattern

	for _, desc := range descriptions {

		if compiledRegex.MatchString(desc.Content) {

			startNode := desc.Node
			endNode := desc.Node

			results = append(results, model.RuleFunctionResult{
				Message:   "description contains an 'eval()' statement, forbidden",
				StartNode: startNode,
				EndNode:   endNode,
				Path:      desc.Path,
				Rule:      context.Rule,
			})
		}
	}

	return results

}
