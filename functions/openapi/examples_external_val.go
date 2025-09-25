// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package openapi

import (
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"go.yaml.in/yaml/v4"
)

// ExamplesExternalCheck checks Example objects don't use both `externalValue` and `value`.
type ExamplesExternalCheck struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the ComponentDescription rule.
func (eec ExamplesExternalCheck) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "oasExampleExternal"}
}

// GetCategory returns the category of the ComponentDescription rule.
func (eec ExamplesExternalCheck) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the ComponentDescription rule, based on supplied context and a supplied []*yaml.Node slice.
func (eec ExamplesExternalCheck) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	buildResult := func(message, path string, node *yaml.Node, component v3.AcceptsRuleResults) model.RuleFunctionResult {
		result := model.RuleFunctionResult{
			Message:   message,
			StartNode: node,
			EndNode:   vacuumUtils.BuildEndNode(node),
			Path:      path,
			Rule:      context.Rule,
		}
		component.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
		return result
	}

	checkExample := func(example *v3.Example) bool {
		if example.Value.Value != nil && example.Value.ExternalValue != "" {
			return false
		}
		return true
	}

	if context.DrDocument.Parameters != nil {
		for i := range context.DrDocument.Parameters {
			p := context.DrDocument.Parameters[i]
			for exp := p.Examples.First(); exp != nil; exp = exp.Next() {
				v := exp.Value()
				if !checkExample(v) {
					results = append(results,
						buildResult(vacuumUtils.SuppliedOrDefault(context.Rule.Message,
							"parameter example contains both `externalValue` and `value`"),
							v.GenerateJSONPath(),
							v.Value.GoLow().RootNode, v))
				}
			}
		}
	}

	if context.DrDocument.Headers != nil {
		for i := range context.DrDocument.Headers {
			h := context.DrDocument.Headers[i]
			for exp := h.Examples.First(); exp != nil; exp = exp.Next() {
				v := exp.Value()
				if !checkExample(v) {
					results = append(results,
						buildResult(vacuumUtils.SuppliedOrDefault(context.Rule.Message,
							"header example contains both `externalValue` and `value`"),
							v.GenerateJSONPath(),
							v.Value.GoLow().RootNode, v))
				}
			}
		}
	}

	if context.DrDocument.MediaTypes != nil {
		for i := range context.DrDocument.MediaTypes {
			mt := context.DrDocument.MediaTypes[i]
			for exp := mt.Examples.First(); exp != nil; exp = exp.Next() {
				v := exp.Value()
				if !checkExample(v) {
					results = append(results,
						buildResult(vacuumUtils.SuppliedOrDefault(context.Rule.Message,
							"media type example contains both `externalValue` and `value`"),
							v.GenerateJSONPath(),
							v.Value.GoLow().RootNode, v))
				}
			}
		}
	}

	return results
}
