// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/libopenapi/utils"
	"go.yaml.in/yaml/v4"
)

// UniqueOperationId is a rule that will check if each operation provides an operationId, as well as making sure
// that all the operationId's in the spec are unique.
type UniqueOperationId struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the UniqueOperationId rule.
func (oId UniqueOperationId) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "oasOpIdUnique",
	}
}

// GetCategory returns the category of the UniqueOperationId rule.
func (oId UniqueOperationId) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the UniqueOperationId rule, based on supplied context and a supplied []*yaml.Node slice.
func (oId UniqueOperationId) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	paths := context.Index.GetAllPaths()
	seenIds := make(map[string]bool)

	for path, methodMap := range paths {

		for method, methodNode := range methodMap {

			_, operationId := utils.FindKeyNode("operationId", methodNode.Node.Content)

			if operationId != nil {
				if seenIds[operationId.Value] {
					results = append(results, model.RuleFunctionResult{
						Message: fmt.Sprintf("the '%s' operation at path '%s' contains a "+
							"duplicate operationId '%s'", method, path, operationId.Value),
						StartNode: methodNode.Node,
						EndNode:   vacuumUtils.BuildEndNode(methodNode.Node),
						Path:      fmt.Sprintf("$.paths['%s'].%s", path, method),
						Rule:      context.Rule,
					})
				} else {
					seenIds[operationId.Value] = true
				}
			}
		}
	}
	return results
}
