// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"gopkg.in/yaml.v3"
)

// OperationParameters is a rule that checks for valid parameters and parameters combinations
type OperationParameters struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the OperationParameters rule.
func (op OperationParameters) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "operation_parameters",
	}
}

// RunRule will execute the OperationParameters rule, based on supplied context and a supplied []*yaml.Node slice.
func (op OperationParameters) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	for _, node := range nodes {

		for x, pn := range node.Content {
			var currentPath string
			var currentVerb string
			if pn.Tag == "!!str" {
				currentPath = pn.Value
				verbNode := node.Content[x+1]

				for y, verbMapNode := range verbNode.Content {

					seenParamNames := make(map[string]bool)
					seenParamInLocations := make(map[string]bool)
					if verbMapNode.Tag == "!!str" {
						currentVerb = verbMapNode.Value
					} else {
						continue
					}

					verbDataNode := verbNode.Content[y+1]

					_, parametersNode := utils.FindFirstKeyNode("parameters", verbDataNode.Content, 0)

					if parametersNode != nil {

						for j, paramNode := range parametersNode.Content {
							if paramNode.Tag == "!!map" {

								startNode := paramNode
								endNode := utils.FindLastChildNode(startNode)
								if j+1 < len(parametersNode.Content) {
									endNode = parametersNode.Content[j+1]
								}

								// check for 'in' and 'name' nodes in operation parameters.
								_, paramInNode := utils.FindFirstKeyNode("in", paramNode.Content, 0)
								_, paramNameNode := utils.FindFirstKeyNode("name", paramNode.Content, 0)

								resultPath := fmt.Sprintf("$.paths.%s.%s.parameters", currentPath, currentVerb)

								if paramInNode != nil {
									if seenParamInLocations[paramInNode.Value] {
										if paramInNode.Value == "body" {
											results = append(results, model.RuleFunctionResult{
												Message: fmt.Sprintf("the '%s' operation at path '%s' contains a "+
													"duplicate param in:body definition", currentVerb, currentPath),
												StartNode: startNode,
												EndNode:   endNode,
												Path:      resultPath,
											})
										}
									} else {
										if paramInNode.Value == "body" || paramInNode.Value == "formData" {
											if seenParamInLocations["formData"] || seenParamInLocations["body"] {
												results = append(results, model.RuleFunctionResult{
													Message: fmt.Sprintf("the '%s' operation at path '%s' "+
														"contains parameters using both in:body and in:formData",
														currentVerb, currentPath),
													StartNode: startNode,
													EndNode:   endNode,
													Path:      resultPath,
												})
											}
										}
										seenParamInLocations[paramInNode.Value] = true
									}
								} else {
									results = append(results, model.RuleFunctionResult{
										Message: fmt.Sprintf("the '%s' operation at path '%s' contains a "+
											"parameter with no 'in' value", currentVerb, currentPath),
										StartNode: startNode,
										EndNode:   endNode,
										Path:      resultPath,
									})

								}

								if paramNameNode != nil {
									if seenParamNames[paramNameNode.Value] {
										results = append(results, model.RuleFunctionResult{
											Message: fmt.Sprintf("the '%s' operation at path '%s' contains a "+
												"parameter with duplicate name '%s'", currentVerb, currentPath, paramNameNode.Value),
											StartNode: startNode,
											EndNode:   endNode,
											Path:      resultPath,
										})
									} else {
										seenParamNames[paramNameNode.Value] = true
									}
								} else {
									results = append(results, model.RuleFunctionResult{
										Message: fmt.Sprintf("the '%s' operation at path '%s' contains a "+
											"parameter with no 'name' value", currentVerb, currentPath),
										StartNode: startNode,
										EndNode:   endNode,
										Path:      resultPath,
									})
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
