// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/doctor/model/high/base"
	"gopkg.in/yaml.v3"
	"slices"
)

type IntegerLimit struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the DefineError rule.
func (il IntegerLimit) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "integer_limit"}
}

// RunRule will execute the DefineError rule, based on supplied context and a supplied []*yaml.Node slice.
func (il IntegerLimit) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	for _, schema := range context.DrDocument.Schemas {
		if slices.Contains(schema.Value.Type, "integer") {

			node := schema.Value.GoLow().Type.KeyNode
			result := model.RuleFunctionResult{
				Message: "schema of type `string` must specify `minimum` and `maximum` or " +
					"`exclusiveMinimum` and `exclusiveMaximum`",
				StartNode: node,
				EndNode:   node,
				Path:      schema.GenerateJSONPath(),
				Rule:      context.Rule,
			}
			if schema.Value.Minimum == nil && schema.Value.Maximum == nil &&
				schema.Value.ExclusiveMinimum == nil && schema.Value.ExclusiveMaximum == nil {
				schema.AddRuleFunctionResult(base.ConvertRuleResult(&result))
				results = append(results, result)
				continue
			}

			if schema.Value.Minimum != nil || schema.Value.Maximum != nil {
				if schema.Value.Minimum == nil && schema.Value.Maximum != nil {
					schema.AddRuleFunctionResult(base.ConvertRuleResult(&result))
					results = append(results, result)
					continue
				}
				if schema.Value.Minimum != nil && schema.Value.Maximum == nil {
					schema.AddRuleFunctionResult(base.ConvertRuleResult(&result))
					results = append(results, result)
					continue
				}
			}
			if schema.Value.ExclusiveMinimum == nil && schema.Value.ExclusiveMaximum == nil {
				if schema.Value.ExclusiveMinimum == nil && schema.Value.ExclusiveMaximum != nil {
					schema.AddRuleFunctionResult(base.ConvertRuleResult(&result))
					results = append(results, result)
					continue
				}
				if schema.Value.ExclusiveMinimum != nil && schema.Value.ExclusiveMaximum == nil {
					schema.AddRuleFunctionResult(base.ConvertRuleResult(&result))
					results = append(results, result)
					continue
				}
			}
		}
	}
	return results
}
