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

	var results = &[]model.RuleFunctionResult{}

	// check operations first.
	ops := GetOperationsFromRoot(nodes)

	// check requests.
	_, rbNode := utils.FindFirstKeyNode("requestBody", ops, 0)
	if rbNode != nil {
		results = checkExamples(rbNode, results)
	}

	// check responses
	_, respNode := utils.FindFirstKeyNode("responses", ops, 0)

	// for each response code, check examples.
	if respNode != nil {
		for x, respCodeNode := range respNode.Content {
			if x%2 == 0 {
				continue
			}
			results = checkExamples(respCodeNode, results)
		}
	}

	// TODO: check params and components.

	return *results

}

func checkExamples(rbNode *yaml.Node, results *[]model.RuleFunctionResult) *[]model.RuleFunctionResult {
	// don't bother if we can't see anything.
	if rbNode == nil {
		return results
	}
	for n, rbNodeChild := range rbNode.Content {
		if rbNodeChild.Value != "content" {
			continue
		} else {
			mediaTypeNode := rbNode.Content[n+1]

			results = analyzeExample(mediaTypeNode, n, results)
		}
	}
	return results
}

func analyzeExample(nameNode *yaml.Node, n int, results *[]model.RuleFunctionResult) *[]model.RuleFunctionResult {
	var nameNodeValue string
	for y, nameValueNode := range nameNode.Content {
		if nameValueNode.Value == "" {
			continue
		}
		if n+1 >= len(nameNode.Content) {
			continue
		}
		nameNodeValue = nameValueNode.Value
		_, sValue := utils.FindFirstKeyNode("schema", []*yaml.Node{nameNode.Content[y+1]}, 0)
		_, esValue := utils.FindFirstKeyNode("examples", []*yaml.Node{nameNode.Content[y+1]}, 0)
		_, eValue := utils.FindFirstKeyNode("example", []*yaml.Node{nameNode.Content[y+1]}, 0)

		// if there are no examples, or an example next to a schema, then add a result.
		if sValue != nil && (esValue == nil && eValue == nil) {
			res := model.BuildFunctionResultString(fmt.Sprintf("schema for '%s' does not "+
				"contain a sibling 'example' or 'examples', examples are *super* important", nameNodeValue))

			res.StartNode = nameNode
			res.EndNode = sValue
			*results = append(*results, res)
			continue
		}

		// extract the schema
		schema, _ := parser.ConvertNodeDefinitionIntoSchema(sValue)

		// look through multiple examples and evaluate them.
		var exampleName string

		if esValue != nil {
			for v, multiExampleNode := range esValue.Content {
				if v%2 == 0 {
					exampleName = multiExampleNode.Value
					continue
				}

				// check if the node is a map (object)
				if !utils.IsNodeMap(multiExampleNode) {
					res := model.BuildFunctionResultString(fmt.Sprintf("example '%s' must be an object, not a %v",
						exampleName, utils.MakeTagReadable(multiExampleNode)))
					res.StartNode = esValue
					res.EndNode = multiExampleNode
					*results = append(*results, res)
					continue
				}

				// check if the example validates against the schema
				res, _ := parser.ValidateNodeAgainstSchema(schema, multiExampleNode)
				if !res.Valid() {
					// extract all validation errors.
					for _, resError := range res.Errors() {

						z := model.BuildFunctionResultString(fmt.Sprintf("example '%s' is not valid: '%s' on field '%s'",
							exampleName, resError.Description(), resError.Field()))
						z.StartNode = esValue
						z.EndNode = multiExampleNode
						*results = append(*results, z)
					}
				}

				// check if the example contains a summary
				_, summaryNode := utils.FindFirstKeyNode("summary", []*yaml.Node{multiExampleNode}, 0)
				if summaryNode == nil {
					z := model.BuildFunctionResultString(fmt.Sprintf("example '%s' missing a 'summary', "+
						"examples need explaining", exampleName))
					z.StartNode = esValue
					z.EndNode = multiExampleNode
					*results = append(*results, z)
				}
			}
		}

		// handle single examples when a schema is used.
		if sValue != nil && eValue != nil {
			// there should be two nodes, the second one should be a map, not a value.
			if len(eValue.Content) > 0 {

				// ok, so let's check the object is valid against the schema.
				res, _ := parser.ValidateNodeAgainstSchema(schema, eValue)
				fmt.Print(res)

				// extract all validation errors.
				for _, resError := range res.Errors() {

					z := model.BuildFunctionResultString(fmt.Sprintf("example for '%s' is not valid: '%s' on field '%s'",
						nameNodeValue, resError.Description(), resError.Field()))
					z.StartNode = eValue
					z.EndNode = eValue.Content[len(eValue.Content)-1]
					*results = append(*results, z)
				}

			} else {

				// no good, so let's report it.
				nodeVal := "unknown"
				if len(eValue.Content) == 0 {
					nodeVal = eValue.Value
				}

				z := model.BuildFunctionResultString(fmt.Sprintf("example for media type '%s' "+
					"is malformed, should be object, not '%s'", nameNodeValue, nodeVal))
				z.StartNode = eValue
				z.EndNode = eValue
				*results = append(*results, z)
			}

		}
	}
	return results
}
