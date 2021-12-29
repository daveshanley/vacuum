package openapi_functions

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"gopkg.in/yaml.v3"
)

type UniqueOperationId struct {
}

func (oId UniqueOperationId) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "unique_operation_id",
	}
}

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
