// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/base"
	"gopkg.in/yaml.v3"
	"strings"
)

type AuthInsecureSchemes struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the DefineError rule.
func (is AuthInsecureSchemes) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "jwt_best_practice"}
}

// RunRule will execute the DefineError rule, based on supplied context and a supplied []*yaml.Node slice.
func (is AuthInsecureSchemes) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	if context.DrDocument.V3Document != nil && context.DrDocument.V3Document.Components != nil {
		ss := context.DrDocument.V3Document.Components.SecuritySchemes
		for schemePairs := ss.First(); schemePairs != nil; schemePairs = schemePairs.Next() {
			scheme := schemePairs.Value()
			if scheme.Value.Type == "http" {
				if strings.ToLower(scheme.Value.Scheme) == "negotiate" ||
					strings.ToLower(scheme.Value.Scheme) == "oauth" {
					node := scheme.Value.GoLow().Scheme.KeyNode
					result := model.RuleFunctionResult{
						Message:   utils.SuppliedOrDefault(context.Rule.Message, "authentication scheme is considered outdated or insecure"),
						StartNode: node,
						EndNode:   utils.BuildEndNode(node),
						Path:      fmt.Sprintf("%s.%s", scheme.GenerateJSONPath(), "scheme"),
						Rule:      context.Rule,
					}
					scheme.AddRuleFunctionResult(base.ConvertRuleResult(&result))
					results = append(results, result)
				}
			}
		}
	}
	return results
}
