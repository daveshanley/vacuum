// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

// OperationSingleTag checks that each operation only has a single tag.
type OperationSingleTag struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the OperationSingleTag rule.
func (ost OperationSingleTag) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "operation_single_tag",
	}
}

// RunRule will execute the OperationSingleTag rule, based on supplied context and a supplied []*yaml.Node slice.
func (ost OperationSingleTag) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	paths := context.Index.GetAllPaths()
	operationTags := context.Index.GetOperationTags()

	for path, methodMap := range paths {

		for method, methodNode := range methodMap {

			tags := operationTags[path][method]

			if len(tags) > 1 {

				tagsNode, _ := utils.FindKeyNode("tags", methodNode.Node.Content)
				lastNode := utils.FindLastChildNodeWithLevel(tagsNode, 0)

				results = append(results, model.RuleFunctionResult{
					Message: fmt.Sprintf("the `%s` operation at path `%s` contains more "+
						"than one tag (%d is too many)'", method, path, len(tags)),
					StartNode: tagsNode,
					EndNode:   lastNode,
					Path:      fmt.Sprintf("$.paths['%s'].%s", path, method),
					Rule:      context.Rule,
				})
			}
		}
	}

	return results
}
