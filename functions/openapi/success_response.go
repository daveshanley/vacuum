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

	var currentPath string
	var currentVerb string

	for _, n := range nodes {
		//for j, operationNode := range n.Content {

		_, pathNode := utils.FindKeyNode("paths", n.Content)

		for j, operationNode := range pathNode.Content {

			if utils.IsNodeStringValue(operationNode) {
				currentPath = operationNode.Value
			}
			if utils.IsNodeMap(operationNode) {

				for h, verbMapNode := range operationNode.Content {
					if utils.IsNodeStringValue(verbMapNode) && isHttpVerb(verbMapNode.Value) {
						currentVerb = verbMapNode.Value
					} else {
						continue
					}
					verbDataNode := operationNode.Content[h+1]

					fieldNode, valNode := utils.FindFirstKeyNode(context.RuleAction.Field, verbDataNode.Content)
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
							_, g := utils.FindKeyNode("operationId", operationNode.Content)
							var name string
							if g != nil {
								name = g.Value
							} else {
								name = "undefined operation (no operationId)"
							}

							endNode := operationNode
							if j+1 < len(operationNode.Content) {
								endNode = operationNode.Content[j+1]
							}

							results = append(results, model.RuleFunctionResult{
								Message:   fmt.Sprintf("Operation '%s' must define at least a single 2xx or 3xx response", name),
								StartNode: operationNode,
								EndNode:   endNode,
								Path:      fmt.Sprintf("$.paths.%s.%s.%s", currentPath, currentVerb, context.RuleAction.Field),
							})
						}
					}

				}

			}
		}

	}
	return results
}
