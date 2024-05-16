// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

// FormDataConsumeCheck will check enum values match the types provided
type FormDataConsumeCheck struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the FormDataConsumeCheck rule.
func (fd FormDataConsumeCheck) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "formData_consume_check",
	}
}

// GetCategory returns the category of the FormDataConsumeCheck rule.
func (fd FormDataConsumeCheck) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the FormDataConsumeCheck rule, based on supplied context and a supplied []*yaml.Node slice.
func (fd FormDataConsumeCheck) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	paths := context.Index.GetAllPaths()
	opParams := context.Index.GetAllParametersFromOperations()

	for path, methodMap := range paths {
		var topParams map[string][]*index.Reference

		// check for top params
		if opParams[path]["top"] != nil {
			topParams = opParams[path]["top"]
		}

		for method, node := range methodMap {

			// extract consumes value
			_, consumesNode := utils.FindKeyNode("consumes", node.Node.Content)

			// does this operation contain params?
			if opParams[path][method] != nil {
				paramMap := opParams[path][method]
				results = fd.paramCheck(paramMap, consumesNode, results, path, method, context, false)
			}

			// are there top params defined?
			if topParams != nil {
				results = fd.paramCheck(topParams, consumesNode, results, path, method, context, true)
			}
		}
	}

	return results
}

func (fd FormDataConsumeCheck) paramCheck(paramMap map[string][]*index.Reference, consumesNode *yaml.Node,
	results []model.RuleFunctionResult, path string, method string, context model.RuleFunctionContext, top bool) []model.RuleFunctionResult {

	for paramName, paramNode := range paramMap {
		if paramNode == nil {
			results = append(results, model.RuleFunctionResult{
				Message:   fmt.Sprintf("parameter value for '%s' is empty / missing", paramName),
				StartNode: consumesNode,
				EndNode:   vacuumUtils.BuildEndNode(consumesNode),
				Path:      fmt.Sprintf("$.paths['%s'].%s.parameters", path, method),
				Rule:      context.Rule,
			})
			continue
		}

		for r := range paramNode {
			inNodeStart, inNode := utils.FindKeyNode("in", paramNode[r].Node.Content)
			if inNode != nil && inNode.Value == "formData" {

				pathString := fmt.Sprintf("$.paths['%s'].%s.parameters", path, method)
				if top {
					pathString = fmt.Sprintf("$.paths['%s'].parameters", path)
				}

				// using formData without a consumes sequence.
				if consumesNode == nil {
					results = append(results, model.RuleFunctionResult{
						Message:   fmt.Sprintf("in:formData param '%s' used without 'consumes' defined", paramName),
						StartNode: inNodeStart,
						EndNode:   vacuumUtils.BuildEndNode(inNodeStart),
						Path:      pathString,
						Rule:      context.Rule,
					})
				}

				validConsumer := false
				if consumesNode != nil {
					for _, consumeNode := range consumesNode.Content {
						switch consumeNode.Value {
						case "application/x-www-form-urlencoded":
							validConsumer = true
						case "multipart/form-data":
							validConsumer = true
						}
					}
				}

				if !validConsumer {
					results = append(results, model.RuleFunctionResult{
						Message: fmt.Sprintf("in:formData param '%s' parameter must include 'application/x-www-form-urlencoded'"+
							" or 'multipart/form-data' in their 'consumes' property", paramName),
						StartNode: inNodeStart,
						EndNode:   vacuumUtils.BuildEndNode(inNodeStart),
						Path:      pathString,
						Rule:      context.Rule,
					})
				}
			}
		}

	}
	return results
}
