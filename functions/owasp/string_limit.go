// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/doctor/model/high/base"
	"gopkg.in/yaml.v3"
	"slices"
)

type StringLimit struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the DefineError rule.
func (st StringLimit) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "string_limit"}
}

// RunRule will execute the DefineError rule, based on supplied context and a supplied []*yaml.Node slice.
func (st StringLimit) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	for _, schema := range context.DrDocument.Schemas {
		if slices.Contains(schema.Value.Type, "string") {
			if schema.Value.MaxLength == nil && schema.Value.Const == nil && schema.Value.Enum == nil {
				node := schema.Value.GoLow().Type.KeyNode
				result := model.RuleFunctionResult{
					Message:   "schema of type `string` must specify `maxLength`, `const` or `enum`",
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
