// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package openapi

import (
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"go.yaml.in/yaml/v4"
)

// InfoContact will check that the info object has a contact object.
type InfoContact struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the InfoContact rule.
func (id InfoContact) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "infoContact",
	}
}

// GetCategory returns the category of the InfoContact rule.
func (id InfoContact) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the InfoContact rule, based on supplied context and a supplied []*yaml.Node slice.
func (id InfoContact) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	info := context.DrDocument.V3Document.Info

	if info != nil && info.Value.Contact == nil {
		res := model.RuleFunctionResult{
			Message:   vacuumUtils.SuppliedOrDefault(context.Rule.Message, "`info` section must contain `contact` details"),
			StartNode: info.Value.GoLow().KeyNode,
			EndNode:   vacuumUtils.BuildEndNode(info.Value.GoLow().KeyNode),
			Path:      "$.info",
			Rule:      context.Rule,
		}
		results = append(results, res)
		info.AddRuleFunctionResult(v3.ConvertRuleResult(&res))
	}

	return results
}
