// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"gopkg.in/yaml.v3"
	"strconv"
)

// SuccessResponse is a rule that checks if an operation returns a code >= 200 and <= 400
type SuccessResponse struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the SuccessResponse rule.
func (sr SuccessResponse) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "success_response"}
}

// RunRule will execute the SuccessResponse rule, based on supplied context and a supplied []*yaml.Node slice.
func (sr SuccessResponse) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult
	for _, node := range nodes {

		fieldNode, valNode := utils.FindFirstKeyNode(context.RuleAction.Field, node.Content)
		if fieldNode != nil && valNode != nil {
			var responseSeen bool
			for _, response := range valNode.Content {
				if response.Tag == "!!str" {
					responseCode, _ := strconv.Atoi(response.Value)
					if responseCode >= 200 && responseCode <= 400 {
						responseSeen = true
					}
				}
			}
			if !responseSeen {

				// see if we can extract a name from the operationId
				_, g := utils.FindKeyNode("operationId", node.Content)
				var name string
				if g != nil {
					name = g.Value
				} else {
					name = "undefined operation (no operationId)"
				}

				results = append(results, model.RuleFunctionResult{
					Message: fmt.Sprintf("Operation '%s' must define at least a single 2xx or 3xx response", name),
				})
			}
		}

	}
	return results
}
