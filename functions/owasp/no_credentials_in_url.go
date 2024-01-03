// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/base"
	"gopkg.in/yaml.v3"
	"regexp"
)

type NoCredentialsInUrl struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the DefineError rule.
func (c NoCredentialsInUrl) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "no_credentials_in_url"}
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
						fmt.Sprintf("URL parameters must not contain credentials, passwords, or secrets (`%s`)",
							param.Value.Name)),
					StartNode: node,
					EndNode:   node,
					Path:      fmt.Sprintf("%s.%s", param.GenerateJSONPath(), "name"),
					Rule:      context.Rule,
				}
				param.AddRuleFunctionResult(base.ConvertRuleResult(&result))
				results = append(results, result)
			}
		}
	}
	return results
}
