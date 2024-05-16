// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/base"
	"gopkg.in/yaml.v3"
	"strings"
)

type NoApiKeyInUrl struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the DefineError rule.
func (ak NoApiKeyInUrl) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "no_api_key_in_url"}
}

// GetCategory returns the category of the NoApiKeyInUrl rule.
func (ak NoApiKeyInUrl) GetCategory() string {
	return model.FunctionCategoryOWASP
}

// RunRule will execute the DefineError rule, based on supplied context and a supplied []*yaml.Node slice.
func (ak NoApiKeyInUrl) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	doc := context.DrDocument

	if doc.V3Document != nil && doc.V3Document.Components != nil {
		security := doc.V3Document.Components.SecuritySchemes
		for securityPairs := security.First(); securityPairs != nil; securityPairs = securityPairs.Next() {
			securityScheme := securityPairs.Value()
			if strings.ToLower(securityScheme.Value.Type) == "apikey" {
				if strings.ToLower(securityScheme.Value.In) == "query" || strings.ToLower(securityScheme.Value.In) == "path" {
					node := securityScheme.Value.GoLow().In.KeyNode
					result := model.RuleFunctionResult{
						Message: vacuumUtils.SuppliedOrDefault(context.Rule.Message,
							fmt.Sprintf("API keys must not be passed via URL parameters (`%s`)",
								securityScheme.Value.In)),
						StartNode: node,
						EndNode:   vacuumUtils.BuildEndNode(node),
						Path:      fmt.Sprintf("%s.%s", securityScheme.GenerateJSONPath(), "in"),
						Rule:      context.Rule,
					}
					securityScheme.AddRuleFunctionResult(base.ConvertRuleResult(&result))
					results = append(results, result)
				}
			}
		}
	}
	return results
}
