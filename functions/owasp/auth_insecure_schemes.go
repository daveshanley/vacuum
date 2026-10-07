// Copyright 2023 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/go-yaml"
	libutils "github.com/pb33f/libopenapi/utils"
	"strings"
)

// AuthInsecureSchemes checks legacy HTTP authentication and selected OAuth flows.
type AuthInsecureSchemes struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the AuthInsecureSchemes rule.
func (is AuthInsecureSchemes) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name:       "owaspAuthInsecureSchemes",
		Properties: []model.RuleFunctionProperty{{Name: "flow", Description: "OAuth flow to reject: password or implicit. Omit to check HTTP authentication schemes."}},
	}
}

// GetCategory returns the category of the AuthInsecureSchemes rule.
func (is AuthInsecureSchemes) GetCategory() string {
	return model.FunctionCategoryOWASP
}

// RunRule reports legacy HTTP authentication or the OAuth flow selected by the rule.
func (is AuthInsecureSchemes) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult
	flowName, _ := libutils.ExtractValueFromInterfaceMap("flow", context.Options).(string)

	if context.DrDocument == nil {
		return results
	}

	if context.DrDocument.V3Document != nil && context.DrDocument.V3Document.Components != nil {
		ss := context.DrDocument.V3Document.Components.SecuritySchemes
		for schemePairs := ss.First(); schemePairs != nil; schemePairs = schemePairs.Next() {
			scheme := schemePairs.Value()
			if flowName != "" {
				if scheme.Value.Type != "oauth2" || scheme.Flows == nil {
					continue
				}
				var flow *v3.OAuthFlow
				switch flowName {
				case "password":
					flow = scheme.Flows.Password
				case "implicit":
					flow = scheme.Flows.Implicit
				}
				if flow == nil {
					continue
				}
				node := flow.Value.GoLow().RootNode
				result := model.RuleFunctionResult{
					Message:   utils.SuppliedOrDefault(context.Rule.Message, fmt.Sprintf("OAuth %s flow should not be used; use authorization code with PKCE for user authorization", flowName)),
					StartNode: node, EndNode: utils.BuildEndNode(node), Path: flow.GenerateJSONPath(), Rule: context.Rule,
				}
				flow.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
				results = append(results, result)
				continue
			}
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
					scheme.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
					results = append(results, result)
				}
			}
		}
	}
	return results
}
