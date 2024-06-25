// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package openapi

import (
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/base"
	"gopkg.in/yaml.v3"
)

// InfoLicenseURL will check that the info object has a contact object.
type InfoLicenseURL struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the InfoLicenseURL rule.
func (id InfoLicenseURL) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "infoLicenseURL",
	}
}

// GetCategory returns the category of the InfoLicenseURL rule.
func (id InfoLicenseURL) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the InfoLicenseURL rule, based on supplied context and a supplied []*yaml.Node slice.
func (id InfoLicenseURL) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	info := context.DrDocument.V3Document.Info

	if info != nil && info.License != nil && info.License.Value.URL == "" {
		res := model.RuleFunctionResult{
			Message:   vacuumUtils.SuppliedOrDefault(context.Rule.Message, "`license` section must contain a `url`"),
			StartNode: info.License.Value.GoLow().KeyNode,
			EndNode:   vacuumUtils.BuildEndNode(info.License.Value.GoLow().KeyNode),
			Path:      "$.info.license",
			Rule:      context.Rule,
		}
		results = append(results, res)
		info.AddRuleFunctionResult(base.ConvertRuleResult(&res))
	}

	return results
}
