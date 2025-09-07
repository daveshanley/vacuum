// Copyright 2022-2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"

	"github.com/daveshanley/vacuum/model"
	vutils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

// ParameterDescription will check swagger spec parameters for a description. ($.parameters)
type ParameterDescription struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the ParameterDescription rule.
func (pd ParameterDescription) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "oasParamDescriptions",
	}
}

// GetCategory returns the category of the ParameterDescription rule.
func (pd ParameterDescription) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the ParameterDescription rule, based on supplied context and a supplied []*yaml.Node slice.
func (pd ParameterDescription) RunRule(nodes []*yaml.Node,
	context model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult

	// Use DrDocument if available (preferred for multifile spec support)
	if context.DrDocument != nil && context.DrDocument.V3Document != nil {
		return pd.runRuleWithDrDocument(context)
	}

	// Fallback to index-based approach for backward compatibility with tests
	if context.Index == nil {
		return results
	}
	return pd.runRuleWithIndex(nodes, context)
}

func (pd ParameterDescription) runRuleWithDrDocument(context model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult

	buildResult := func(message, path string, node *yaml.Node,
		component v3.AcceptsRuleResults, paths []string) model.RuleFunctionResult {
		result := model.RuleFunctionResult{
			Message:   message,
			StartNode: node,
			EndNode:   vutils.BuildEndNode(node),
			Path:      path,
			Rule:      context.Rule,
		}
		if len(paths) > 1 {
			result.Paths = paths
		}
		if component != nil {
			component.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
		}
		return result
	}

	// check component parameters
	components := context.DrDocument.V3Document.Components
	if components != nil && components.Parameters != nil {
		for key, paramValue := range components.Parameters.FromOldest() {
			if paramValue.Value.Description == "" {
				node := paramValue.Value.GoLow().RootNode
				primaryPath, allPaths := vutils.LocateComponentPaths(context, paramValue, node, node)

				results = append(results,
					buildResult(fmt.Sprintf("the parameter `%s` does not contain a description", key),
						primaryPath,
						node,
						paramValue,
						allPaths))
			}
		}
	}

	// check path and operation parameters
	paths := context.DrDocument.V3Document.Paths
	if paths != nil {
		for _, pathItem := range paths.PathItems.FromOldest() {
			// check path-level parameters
			if pathItem.Parameters != nil {
				for i, param := range pathItem.Parameters {
					if param != nil && param.Value.Description == "" {
						paramName := param.Value.Name
						if paramName == "" {
							paramName = fmt.Sprintf("parameter[%d]", i)
						}

						node := param.Value.GoLow().RootNode
						primaryPath, allPaths := vutils.LocateComponentPaths(context, param, node, node)

						results = append(results,
							buildResult(fmt.Sprintf(
								"the parameter `%s` does not contain a description", paramName),
								primaryPath,
								node,
								param,
								allPaths))
					}
				}
			}

			// check all operations in this path
			for _, operation := range pathItem.GetOperations().FromOldest() {
				// check operation parameters
				if operation.Value.Parameters != nil {
					for i, param := range operation.Value.Parameters {
						if param != nil && param.Description == "" {
							paramName := param.Name
							if paramName == "" {
								paramName = fmt.Sprintf("parameter[%d]", i)
							}

							// Build path from operation's base path plus parameter index
							path := fmt.Sprintf("%s.parameters[%d]", operation.GenerateJSONPath(), i)
							results = append(results,
								buildResult(fmt.Sprintf(
									"the parameter `%s` does not contain a description", paramName),
									path,
									param.GoLow().RootNode,
									nil,
									nil))
						}
					}
				}
			}
		}
	}

	return results
}

func (pd ParameterDescription) runRuleWithIndex(nodes []*yaml.Node,
	context model.RuleFunctionContext) []model.RuleFunctionResult {
	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	// Component parameters
	params := context.Index.GetAllParameters()
	msg := "the parameter `%s` does not contain a description"

	for id, param := range params {
		// No longer checking for 'in' field - validate all parameters
		_, desc := utils.FindKeyNodeTop("description", param.Node.Content)
		if desc == nil || desc.Value == "" {
			results = append(results, model.RuleFunctionResult{
				Message:   fmt.Sprintf(msg, id),
				StartNode: param.Node,
				EndNode:   vutils.BuildEndNode(param.Node),
				Path:      fmt.Sprintf("$.components.parameters['%s']", id),
				Rule:      context.Rule,
			})
		}
	}

	// Operation parameters
	opParams := context.Index.GetAllParametersFromOperations()
	for path, methodMap := range opParams {
		for method, paramMap := range methodMap {
			for pName, opParam := range paramMap {
				for _, param := range opParam {
					if param != nil && param.Node != nil {
						// No longer checking for 'in' field - validate all parameters
						_, desc := utils.FindKeyNodeTop("description", param.Node.Content)
						if desc == nil || desc.Value == "" {
							pathString := fmt.Sprintf("$.paths['%s'].%s.parameters", path, method)
							results = append(results, model.RuleFunctionResult{
								Message:   fmt.Sprintf(msg, pName),
								StartNode: param.Node,
								EndNode:   vutils.BuildEndNode(param.Node),
								Path:      pathString,
								Rule:      context.Rule,
							})
						}
					}
				}
			}
		}
	}

	return results
}
