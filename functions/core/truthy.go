// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package core

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"gopkg.in/yaml.v3"
)

// Truthy is a rule that will determine if something is seen as 'true' (could be a 1 or "pizza", or actually 'true')
type Truthy struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the Truthy rule.
func (t Truthy) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "truthy",
	}
}

// RunRule will execute the Truthy rule, based on supplied context and a supplied []*yaml.Node slice.
func (t *Truthy) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult
	pathValue := "unknown"
	if path, ok := context.Given.(string); ok {
		pathValue = path
	}
	for _, node := range nodes {

		fieldNode, fieldNodeValue := utils.FindKeyNode(context.RuleAction.Field, node.Content)
		if fieldNode == nil && fieldNodeValue == nil ||
			fieldNodeValue.Value == "" || fieldNodeValue.Value == "false" ||
			fieldNodeValue.Value == "0" {
			results = append(results, model.RuleFunctionResult{
				Message:   fmt.Sprintf("%s: '%s' must be set", context.Rule.Description, context.RuleAction.Field),
				StartNode: node,
				EndNode:   node.Content[len(node.Content)-1],
				Path:      pathValue,
			})
		}

	}

	return results
}
