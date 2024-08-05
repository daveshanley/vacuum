// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package core

import (
	"fmt"
	vacuumUtils "github.com/daveshanley/vacuum/utils"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

// Falsy is a rule that will determine if something is seen as 'false' (could be a 0 or missing, or actually 'false')
type Falsy struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the Falsy rule.
func (f Falsy) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "falsy",
	}
}

// GetCategory returns the category of the Falsy rule.
func (f Falsy) GetCategory() string {
	return model.FunctionCategoryCore
}

// RunRule will execute the Falsy rule, based on supplied context and a supplied []*yaml.Node slice.
func (f Falsy) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	pathValue := "unknown"
	if path, ok := context.Given.(string); ok {
		pathValue = path
	}

	ruleMessage := context.Rule.Description
	if context.Rule.Message != "" {
		ruleMessage = context.Rule.Message
	}

	for _, node := range nodes {

		fieldNode, fieldNodeValue := utils.FindKeyNode(context.RuleAction.Field, node.Content)
		if (fieldNode != nil && fieldNodeValue != nil) &&
			(fieldNodeValue.Value != "" && fieldNodeValue.Value != "false" && fieldNodeValue.Value != "0" || (fieldNodeValue.Value == "" && fieldNodeValue.Content != nil)) {
			locatedObject, err := context.DrDocument.LocateModel(node)
			locatedPath := pathValue
			if err == nil && locatedObject != nil {
				locatedPath = locatedObject.GenerateJSONPath()
			}
			results = append(results, model.RuleFunctionResult{
				Message:   fmt.Sprintf("%s: `%s` must be falsy", ruleMessage, context.RuleAction.Field),
				StartNode: node,
				EndNode:   vacuumUtils.BuildEndNode(node),
				Path:      locatedPath,
				Rule:      context.Rule,
			})
		}
	}

	return results
}
