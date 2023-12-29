// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/doctor/model/high/base"
	"gopkg.in/yaml.v3"
	"slices"
)

type ArrayLimit struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the DefineError rule.
func (ar ArrayLimit) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "array_limit"}
}

// RunRule will execute the DefineError rule, based on supplied context and a supplied []*yaml.Node slice.
func (ar ArrayLimit) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	for _, schema := range context.DrDocument.Schemas {
		if slices.Contains(schema.Value.Type, "array") {
			if schema.Value.MaxItems == nil {
				// no max items specified
				node := schema.Value.GoLow().Type.KeyNode
				result := model.RuleFunctionResult{
					Message:   "schema of type `array` must specify `maxItems`",
					StartNode: node,
					EndNode:   node,
					Path:      schema.GenerateJSONPath(),
					Rule:      context.Rule,
				}
				schema.AddRuleFunctionResult(base.ConvertRuleResult(&result))
				results = append(results, result)
			}
		}
	}
	return results
}
