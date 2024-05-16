// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/base"
	"gopkg.in/yaml.v3"
	"slices"
)

type IntegerFormat struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the DefineError rule.
func (i IntegerFormat) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "integer_format"}
}

// GetCategory returns the category of the IntegerFormat rule.
func (i IntegerFormat) GetCategory() string {
	return model.FunctionCategoryOWASP
}

// RunRule will execute the DefineError rule, based on supplied context and a supplied []*yaml.Node slice.
func (i IntegerFormat) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	for _, schema := range context.DrDocument.Schemas {
		if slices.Contains(schema.Value.Type, "integer") {
			if schema.Value.Format == "" ||
				(schema.Value.Format != "int32" && schema.Value.Format != "int64") {

				node := schema.Value.GoLow().Type.KeyNode
				result := model.RuleFunctionResult{
					Message: vacuumUtils.SuppliedOrDefault(context.Rule.Message,
						"schema of type `integer` must specify a format of `int32` or `int64`"),
					StartNode: node,
					EndNode:   vacuumUtils.BuildEndNode(node),
					Path:      schema.GenerateJSONPath(),
					Rule:      context.Rule,
				}
				schema.AddRuleFunctionResult(base.ConvertRuleResult(&result))
				results = append(results, result)
				continue
			}
		}
	}
	return results
}
