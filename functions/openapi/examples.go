// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/parser"
	"github.com/daveshanley/vacuum/utils"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
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

	// check components.
	path, _ := yamlpath.NewPath("$.components.schemas")
	compNodes, _ := path.Find(nodes[0]) // root node.

	compNode := compNodes[0] // can only be a single components' node in a spec.
	if compNode != nil {
		var compName string
		for n, schemaNode := range compNode.Content {
			if n%2 == 0 {
				compName = schemaNode.Value
				continue
			}

			results = checkComponentForExample(schemaNode, compName, results)

			//schema, _ := parser.ConvertNodeDefinitionIntoSchema(schemaNode)
			//
			//// check schema for example.
			//noExample := true
			//
			//fmt.Print(schema)
			//fmt.Print(compName)
			//
			//rd, _ := yaml.Marshal(schemaNode)
			//
			//fmt.Print(rd)

		}
		fmt.Print(compName)
		//results = checkExamples(compNode, results)

		// TODO: check rules, don't run doubles... also handle parameters.
	}

	return *results

}

func checkComponentForExample(componentNode *yaml.Node, compName string, results *[]model.RuleFunctionResult) *[]model.RuleFunctionResult {

	// extract properties and a top level example, if it exists.
	topExKey, topExValue := utils.FindKeyNode("example", []*yaml.Node{componentNode})
	pkey, pValue := utils.FindKeyNode("properties", []*yaml.Node{componentNode})
	var pName string

	// if no object level example exists, check for property examples.
	if topExKey == nil && topExValue == nil {
		for n, prop := range pValue.Content {
			if n%2 == 0 {
				pName = prop.Value
				continue
			}
			if utils.IsNodeMap(prop) {

				// check for an example
				exKey, exValue := utils.FindFirstKeyNode("example", prop.Content, 0)
				if exKey == nil && exValue == nil {

					res := model.BuildFunctionResultString(fmt.Sprintf("missing example for '%s' on component '%s'",
						pName, compName))

					res.StartNode = prop
					res.EndNode = prop.Content[len(prop.Content)-1]
					*results = append(*results, res)
					continue

				} else {

					// so there is an example, lets validate it.
					schema, _ := parser.ConvertNodeDefinitionIntoSchema(prop)
					res, _ := parser.ValidateNodeAgainstSchema(schema, exValue)

					// extract all validation errors.
					for _, resError := range res.Errors() {

						z := model.BuildFunctionResultString(fmt.Sprintf("example for property '%s' is not valid: '%s'. "+
							"Value '%s' is not compatible",
							pName, resError.Description(), resError.Value()))
						z.StartNode = exKey
						z.EndNode = exValue
						*results = append(*results, z)
					}

				}

				continue
			}

		}
	} else {
		// we have an object level example here, so let's convert our properties into a schema and validate
		// this example against the schema.
		schema, _ := parser.ConvertNodeDefinitionIntoSchema(componentNode)
		res, _ := parser.ValidateNodeAgainstSchema(schema, topExValue)

		// extract all validation errors.
		for _, resError := range res.Errors() {

			z := model.BuildFunctionResultString(fmt.Sprintf("example for component '%s' is not valid: '%s'. "+
				"Value '%s' is not compatible", compName, resError.Description(), resError.Value()))
			z.StartNode = topExKey
			z.EndNode = topExValue.Content[len(topExValue.Content)-1]
			*results = append(*results, z)
		}

	}
	fmt.Print(pkey)
	return results
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
