// Copyright 2024 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package openapi

import (
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/base"
	"gopkg.in/yaml.v3"
)

// ExamplesMissing will check anything that can have an example, has one.
type ExamplesMissing struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the ComponentDescription rule.
func (em ExamplesMissing) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "examples_missing"}
}

// RunRule will execute the ComponentDescription rule, based on supplied context and a supplied []*yaml.Node slice.
func (em ExamplesMissing) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	buildResult := func(message, path string, node *yaml.Node, component base.AcceptsRuleResults) model.RuleFunctionResult {
		result := model.RuleFunctionResult{
			Message:   message,
			StartNode: node,
			EndNode:   node,
			Path:      path,
			Rule:      context.Rule,
		}
		component.AddRuleFunctionResult(base.ConvertRuleResult(&result))
		return result
	}

	if context.DrDocument.Schemas != nil {
		for i := range context.DrDocument.Schemas {
			s := context.DrDocument.Schemas[i]
			if s.Value.Examples == nil && s.Value.Example == nil {
				results = append(results,
					buildResult(vacuumUtils.SuppliedOrDefault(context.Rule.Message, "schema is missing examples"),
						s.GenerateJSONPath(),
						s.Value.ParentProxy.GetSchemaKeyNode(), s))

			}
		}
	}

	if context.DrDocument.Parameters != nil {
		for i := range context.DrDocument.Parameters {
			p := context.DrDocument.Parameters[i]
			if p.Value.Examples.Len() <= 0 {
				results = append(results,
					buildResult(vacuumUtils.SuppliedOrDefault(context.Rule.Message, "parameter is missing examples"),
						p.GenerateJSONPath(),
						p.Value.GoLow().RootNode, p))
			}
		}
	}

	if context.DrDocument.Headers != nil {
		for i := range context.DrDocument.Headers {
			h := context.DrDocument.Headers[i]
			if h.Value.Examples.Len() <= 0 {
				results = append(results,
					buildResult(vacuumUtils.SuppliedOrDefault(context.Rule.Message, "header is missing examples"),
						h.GenerateJSONPath(),
						h.Value.GoLow().RootNode, h))
			}
		}
	}

	if context.DrDocument.MediaTypes != nil {
		for i := range context.DrDocument.MediaTypes {
			h := context.DrDocument.MediaTypes[i]
			if h.Value.Examples.Len() <= 0 {
				results = append(results,
					buildResult(vacuumUtils.SuppliedOrDefault(context.Rule.Message, "media type is missing examples"),
						h.GenerateJSONPath(),
						h.Value.GoLow().RootNode, h))
			}
		}
	}

	return results
}
