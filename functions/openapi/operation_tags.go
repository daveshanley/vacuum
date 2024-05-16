// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

// OperationTags is a rule that checks operations are using tags and they are not empty.
type OperationTags struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the TagDefined rule.
func (ot OperationTags) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "operation_tags",
	}
}

// GetCategory returns the category of the TagDefined rule.
func (ot OperationTags) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the OperationTags rule, based on supplied context and a supplied []*yaml.Node slice.
func (ot OperationTags) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult
	pathsNode := context.Index.GetPathsNode()

	if pathsNode == nil {
		return results
	}

	for x, operationNode := range pathsNode.Content {
		var currentPath string
		var currentVerb string
		if operationNode.Tag == "!!str" {
			currentPath = operationNode.Value

			var verbNode *yaml.Node
			if x+1 == len(pathsNode.Content) {
				verbNode = pathsNode.Content[x]
			} else {
				verbNode = pathsNode.Content[x+1]
			}
			skip := false
			for y, verbMapNode := range verbNode.Content {

				if verbMapNode.Tag == "!!str" {
					currentVerb = verbMapNode.Value
				} else {
					continue
				}
				// skip non-operations
				switch currentVerb {
				case
					// No v2.*Label here, they're duplicates
					v3.GetLabel, v3.PutLabel, v3.PostLabel, v3.DeleteLabel, v3.OptionsLabel, v3.HeadLabel, v3.PatchLabel, v3.TraceLabel:
					// Ok, an operation
				default:
					skip = true
					continue
				}
				if skip {
					skip = false
					continue
				}
				var opTagsNode *yaml.Node

				if y+1 < len(verbNode.Content) {
					verbDataNode := verbNode.Content[y+1]
					_, opTagsNode = utils.FindKeyNode("tags", verbDataNode.Content)
				} else {
					verbDataNode := verbNode.Content[y]
					_, opTagsNode = utils.FindKeyNode("tags", verbDataNode.Content)
				}

				if opTagsNode == nil || len(opTagsNode.Content) <= 0 {

					var msg string
					if opTagsNode == nil {
						msg = fmt.Sprintf("tags for `%s` operation at path `%s` are missing",
							currentVerb, currentPath)
					} else {
						msg = fmt.Sprintf("tags for `%s` operation at path `%s` are empty",
							currentVerb, currentPath)
					}

					results = append(results, model.RuleFunctionResult{
						Message:   msg,
						StartNode: verbMapNode,
						EndNode:   vacuumUtils.BuildEndNode(verbMapNode),
						Path:      fmt.Sprintf("$.paths['%s'].%s", currentPath, currentVerb),
						Rule:      context.Rule,
					})
				}
			}
		}
	}
	return results

}
