// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
    "fmt"
    "github.com/daveshanley/vacuum/model"
    vacuumUtils "github.com/daveshanley/vacuum/utils"
    "github.com/pb33f/doctor/model/high/v3"
    "gopkg.in/yaml.v3"
    "strings"
)

type HostsHttps struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the DefineError rule.
func (hh HostsHttps) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "owaspHostsHttps"}
}

// GetCategory returns the category of the HostsHttps rule.
func (hh HostsHttps) GetCategory() string {
	return model.FunctionCategoryOWASP
}

// RunRule will execute the DefineError rule, based on supplied context and a supplied []*yaml.Node slice.
func (hh HostsHttps) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	for _, server := range context.DrDocument.V3Document.Servers {
		if !strings.HasPrefix(server.Value.URL, "https") {
			node := server.Value.GoLow().URL.KeyNode
			result := model.RuleFunctionResult{
				Message:   "server URLs should use TLS (https)",
				StartNode: node,
				EndNode:   vacuumUtils.BuildEndNode(node),
				Path: vacuumUtils.SuppliedOrDefault(context.Rule.Message,
					fmt.Sprintf("%s.%s", server.GenerateJSONPath(), "url")),
				Rule: context.Rule,
			}
			server.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
			results = append(results, result)
		}
	}
	return results
}
