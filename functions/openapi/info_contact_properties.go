// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package openapi

import (
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/base"
	"gopkg.in/yaml.v3"
)

// InfoContactProperties will check that the info object has a contact object.
type InfoContactProperties struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the InfoContactProperties rule.
func (id InfoContactProperties) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "infoContactProperties",
	}
}

// GetCategory returns the category of the InfoContactProperties rule.
func (id InfoContactProperties) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the InfoContactProperties rule, based on supplied context and a supplied []*yaml.Node slice.
func (id InfoContactProperties) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	info := context.DrDocument.V3Document.Info
	if info != nil && info.Value.Contact != nil {

		items := []string{"name", "url", "email"}

		for _, item := range items {
			switch item {
			case "name":
				if info.Value.Contact.Name == "" {
					res := model.RuleFunctionResult{
						Message:   vacuumUtils.SuppliedOrDefault(context.Rule.Message, "`contact` section must contain a `name`"),
						StartNode: info.Contact.Value.GoLow().KeyNode,
						EndNode:   vacuumUtils.BuildEndNode(info.Contact.Value.GoLow().KeyNode),
						Rule:      context.Rule,
						Path:      "$.info.contact",
					}
					results = append(results, res)
					info.Contact.AddRuleFunctionResult(base.ConvertRuleResult(&res))
				}

			case "url":
				if info.Value.Contact.URL == "" {
					res := model.RuleFunctionResult{
						Message:   vacuumUtils.SuppliedOrDefault(context.Rule.Message, "`contact` section must contain a `url`"),
						StartNode: info.Contact.Value.GoLow().KeyNode,
						EndNode:   vacuumUtils.BuildEndNode(info.Contact.Value.GoLow().KeyNode),
						Rule:      context.Rule,
						Path:      "$.info.contact",
					}
					results = append(results, res)
					info.Contact.AddRuleFunctionResult(base.ConvertRuleResult(&res))
				}
			case "email":
				if info.Value.Contact.Email == "" {
					res := model.RuleFunctionResult{
						Message:   vacuumUtils.SuppliedOrDefault(context.Rule.Message, "`contact` section must contain a `email`"),
						StartNode: info.Contact.Value.GoLow().KeyNode,
						EndNode:   vacuumUtils.BuildEndNode(info.Contact.Value.GoLow().KeyNode),
						Rule:      context.Rule,
						Path:      "$.info.contact",
					}
					results = append(results, res)
					info.Contact.AddRuleFunctionResult(base.ConvertRuleResult(&res))
				}
			}
		}
	}
	return results
}
