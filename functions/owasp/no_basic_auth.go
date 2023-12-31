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

type NoBasicAuth struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the DefineError rule.
func (ba NoBasicAuth) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "no_basic_auth"}
}

// RunRule will execute the DefineError rule, based on supplied context and a supplied []*yaml.Node slice.
func (ba NoBasicAuth) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

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
					EndNode:   node,
					Path:      fmt.Sprintf("%s.%s", scheme.GenerateJSONPath(), "scheme"),
					Rule:      context.Rule,
				}
				scheme.AddRuleFunctionResult(base.ConvertRuleResult(&result))
				results = append(results, result)
			}
		}
	}
	return results
}
