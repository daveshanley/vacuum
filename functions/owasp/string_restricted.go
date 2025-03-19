// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
    "github.com/daveshanley/vacuum/model"
    vacuumUtils "github.com/daveshanley/vacuum/utils"
    "github.com/pb33f/doctor/model/high/v3"
    "gopkg.in/yaml.v3"
    "slices"
)

type StringRestricted struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the DefineError rule.
func (st StringRestricted) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "owaspStringRestricted"}
}

// GetCategory returns the category of the StringRestricted rule.
func (st StringRestricted) GetCategory() string {
	return model.FunctionCategoryOWASP
}

// RunRule will execute the DefineError rule, based on supplied context and a supplied []*yaml.Node slice.
func (st StringRestricted) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	for _, schema := range context.DrDocument.Schemas {
		if slices.Contains(schema.Value.Type, "string") {
			if schema.Value.Format == "" &&
				schema.Value.Const == nil &&
				schema.Value.Enum == nil &&
				schema.Value.Pattern == "" {

				node := schema.Value.GoLow().Type.KeyNode
				result := model.RuleFunctionResult{
					Message: vacuumUtils.SuppliedOrDefault(context.Rule.Message,
						"schema of type `string` must specify `format`, `const`, `enum` or `pattern`"),
					StartNode: node,
					EndNode:   vacuumUtils.BuildEndNode(node),
					Path:      schema.GenerateJSONPath(),
					Rule:      context.Rule,
				}
				schema.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
				results = append(results, result)
			}
		}
	}
	return results
}
