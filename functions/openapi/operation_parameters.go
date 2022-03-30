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

	// add any param indexing errors already found.
	for _, paramIndexError := range context.Index.GetOperationParametersIndexErrors() {
		results = append(results, model.RuleFunctionResult{
			Message:   paramIndexError.Error.Error(),
			StartNode: paramIndexError.Node,
			EndNode:   paramIndexError.Node,
			Path:      paramIndexError.Path,
			Rule:      context.Rule,
		})
	}

	// look in the index for all operations params.
	for path, methods := range context.Index.GetOperationParameterReferences() {
		for method, methodNode := range methods {

			seenParamInLocations := make(map[string]bool)

			for _, param := range methodNode {

				_, paramInNode := utils.FindKeyNode("in", param.Node.Content)

				currentVerb := method
				currentPath := path

				startNode := param.Node
				endNode := utils.FindLastChildNode(startNode)

				resultPath := fmt.Sprintf("$.paths.%s.%s.parameters", path, currentVerb)

				if paramInNode != nil {
					if seenParamInLocations[paramInNode.Value] {
						if paramInNode.Value == "body" {
							results = append(results, model.RuleFunctionResult{
								Message: fmt.Sprintf("the '%s' operation at path '%s' contains a "+
									"duplicate param in:body definition", currentVerb, currentPath),
								StartNode: startNode,
								EndNode:   endNode,
								Path:      resultPath,
								Rule:      context.Rule,
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
									Rule:      context.Rule,
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
						Rule:      context.Rule,
					})

				}
			}
		}
	}

	return results
}
