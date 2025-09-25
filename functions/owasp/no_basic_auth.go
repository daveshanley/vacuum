// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"go.yaml.in/yaml/v4"
	"strings"
)

type NoBasicAuth struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the DefineError rule.
func (ba NoBasicAuth) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "owaspNoBasicAuth"}
}

// GetCategory returns the category of the NoBasicAuth rule.
func (ba NoBasicAuth) GetCategory() string {
	return model.FunctionCategoryOWASP
}

// RunRule will execute the DefineError rule, based on supplied context and a supplied []*yaml.Node slice.
func (ba NoBasicAuth) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	if context.DrDocument.V3Document != nil && context.DrDocument.V3Document.Components != nil {
		ss := context.DrDocument.V3Document.Components.SecuritySchemes
		for schemePairs := ss.First(); schemePairs != nil; schemePairs = schemePairs.Next() {
			scheme := schemePairs.Value()
			if scheme.Value.Type == "http" {
				if strings.ToLower(scheme.Value.Scheme) == "basic" || strings.ToLower(scheme.Value.Scheme) == "negotiate" {
					node := scheme.Value.GoLow().Scheme.KeyNode
					result := model.RuleFunctionResult{
						Message: vacuumUtils.SuppliedOrDefault(context.Rule.Message,
							"security scheme uses HTTP Basic Auth, which is an insecure practice"),
						StartNode: node,
						EndNode:   vacuumUtils.BuildEndNode(node),
						Path:      model.GetStringTemplates().BuildJSONPath(scheme.GenerateJSONPath(), "scheme"),
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
