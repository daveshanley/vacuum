// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
    "fmt"
    "github.com/daveshanley/vacuum/model"
    "github.com/pb33f/libopenapi/utils"
    "gopkg.in/yaml.v3"
)

// ParameterDescription will check swagger spec parameters for a description. ($.parameters)
type ParameterDescription struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the ParameterDescription rule.
func (pd ParameterDescription) GetSchema() model.RuleFunctionSchema {
    return model.RuleFunctionSchema{
        Name: "oas2_parameter_description",
    }
}

// RunRule will execute the ParameterDescription rule, based on supplied context and a supplied []*yaml.Node slice.
func (pd ParameterDescription) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

    if len(nodes) <= 0 {
        return nil
    }

    var results []model.RuleFunctionResult

    params := context.Index.GetAllParameters()
    opParams := context.Index.GetAllParametersFromOperations()

    msg := "the parameter `%s` does not contain a description"

    // look through top level params first.
    for id, param := range params {
        // only check if the param has an 'in' property.
        _, in := utils.FindKeyNodeTop("in", param.Node.Content)
        _, desc := utils.FindKeyNodeTop("description", param.Node.Content)
        lastNode := utils.FindLastChildNodeWithLevel(param.Node, 0)

        if in != nil {
            if desc == nil || desc.Value == "" {
                _, path := utils.ConvertComponentIdIntoPath(id)
                results = append(results, model.RuleFunctionResult{
                    Message:   fmt.Sprintf(msg, id),
                    StartNode: param.Node,
                    EndNode:   lastNode,
                    Path:      path,
                    Rule:      context.Rule,
                })
            }
        }
    }

    // look through all parameters from operations.
    for path, methodMap := range opParams {
        for method, paramMap := range methodMap {
            for pName, opParam := range paramMap {
                for _, param := range opParam {
                    if param != nil && param.Node != nil {
                        _, in := utils.FindKeyNodeTop("in", param.Node.Content)
                        _, desc := utils.FindKeyNodeTop("description", param.Node.Content)
                        lastNode := utils.FindLastChildNodeWithLevel(param.Node, 0)

                        if in != nil {
                            if desc == nil || desc.Value == "" {
                                pathString := fmt.Sprintf("$.paths.%s.%s.parameters", path, method)
                                results = append(results, model.RuleFunctionResult{
                                    Message:   fmt.Sprintf(msg, pName),
                                    StartNode: param.Node,
                                    EndNode:   lastNode,
                                    Path:      pathString,
                                    Rule:      context.Rule,
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
