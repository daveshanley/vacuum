// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

// OperationId is a rule that will check if each operation provides an operationId
type OperationId struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the OperationId rule.
func (oId OperationId) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "operation_id",
	}
}

// RunRule will execute the OperationId rule, based on supplied context and a supplied []*yaml.Node slice.
func (oId OperationId) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	paths := context.Index.GetAllPaths()

	for path, methodMap := range paths {

		for method, methodNode := range methodMap {

			_, operationId := utils.FindKeyNode("operationId", methodNode.Node.Content)
			lastNode := utils.FindLastChildNodeWithLevel(methodNode.Node, 0)

			if operationId == nil {
				results = append(results, model.RuleFunctionResult{
					Message: fmt.Sprintf("the '%s' operation at path '%s' does not contain an operationId",
						method, path),
					StartNode: methodNode.Node,
					EndNode:   lastNode,
					Path:      fmt.Sprintf("$.paths.%s.%s", path, method),
					Rule:      context.Rule,
				})
			}
		}
	}
	return results
}
