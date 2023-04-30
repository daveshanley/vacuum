// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
    "fmt"
    "github.com/daveshanley/vacuum/model"
    "github.com/pb33f/libopenapi/utils"
    "gopkg.in/yaml.v3"
    "strconv"
)

// Operation4xResponse is a rule that checks if an operation returns a 4xx (user error) code.
type Operation4xResponse struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the SuccessResponse rule.
func (or Operation4xResponse) GetSchema() model.RuleFunctionSchema {
    return model.RuleFunctionSchema{Name: "operation_4xx_response"}
}

// RunRule will execute the Operation4xResponse rule, based on supplied context and a supplied []*yaml.Node slice.
func (or Operation4xResponse) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

    if len(nodes) <= 0 {
        return nil
    }

    var results []model.RuleFunctionResult

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
        for m, method := range op.Content {
            if m%2 == 0 {
                opMethod = method.Value
                continue
            }
            basePath := fmt.Sprintf("$.paths.%s.%s", opPath, opMethod)
            _, responsesNode := utils.FindKeyNode("responses", method.Content)

            if responsesNode != nil {
                seen := false
                for k, response := range responsesNode.Content {
                    if k%2 != 0 {
                        continue
                    }
                    responseCode, _ := strconv.Atoi(response.Value)
                    if responseCode >= 400 && responseCode <= 499 {
                        seen = true
                    }
                }
                if !seen {
                    results = append(results, model.RuleFunctionResult{
                        Message:   "Operation must define at least one 4xx error response",
                        StartNode: method,
                        EndNode:   utils.FindLastChildNodeWithLevel(method, 0),
                        Path:      basePath,
                        Rule:      context.Rule,
                    })
                }
            }
        }
    }
    return results
}
