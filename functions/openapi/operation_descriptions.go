// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"strconv"
	"strings"
)

// OperationDescription will check if an operation has a description, and if the description is useful
type OperationDescription struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the OperationDescription rule.
func (od OperationDescription) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "operation_description"}
}

// RunRule will execute the OperationDescription rule, based on supplied context and a supplied []*yaml.Node slice.
func (od OperationDescription) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	// check supplied type
	props := utils.ConvertInterfaceIntoStringMap(context.Options)

	minWordsString := props["minWords"]
	minWords, _ := strconv.Atoi(minWordsString)

	if context.Index.GetPathsNode() == nil {
		return results
	}
	ops := context.Index.GetPathsNode().Content

	var opPath, opMethod string
	for i, op := range ops {
		if i%2 == 0 {
			opPath = op.Value
			continue
		}

		skip := false
		for m, method := range op.Content {

			if m%2 == 0 {
				opMethod = method.Value
				continue
			}
			if strings.Contains(strings.ToLower(opMethod), "x-") {
				skip = true
				continue
			}
			if skip {
				skip = false
				continue
			}

			basePath := fmt.Sprintf("$.paths.%s.%s", opPath, opMethod)
			descKey, descNode := utils.FindKeyNode("description", method.Content)
			requestBodyKey, requestBodyNode := utils.FindKeyNode("requestBody", method.Content)
			_, responsesNode := utils.FindKeyNode("responses", method.Content)

			if descNode == nil {

				res := createDescriptionResult(fmt.Sprintf("Operation `%s` at path `%s` is missing a description",
					opMethod, opPath), basePath, method, method)
				res.Rule = context.Rule
				results = append(results, res)
			} else {

				// check if description is above a certain length of words
				words := strings.Split(descNode.Value, " ")
				if len(words) < minWords {

					res := createDescriptionResult(fmt.Sprintf("Operation `%s` description at path `%s` must be "+
						"at least %d words long, (%d is not enough)", opMethod, opPath, minWords, len(words)), basePath, descKey, descNode)
					res.Rule = context.Rule
					results = append(results, res)
				}
			}
			// check operation request body
			if requestBodyNode != nil {

				descKey, descNode = utils.FindKeyNode("description", requestBodyNode.Content)

				if descNode == nil {
					res := createDescriptionResult(fmt.Sprintf("Operation `requestBody` for method `%s` at path `%s` "+
						"is missing a description", opMethod, opPath),
						utils.BuildPath(basePath, []string{"requestBody"}), requestBodyKey, requestBodyNode)
					res.Rule = context.Rule
					results = append(results, res)
				} else {

					// check if request body description is above a certain length of words
					words := strings.Split(descNode.Value, " ")
					if len(words) < minWords {

						res := createDescriptionResult(fmt.Sprintf("Operation `requestBody` for method `%s` description "+
							"at path `%s` must be at least %d words long, (%d is not enough)", opMethod, opPath,
							minWords, len(words)), basePath, descKey, descNode)
						res.Rule = context.Rule
						results = append(results, res)
					}
				}
			}

			// check operation responses
			if responsesNode != nil {

				// run through each response.
				var opCode string
				var opCodeNode *yaml.Node
				for z, response := range responsesNode.Content {
					if z%2 == 0 {
						opCode = response.Value
						opCodeNode = response
						continue
					}

					descKey, descNode = utils.FindKeyNode("description", response.Content)

					if descNode == nil {
						res := createDescriptionResult(fmt.Sprintf("Operation `%s` response `%s` "+
							"at path `%s` is missing a description", opMethod, opCode, opPath),
							utils.BuildPath(basePath, []string{"requestBody"}), opCodeNode, response)
						res.Rule = context.Rule
						results = append(results, res)
					} else {

						// check if response description is above a certain length of words
						words := strings.Split(descNode.Value, " ")
						if len(words) < minWords {

							res := createDescriptionResult(fmt.Sprintf("Operation `%s` response `%s` "+
								"description at path `%s` must be at least %d words long, (%d is not enough)", opMethod, opCode, opPath,
								minWords, len(words)), basePath, descKey, descNode)
							res.Rule = context.Rule
							results = append(results, res)
						}
					}
				}
			}
		}
	}
	return results
}
