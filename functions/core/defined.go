// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package core

import (
    "fmt"
    "github.com/daveshanley/vacuum/model"
    "github.com/pb33f/libopenapi/utils"
    "gopkg.in/yaml.v3"
)

// Defined is a rule that will determine if a field has been set on a node slice.
type Defined struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the Defined rule.
func (d Defined) GetSchema() model.RuleFunctionSchema {
    return model.RuleFunctionSchema{
        Name:          "defined",
        RequiresField: true,
    }
}

// RunRule will execute the Defined rule, based on supplied context and a supplied []*yaml.Node slice.
func (d Defined) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

    if len(nodes) <= 0 {
        return nil
    }

    var results []model.RuleFunctionResult

    pathValue := "unknown"
    if path, ok := context.Given.(string); ok {
        pathValue = path
    }

    for _, node := range nodes {
        fieldNode, _ := utils.FindKeyNode(context.RuleAction.Field, node.Content)
        if fieldNode == nil {
            results = append(results, model.RuleFunctionResult{
                Message:   fmt.Sprintf("%s: '%s' must be defined", context.Rule.Description, context.RuleAction.Field),
                StartNode: node,
                EndNode:   node,
                Path:      pathValue,
                Rule:      context.Rule,
            })
        }
    }

    return results
}
