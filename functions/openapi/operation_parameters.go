package openapi_functions

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"gopkg.in/yaml.v3"
)

type OperationParameters struct {
}

func (op OperationParameters) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "operation_parameters",
	}
}

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

					_, parametersNode := utils.FindFirstKeyNode("parameters", verbDataNode.Content)

					if parametersNode != nil {

						for _, paramNode := range parametersNode.Content {
							if paramNode.Tag == "!!map" {

								// check for 'in' and 'name' nodes in operation parameters.
								_, paramInNode := utils.FindFirstKeyNode("in", paramNode.Content)
								_, paramNameNode := utils.FindFirstKeyNode("name", paramNode.Content)

								if paramInNode != nil {
									if seenParamInLocations[paramInNode.Value] {
										if paramInNode.Value == "body" {
											results = append(results, model.RuleFunctionResult{
												Message: fmt.Sprintf("the '%s' operation at path '%s' contains a "+
													"duplicate param in:body definition", currentVerb, currentPath),
											})
										}
									} else {
										if paramInNode.Value == "body" || paramInNode.Value == "formData" {
											if seenParamInLocations["formData"] || seenParamInLocations["body"] {
												results = append(results, model.RuleFunctionResult{
													Message: fmt.Sprintf("the '%s' operation at path '%s' "+
														"contains parameters using both in:body and in:formData",
														currentVerb, currentPath),
												})
											}
										}
										seenParamInLocations[paramInNode.Value] = true
									}
								} else {
									results = append(results, model.RuleFunctionResult{
										Message: fmt.Sprintf("the '%s' operation at path '%s' contains a "+
											"parameter with no 'in' value", currentVerb, currentPath),
									})

								}

								if paramNameNode != nil {
									if seenParamNames[paramNameNode.Value] {
										results = append(results, model.RuleFunctionResult{
											Message: fmt.Sprintf("the '%s' operation at path '%s' contains a "+
												"parameter with duplicate name '%s'", currentVerb, currentPath, paramNameNode.Value),
										})
									} else {
										seenParamNames[paramNameNode.Value] = true
									}
								} else {
									results = append(results, model.RuleFunctionResult{
										Message: fmt.Sprintf("the '%s' operation at path '%s' contains a "+
											"parameter with no 'name' value", currentVerb, currentPath),
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
