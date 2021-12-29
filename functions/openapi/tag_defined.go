package openapi_functions

import (
	"fmt"
	"github.com/daveshanley/vaccum/model"
	"github.com/daveshanley/vaccum/utils"
	"gopkg.in/yaml.v3"
)

type TagDefined struct {
	tagNodes []*yaml.Node
	opsNodes []*yaml.Node
}

func (td TagDefined) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "tag_defined",
	}
}

func (td TagDefined) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	seenGlobalTags := make(map[string]bool)

	if td.opsNodes == nil {
		td.opsNodes = GetOperationsFromRoot(nodes)
	}
	if td.tagNodes == nil {
		td.tagNodes = GetTagsFromRoot(nodes)
	}

	for _, tagNode := range td.tagNodes {
		_, tag := utils.FindKeyNode("name", []*yaml.Node{tagNode})
		if tag != nil {
			seenGlobalTags[tag.Value] = true
		}
	}

	for x, operationNode := range td.opsNodes {
		var currentPath string
		var currentVerb string
		if operationNode.Tag == "!!str" {
			currentPath = operationNode.Value
			verbNode := td.opsNodes[x+1]
			for y, verbMapNode := range verbNode.Content {

				if verbMapNode.Tag == "!!str" {
					currentVerb = verbMapNode.Value
				} else {
					continue
				}

				verbDataNode := verbNode.Content[y+1]
				_, tagsNode := utils.FindFirstKeyNode("tags", verbDataNode.Content)

				if tagsNode != nil {

					for _, operationTag := range tagsNode.Content {
						if operationTag.Tag == "!!str" {
							if !seenGlobalTags[operationTag.Value] {
								results = append(results, model.RuleFunctionResult{
									Message: fmt.Sprintf("the '%s' operation at path '%s' contains a "+
										"tag '%s', that is not defined in the global document tags",
										currentVerb, currentPath, operationTag.Value),
								})
							}
						}
					}
				}
			}
		}
	}
	return results

}
