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

type JWTBestPractice struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the DefineError rule.
func (jwt JWTBestPractice) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "owaspJWTBestPractice"}
}

// GetCategory returns the category of the JWTBestPractice rule.
func (jwt JWTBestPractice) GetCategory() string {
	return model.FunctionCategoryOWASP
}

// RunRule will execute the DefineError rule, based on supplied context and a supplied []*yaml.Node slice.
func (jwt JWTBestPractice) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	if context.DrDocument.V3Document != nil && context.DrDocument.V3Document.Components != nil {
		ss := context.DrDocument.V3Document.Components.SecuritySchemes
		for schemePairs := ss.First(); schemePairs != nil; schemePairs = schemePairs.Next() {

			scheme := schemePairs.Value()
			if scheme.Value.Type == "oauth2" || strings.ToLower(scheme.Value.BearerFormat) == "jwt" {
				if !strings.Contains(scheme.Value.Description, "RFC8725") {
					node := scheme.Value.GoLow().Description.KeyNode
					if node == nil {
						node = scheme.Value.GoLow().KeyNode
					}
					result := model.RuleFunctionResult{
						Message: vacuumUtils.SuppliedOrDefault(context.Rule.Message,
							"JWTs must explicitly declare support for `RFC8725` in the description"),
						StartNode: node,
						EndNode:   vacuumUtils.BuildEndNode(node),
						Path:      fmt.Sprintf("%s.%s", scheme.GenerateJSONPath(), "description"),
						Rule:      context.Rule,
					}
					scheme.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
					results = append(results, result)
				}
			}
		}
	}
	return results
}
