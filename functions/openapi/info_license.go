// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package openapi

import (
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/base"
	"gopkg.in/yaml.v3"
)

// InfoLicense will check that the info object has a contact object.
type InfoLicense struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the InfoLicense rule.
func (id InfoLicense) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "infoLicense",
	}
}

// GetCategory returns the category of the InfoLicense rule.
func (id InfoLicense) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the InfoLicense rule, based on supplied context and a supplied []*yaml.Node slice.
func (id InfoLicense) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	info := context.DrDocument.V3Document.Info

	if info != nil && info.Value.License == nil {
		res := model.RuleFunctionResult{
			Message:   vacuumUtils.SuppliedOrDefault(context.Rule.Message, "`info` section must contain a `license`"),
			StartNode: info.Value.GoLow().KeyNode,
			EndNode:   vacuumUtils.BuildEndNode(info.Value.GoLow().KeyNode),
			Path:      "$.info",
			Rule:      context.Rule,
		}
		results = append(results, res)
		info.AddRuleFunctionResult(base.ConvertRuleResult(&res))
	}

	return results
}
