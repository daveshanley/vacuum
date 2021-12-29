// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"gopkg.in/yaml.v3"
)

// UniqueOperationId is a rule that will check if each operation provides an operationId, as well as making sure
// that all the operationId's in the spec are unique.
type UniqueOperationId struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the UniqueOperationId rule.
func (oId UniqueOperationId) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "unique_operation_id",
	}
}

// RunRule will execute the UniqueOperationId rule, based on supplied context and a supplied []*yaml.Node slice.
func (oId UniqueOperationId) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	for _, node := range nodes {
		seenIds := make(map[string]bool)
		for x, pn := range node.Content {
			var currentPath string
			var currentVerb string
			if pn.Tag == "!!str" {

				currentPath = pn.Value
				verbNode := node.Content[x+1]

				for y, verbMapNode := range verbNode.Content {

					if verbMapNode.Tag == "!!str" {
						currentVerb = verbMapNode.Value
					} else {
						continue
					}

					verbDataNode := verbNode.Content[y+1]

					_, opIdValueNode := utils.FindFirstKeyNode("operationId", verbDataNode.Content)

					if opIdValueNode == nil {
						results = append(results, model.RuleFunctionResult{
							Message: fmt.Sprintf("the '%s' operation at path '%s' does not contain an operationId",
								currentVerb, currentPath),
						})
					} else {
						if seenIds[opIdValueNode.Value] {
							results = append(results, model.RuleFunctionResult{
								Message: fmt.Sprintf("the '%s' operation at path '%s' contains a "+
									"duplicate operationId '%s'", currentVerb, currentPath, opIdValueNode.Value),
							})
						} else {
							seenIds[opIdValueNode.Value] = true
						}
					}
				}
			}
		}
	}
	return results
}
