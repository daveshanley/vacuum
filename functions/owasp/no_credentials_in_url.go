// Copyright 2023 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"go.yaml.in/yaml/v4"
	"regexp"
)

type NoCredentialsInUrl struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the DefineError rule.
func (c NoCredentialsInUrl) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "owaspNoCredentialsInUrl"}
}

// GetCategory returns the category of the NoCredentialsInUrl rule.
func (c NoCredentialsInUrl) GetCategory() string {
	return model.FunctionCategoryOWASP
}

var noCredsRxp, _ = regexp.Compile(`(?i)^.*(client_?secret|token|access_?token|refresh_?token|id_?token|password|secret|api-?key).*$`)

// RunRule will execute the DefineError rule, based on supplied context and a supplied []*yaml.Node slice.
func (c NoCredentialsInUrl) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	doc := context.DrDocument
	params := doc.Parameters

	for _, param := range params {
		if param.Value.In == "query" || param.Value.In == "path" {

			if noCredsRxp.MatchString(param.Value.Name) {
				node := param.Value.GoLow().Name.KeyNode
				result := model.RuleFunctionResult{
					Message: vacuumUtils.SuppliedOrDefault(context.Rule.Message,
						model.GetStringTemplates().BuildCredentialsMessage(param.Value.Name)),
					StartNode: node,
					EndNode:   vacuumUtils.BuildEndNode(node),
					Path:      model.GetStringTemplates().BuildJSONPath(param.GenerateJSONPath(), "name"),
					Rule:      context.Rule,
				}
				param.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
				results = append(results, result)
			}
		}
	}
	return results
}
