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

type IntegerLimit struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the DefineError rule.
func (il IntegerLimit) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "owaspIntegerLimit"}
}

// GetCategory returns the category of the IntegerLimit rule.
func (il IntegerLimit) GetCategory() string {
	return model.FunctionCategoryOWASP
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
				Message: vacuumUtils.SuppliedOrDefault(context.Rule.Message,
					"schema of type `integer` must specify `minimum` and `maximum` or "+
						"`exclusiveMinimum` and `exclusiveMaximum`"),
				StartNode: node,
				EndNode:   vacuumUtils.BuildEndNode(node),
				Path:      schema.GenerateJSONPath(),
				Rule:      context.Rule,
			}
			if schema.Value.Minimum == nil && schema.Value.Maximum == nil &&
				schema.Value.ExclusiveMinimum == nil && schema.Value.ExclusiveMaximum == nil {
				schema.AddRuleFunctionResult(base.ConvertRuleResult(&result))
				results = append(results, result)
				continue
			}

			_min := schema.Value.Minimum
			_max := schema.Value.Maximum
			_exMin := schema.Value.ExclusiveMinimum
			_exMax := schema.Value.ExclusiveMaximum

			// we got nothing.
			if _min == nil && _max == nil && _exMin == nil && _exMax == nil {
				schema.AddRuleFunctionResult(base.ConvertRuleResult(&result))
				results = append(results, result)
				continue
			}

			// got a min but no max
			if _min != nil && _max == nil && _exMax == nil {
				schema.AddRuleFunctionResult(base.ConvertRuleResult(&result))
				results = append(results, result)
				continue
			}

			// got a max but no min
			if _min == nil && _max != nil && _exMin == nil {
				schema.AddRuleFunctionResult(base.ConvertRuleResult(&result))
				results = append(results, result)
				continue
			}

			// got an exclusive min but no max
			if _min == nil && _max == nil && _exMin != nil && _exMax == nil {
				schema.AddRuleFunctionResult(base.ConvertRuleResult(&result))
				results = append(results, result)
				continue
			}

			// got an exclusive max but no min
			if _min == nil && _max == nil && _exMin == nil && _exMax != nil {
				schema.AddRuleFunctionResult(base.ConvertRuleResult(&result))
				results = append(results, result)
				continue
			}

			// got a min but no exclusive max
			if _min != nil && _max == nil && _exMax == nil {
				schema.AddRuleFunctionResult(base.ConvertRuleResult(&result))
				results = append(results, result)
				continue
			}

			// got an exclusive min, min and exclusive max
			if _min != nil && _exMin != nil && _exMax != nil {
				schema.AddRuleFunctionResult(base.ConvertRuleResult(&result))
				results = append(results, result)
				continue
			}
		}
	}
	return results
}
