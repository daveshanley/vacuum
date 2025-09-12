// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package openapi

import (
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"go.yaml.in/yaml/v4"
)

// InfoDescription will check that the info section has a description.
type InfoDescription struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the InfoDescription rule.
func (id InfoDescription) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "infoDescription",
	}
}

// GetCategory returns the category of the InfoDescription rule.
func (id InfoDescription) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the InfoDescription rule, based on supplied context and a supplied []*yaml.Node slice.
func (id InfoDescription) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	info := context.DrDocument.V3Document.Info

	if info != nil && info.Value.Description == "" {
		res := model.RuleFunctionResult{
			Message:   vacuumUtils.SuppliedOrDefault(context.Rule.Message, "`info` section must have a `description`"),
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
