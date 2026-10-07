// Copyright 2023 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/go-yaml"
	"strings"
)

// JWTBestPractice checks whether explicitly declared JWTs document RFC8725.
type JWTBestPractice struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the JWTBestPractice rule.
func (jwt JWTBestPractice) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "owaspJWTBestPractice"}
}

// GetCategory returns the category of the JWTBestPractice rule.
func (jwt JWTBestPractice) GetCategory() string {
	return model.FunctionCategoryOWASP
}

// RunRule checks JWT documentation without inferring token format from OAuth.
func (jwt JWTBestPractice) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	if context.DrDocument.V3Document != nil && context.DrDocument.V3Document.Components != nil {
		ss := context.DrDocument.V3Document.Components.SecuritySchemes
		for schemePairs := ss.First(); schemePairs != nil; schemePairs = schemePairs.Next() {

			scheme := schemePairs.Value()
			if strings.EqualFold(scheme.Value.BearerFormat, "jwt") {
				if !strings.Contains(scheme.Value.Description, "RFC8725") {
					node := scheme.Value.GoLow().Description.KeyNode
					if node == nil {
						node = scheme.Value.GoLow().KeyNode
					}
					result := model.RuleFunctionResult{
						Message: vacuumUtils.SuppliedOrDefault(context.Rule.Message,
							"JWT descriptions should reference `RFC8725`; this documents intent, not runtime validation"),
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
