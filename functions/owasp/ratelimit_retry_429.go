// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
    "fmt"
    "github.com/daveshanley/vacuum/model"
    vacuumUtils "github.com/daveshanley/vacuum/utils"
    "github.com/pb33f/doctor/model/high/v3"
    "gopkg.in/yaml.v3"
)

type RatelimitRetry429 struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the DefineError rule.
func (r RatelimitRetry429) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "owaspRatelimitRetryAfter"}
}

// GetCategory returns the category of the RatelimitRetry429 rule.
func (r RatelimitRetry429) GetCategory() string {
	return model.FunctionCategoryOWASP
}

// RunRule will execute the DefineError rule, based on supplied context and a supplied []*yaml.Node slice.
func (r RatelimitRetry429) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	if context.DrDocument.V3Document != nil && context.DrDocument.V3Document.Paths != nil {
		for pathPairs := context.DrDocument.V3Document.Paths.PathItems.First(); pathPairs != nil; pathPairs = pathPairs.Next() {
			for opPairs := pathPairs.Value().GetOperations().First(); opPairs != nil; opPairs = opPairs.Next() {
				opValue := opPairs.Value()
				opType := opPairs.Key()

				if opValue.Responses != nil && opValue.Responses.Codes != nil {
					responses := opValue.Responses.Codes

					for respPairs := responses.First(); respPairs != nil; respPairs = respPairs.Next() {
						resp := respPairs.Value()
						respCode := respPairs.Key()
						if respCode == "429" {

							var node *yaml.Node
							if resp.Headers != nil {
								foundHeader := resp.Headers.GetOrZero("Retry-After")
								if foundHeader == nil {
									lowCodes := opValue.Responses.Value.GoLow().Codes
									for lowCodePairs := lowCodes.First(); lowCodePairs != nil; lowCodePairs = lowCodePairs.Next() {
										lowCodeKey := lowCodePairs.Key()
										codeCodeVal := lowCodeKey.KeyNode.Value
										if codeCodeVal == "429" {
											node = lowCodeKey.KeyNode
										}
									}
									result := model.RuleFunctionResult{
										Message: vacuumUtils.SuppliedOrDefault(context.Rule.Message,
											"missing 'Retry-After' header for 429 error response"),
										StartNode: node,
										EndNode:   node,
										Path:      fmt.Sprintf("$.paths.%s.%s.responses.429", pathPairs.Key(), opType),
										Rule:      context.Rule,
									}
									resp.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
									results = append(results, result)
								}
							}
						}
					}
				}
			}
		}
	}
	return results
}
