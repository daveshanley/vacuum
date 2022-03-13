// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/parser"
	"github.com/daveshanley/vacuum/utils"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"github.com/xeipuuv/gojsonschema"
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

	var opPath, opMethod string
	for i, op := range ops {
		if i%2 == 0 {
			opPath = op.Value
			continue
		}

		for m, method := range op.Content {

			if m%2 == 0 {
				opMethod = method.Value
				continue
			}

			basePath := fmt.Sprintf("$.paths.%s.%s", opPath, opMethod)

			// check requests.
			_, rbNode := utils.FindKeyNode("requestBody", method.Content)
			if rbNode != nil {
				results = checkExamples(rbNode, utils.BuildPath(basePath, []string{"requestBody"}), results, context)
			}

			// check parameters.
			_, paramsNode := utils.FindFirstKeyNode("parameters", method.Content, 0)
			if paramsNode != nil && utils.IsNodeArray(paramsNode) {

				for y, param := range paramsNode.Content {

					// extract name from param
					_, nameNode := utils.FindFirstKeyNode("name", []*yaml.Node{param}, 0)
					if nameNode != nil {
						results = analyzeExample(nameNode.Value, param,
							utils.BuildPath(basePath, []string{fmt.Sprintf("%s[%d]", "parameters", y)}), results, context)
					}
				}
			}

			// check responses
			_, respNode := utils.FindFirstKeyNode("responses", method.Content, 0)

			// for each response code, check examples.
			if respNode != nil {
				var code string
				for x, respCodeNode := range respNode.Content {
					if x%2 == 0 {
						code = respCodeNode.Value
						continue
					}
					results = checkExamples(respCodeNode, utils.BuildPath(basePath, []string{fmt.Sprintf("%s.%s",
						"responses", code)}), results, context)
				}
			}
		}
	}

	// check components.
	componentsPathString := "$.components.schemas"
	path, _ := yamlpath.NewPath(componentsPathString)
	objNode, _ := path.Find(nodes[0])

	results = checkAllDefinitionsForExamples(objNode, results, componentsPathString, context)

	// check definitions (swagger)
	defPathString := "$.definitions"
	path, _ = yamlpath.NewPath(defPathString)
	objNode, _ = path.Find(nodes[0])

	results = checkAllDefinitionsForExamples(objNode, results, defPathString, context)

	// check parameters
	componentParamPath := "$.components.parameters"
	path, _ = yamlpath.NewPath(componentParamPath)
	paramsNode, _ := path.Find(nodes[0])

	// check parameters.
	if paramsNode != nil && len(paramsNode) == 1 && utils.IsNodeArray(paramsNode[0]) {

		for x, param := range paramsNode {

			// extract name from param
			_, nameNode := utils.FindFirstKeyNode("name", []*yaml.Node{param}, 0)
			if nameNode != nil {

				results = analyzeExample(nameNode.Value, param,
					utils.BuildPath(componentParamPath, []string{fmt.Sprintf("%s[%d]", "parameters", x)}), results, context)
			}
		}
	}

	return *results
}

func checkAllDefinitionsForExamples(objNode []*yaml.Node,
	results *[]model.RuleFunctionResult, path string, context model.RuleFunctionContext) *[]model.RuleFunctionResult {
	if len(objNode) > 0 {
		if objNode[0] != nil {
			compNode := objNode[0]
			var compName string
			for n, schemaNode := range compNode.Content {
				if n%2 == 0 {
					compName = schemaNode.Value
					continue
				}
				results = checkDefinitionForExample(schemaNode, compName, results, path, context)
			}
		}
	}
	return results
}

func checkDefinitionForExample(componentNode *yaml.Node, compName string,
	results *[]model.RuleFunctionResult, path string, context model.RuleFunctionContext) *[]model.RuleFunctionResult {

	// extract properties and a top level example, if it exists.
	topExKey, topExValue := utils.FindKeyNode("example", []*yaml.Node{componentNode})
	_, pValue := utils.FindKeyNode("properties", []*yaml.Node{componentNode})
	var pName string

	// if no object level example exists, check for property examples.
	if topExKey == nil && topExValue == nil && pValue != nil {
		for n, prop := range pValue.Content {
			if n%2 == 0 {
				pName = prop.Value
				continue
			}
			if utils.IsNodeMap(prop) {

				// check for an example
				exKey, exValue := utils.FindKeyNode("example", prop.Content)
				if exKey == nil && exValue == nil {

					res := model.BuildFunctionResultString(fmt.Sprintf("Missing example for '%s' on component '%s'",
						pName, compName))

					res.StartNode = prop
					res.EndNode = prop.Content[len(prop.Content)-1]
					res.Path = utils.BuildPath(path, []string{compName, pName})
					res.Rule = context.Rule
					*results = append(*results, res)
					continue

				} else {

					// so there is an example, lets validate it.
					schema, _ := parser.ConvertNodeDefinitionIntoSchema(prop)
					var res *gojsonschema.Result
					if schema != nil && *schema.Type == "array" {
						res, _ = parser.ValidateNodeAgainstSchema(schema, exValue, true)
					}
					if schema != nil && *schema.Type != "array" {
						res, _ = parser.ValidateNodeAgainstSchema(schema, exValue, false)
					}

					// extract all validation errors.
					for _, resError := range res.Errors() {

						z := model.BuildFunctionResultString(fmt.Sprintf("Example for property '%s' is not valid: '%s'. "+
							"Value '%s' is not compatible",
							pName, resError.Description(), resError.Value()))
						z.StartNode = exKey
						z.EndNode = exValue
						z.Rule = context.Rule
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
		res, _ := parser.ValidateNodeAgainstSchema(schema, topExValue, false)

		// extract all validation errors.
		for _, resError := range res.Errors() {

			z := model.BuildFunctionResultString(fmt.Sprintf("Example for component '%s' is not valid: '%s'. "+
				"Value '%s' is not compatible", compName, resError.Description(), resError.Value()))
			z.StartNode = topExKey
			z.EndNode = topExValue.Content[len(topExValue.Content)-1]
			z.Rule = context.Rule
			*results = append(*results, z)
		}

	}
	return results
}

func checkExamples(rbNode *yaml.Node, basePath string, results *[]model.RuleFunctionResult, context model.RuleFunctionContext) *[]model.RuleFunctionResult {
	// don't bother if we can't see anything.
	if rbNode == nil {
		return results
	}
	for n, rbNodeChild := range rbNode.Content {
		if rbNodeChild.Value != "content" {
			continue
		} else {
			mediaTypeNode := rbNode.Content[n+1]
			var nameNodeValue string
			for b, nameValueNode := range mediaTypeNode.Content {
				if nameValueNode.Value == "" {
					continue
				}
				if b+1 >= len(mediaTypeNode.Content) {
					continue
				}
				nameNodeValue = nameValueNode.Value
				results = analyzeExample(nameNodeValue, mediaTypeNode.Content[b+1], basePath, results, context)
			}
		}
	}
	return results
}

func analyzeExample(nameNodeValue string, nameNode *yaml.Node, basePath string, results *[]model.RuleFunctionResult, context model.RuleFunctionContext) *[]model.RuleFunctionResult {

	_, sValue := utils.FindKeyNode("schema", nameNode.Content)
	_, esValue := utils.FindKeyNode("examples", nameNode.Content)
	_, eValue := utils.FindKeyNode("example", nameNode.Content)

	_, eInternalValue := utils.FindFirstKeyNode("example", nameNode.Content, 0)

	// if there are no examples, anywhere then add a result.
	if sValue != nil && (esValue == nil && eValue == nil && eInternalValue == nil) {
		res := model.BuildFunctionResultString(fmt.Sprintf("Schema for '%s' does not "+
			"contain any examples or example data", nameNodeValue))

		res.StartNode = nameNode
		res.EndNode = sValue
		res.Path = basePath
		res.Rule = context.Rule
		*results = append(*results, res)

	}

	var schema *parser.Schema

	// look through multiple examples and evaluate them.
	var exampleName string

	if esValue != nil {

		for v, multiExampleNode := range esValue.Content {
			if v%2 == 0 {
				exampleName = multiExampleNode.Value
				continue
			}

			nodePath := utils.BuildPath(basePath, []string{"content", nameNodeValue, "schema", "examples", exampleName, "value"})

			_, valueNode := utils.FindFirstKeyNode("value", []*yaml.Node{multiExampleNode}, 0)
			//_, externalValueNode := utils.FindFirstKeyNode("externalValue", nameNode.Content, 0)

			if valueNode != nil {
				// check if the example validates against the schema

				// extract the schema
				schema, _ = parser.ConvertNodeDefinitionIntoSchema(sValue)

				res, _ := parser.ValidateNodeAgainstSchema(schema, valueNode, false)
				if !res.Valid() {
					// extract all validation errors.
					for _, resError := range res.Errors() {

						z := model.BuildFunctionResultString(fmt.Sprintf("Example '%s' is not valid: '%s'",
							exampleName, resError.Description()))
						z.StartNode = esValue
						z.EndNode = valueNode
						z.Path = nodePath
						z.Rule = context.Rule
						*results = append(*results, z)
					}
				}

				// check if the example contains a summary
				_, summaryNode := utils.FindFirstKeyNode("summary", []*yaml.Node{valueNode}, 0)
				if summaryNode == nil {
					z := model.BuildFunctionResultString(fmt.Sprintf("Example '%s' missing a 'summary', "+
						"examples need explaining", exampleName))
					z.StartNode = esValue
					z.EndNode = valueNode
					z.Path = nodePath
					z.Rule = context.Rule
					*results = append(*results, z)
				}
			} else {
				// no value on example,
				nodePath = utils.BuildPath(basePath, []string{"content", nameNodeValue, "schema", "examples", exampleName})
				z := model.BuildFunctionResultString(fmt.Sprintf("Example '%s' has no value, it's malformed", exampleName))
				z.StartNode = esValue
				z.EndNode = esValue
				z.Path = nodePath
				z.Rule = context.Rule
				*results = append(*results, z)

			}

		}
	}

	// handle single examples when a schema is used.
	if sValue != nil && eValue != nil {
		// there should be two nodes, the second one should be a map, not a value.
		if len(eValue.Content) > 0 {

			// ok, so let's check the object is valid against the schema.
			// extract the schema
			if schema == nil {
				schema, _ = parser.ConvertNodeDefinitionIntoSchema(sValue)
			}
			res, _ := parser.ValidateNodeAgainstSchema(schema, eValue, false)

			// extract all validation errors.
			for _, resError := range res.Errors() {

				z := model.BuildFunctionResultString(fmt.Sprintf("Example for '%s' is not valid: '%s' on field '%s'",
					nameNodeValue, resError.Description(), resError.Field()))
				z.StartNode = eValue
				z.EndNode = eValue.Content[len(eValue.Content)-1]
				z.Rule = context.Rule
				*results = append(*results, z)
			}

		} else {

			// no good, so let's report it.
			nodeVal := "unknown"
			if len(eValue.Content) == 0 {
				nodeVal = eValue.Value
			}

			z := model.BuildFunctionResultString(fmt.Sprintf("Example for media type '%s' "+
				"is malformed, should be object, not '%s'", nameNodeValue, nodeVal))
			z.StartNode = eValue
			z.EndNode = eValue
			z.Rule = context.Rule
			*results = append(*results, z)
		}

	}

	// check if there are any example fields set, if so, validate schema.
	ex := 0
	if sValue != nil {
		p, _ := yamlpath.NewPath("$..example")
		z, _ := p.Find(sValue)
		ex = len(z)
	}
	if ex > 0 {
		if schema == nil {
			schema, _ = parser.ConvertNodeDefinitionIntoSchema(sValue)
		}
		exampleValidation := parser.ValidateExample(schema)
		if len(exampleValidation) > 0 {
			_, pNode := utils.FindKeyNode("properties", sValue.Content)
			var endNode *yaml.Node
			if pNode != nil && len(pNode.Content) > 0 {
				endNode = pNode.Content[len(pNode.Content)-1]
			}
			for _, example := range exampleValidation {
				z := model.BuildFunctionResultString(example.Message)
				z.StartNode = pNode
				z.EndNode = endNode
				z.Rule = context.Rule
				*results = append(*results, z)
			}
		}
	}

	return results
}
