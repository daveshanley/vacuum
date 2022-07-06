// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"gopkg.in/yaml.v3"
)

// TagDefined is a rule that checks if an operation uses a tag, it's also defined in the global tag definitions.
type TagDefined struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the TagDefined rule.
func (td TagDefined) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "tag_defined",
	}
}

// RunRule will execute the TagDefined rule, based on supplied context and a supplied []*yaml.Node slice.
func (td TagDefined) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	seenGlobalTags := make(map[string]bool)
	tagsNode := context.Index.GetGlobalTagsNode()
	pathsNode := context.Index.GetPathsNode()

	if tagsNode != nil {
		for _, tagNode := range tagsNode.Content {
			_, tag := utils.FindKeyNode("name", []*yaml.Node{tagNode})
			if tag != nil {
				seenGlobalTags[tag.Value] = true
			}
		}
	}

	if pathsNode == nil {
		return results
	}

	for x, operationNode := range pathsNode.Content {
		var currentPath string
		var currentVerb string
		if operationNode.Tag == "!!str" {
			currentPath = operationNode.Value
			verbNode := pathsNode.Content[x+1]
			for y, verbMapNode := range verbNode.Content {

				if verbMapNode.Tag == "!!str" {
					currentVerb = verbMapNode.Value
				} else {
					continue
				}

				verbDataNode := verbNode.Content[y+1]
				_, opTagsNode := utils.FindKeyNode("tags", verbDataNode.Content)

				if opTagsNode != nil {

					tagIndex := 0
					for j, operationTag := range opTagsNode.Content {
						if operationTag.Tag == "!!str" {
							if !seenGlobalTags[operationTag.Value] {
								endNode := utils.FindLastChildNode(operationTag)
								if j+1 < len(opTagsNode.Content) {
									endNode = opTagsNode.Content[j+1]
								}
								results = append(results, model.RuleFunctionResult{
									Message: fmt.Sprintf("the '%s' operation at path '%s' contains a "+
										"tag '%s', that is not defined in the global document tags",
										currentVerb, currentPath, operationTag.Value),
									StartNode: operationTag,
									EndNode:   endNode,
									Path:      fmt.Sprintf("$.paths.%s.%s.tags[%v]", currentPath, currentVerb, tagIndex),
									Rule:      context.Rule,
								})
							}
							tagIndex++

						}
					}
				}
			}
		}
	}
	return results

}
