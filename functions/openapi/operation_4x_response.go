// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
    "github.com/daveshanley/vacuum/model"
    vacuumUtils "github.com/daveshanley/vacuum/utils"
    "github.com/pb33f/doctor/model/high/v3"
    "gopkg.in/yaml.v3"
    "strconv"
)

// Operation4xResponse is a rule that checks if an operation returns a 4xx (user error) code.
type Operation4xResponse struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the SuccessResponse rule.
func (or Operation4xResponse) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "oasOpErrorResponse"}
}

// GetCategory returns the category of the SuccessResponse rule.
func (or Operation4xResponse) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the Operation4xResponse rule, based on supplied context and a supplied []*yaml.Node slice.
func (or Operation4xResponse) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	if context.DrDocument.V3Document == nil || context.DrDocument.V3Document.Paths == nil || context.DrDocument.V3Document.Paths.PathItems == nil {
		return results
	}

	for pathPairs := context.DrDocument.V3Document.Paths.PathItems.First(); pathPairs != nil; pathPairs = pathPairs.Next() {
		pathItem := pathPairs.Value()

		// extract operations
		for methodPairs := pathItem.GetOperations().First(); methodPairs != nil; methodPairs = methodPairs.Next() {
			operation := methodPairs.Value()

			// check if operation has a 4xx response
			if operation.Value.Responses != nil {
				seen := false
				for codePairs := operation.Responses.Codes.First(); codePairs != nil; codePairs = codePairs.Next() {
					code := codePairs.Key()

					// convert code to int
					codeVal, er := strconv.Atoi(code)
					if er == nil {
						if codeVal >= 400 && codeVal < 500 {
							seen = true
							break
						}
					}
				}
				if !seen {
					res := model.RuleFunctionResult{
						Message:   "operation must define at least one 4xx error response",
						StartNode: operation.Value.GoLow().Responses.KeyNode,
						EndNode:   vacuumUtils.BuildEndNode(operation.Value.GoLow().Responses.KeyNode),
						Path:      operation.Responses.GenerateJSONPath(),
						Rule:      context.Rule,
					}
					results = append(results, res)
					operation.Responses.AddRuleFunctionResult(v3.ConvertRuleResult(&res))
				}
			}
		}
	}
	return results
}
