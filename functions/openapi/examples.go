// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/parser"
	"github.com/daveshanley/vacuum/utils"
	"gopkg.in/yaml.v3"
)

// Examples is a rule that checks that examples are being correctly used.
type Examples struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the TagDefined rule.
func (ex Examples) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "examples",
	}
}

// RunRule will execute the Examples rule, based on supplied context and a supplied []*yaml.Node slice.
func (ex Examples) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	// check paths first.
	ops := GetOperationsFromRoot(nodes)
	_, rbNode := utils.FindFirstKeyNode("requestBody", ops, 0)

	for n, rbNodeChild := range rbNode.Content {
		if rbNodeChild.Value != "content" {
			continue
		} else {
			mediaTypeNode := rbNode.Content[n+1]

			var mediaTypeValue string
			for y, mediaType := range mediaTypeNode.Content {
				if mediaType.Value == "" {
					continue
				}
				if n+1 >= len(mediaTypeNode.Content) {
					continue
				}
				mediaTypeValue = mediaType.Value
				_, sValue := utils.FindFirstKeyNode("schema", []*yaml.Node{mediaTypeNode.Content[y+1]}, 0)
				_, esValue := utils.FindFirstKeyNode("examples", []*yaml.Node{mediaTypeNode.Content[y+1]}, 0)
				_, eValue := utils.FindFirstKeyNode("example", []*yaml.Node{mediaTypeNode.Content[y+1]}, 0)

				// if there are no examples, or an example next to a schema, then add a result.
				if sValue != nil && (esValue == nil && eValue == nil) {
					res := model.BuildFunctionResultString(fmt.Sprintf("schema for '%s' does not "+
						"contain a sibling 'example' or 'examples', examples are *super* important", mediaTypeValue))

					res.StartNode = mediaTypeNode
					res.EndNode = sValue
					results = append(results, res)
					continue
				}

				// extract the schema
				schema, _ := parser.ConvertNodeDefinitionIntoSchema(sValue)

				// look through multiple examples and evaluate them.
				var exampleName string
				for v, multiExampleNode := range esValue.Content {
					if v%2 == 0 {
						exampleName = multiExampleNode.Value
						continue
					}
					if !utils.IsNodeMap(multiExampleNode) {
						res := model.BuildFunctionResultString(fmt.Sprintf("example '%s' must be an object, not a %v",
							exampleName, utils.MakeTagReadable(multiExampleNode)))

						res.StartNode = esValue
						res.EndNode = multiExampleNode
						results = append(results, res)
						continue
					}

					res, _ := parser.ValidateNodeAgainstSchema(schema, multiExampleNode)
					if !res.Valid() {
						// extract all validation errors.
						for _, resError := range res.Errors() {

							z := model.BuildFunctionResultString(fmt.Sprintf("example '%s' is not valid: '%s'", exampleName, resError.Description()))
							z.StartNode = esValue
							z.EndNode = multiExampleNode
							results = append(results, z)
						}
					}

				}
			}
		}
	}

	return results

}
