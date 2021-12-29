// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package core

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
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

// RunRule will execute the Falsy rule, based on supplied context and a supplied []*yaml.Node slice.
func (f Falsy) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult
	for _, node := range nodes {

		fieldNode, fieldNodeValue := utils.FindKeyNode(context.RuleAction.Field, node.Content)
		if (fieldNode != nil && fieldNodeValue != nil) &&
			(fieldNodeValue.Value != "" && fieldNodeValue.Value != "false" || fieldNodeValue.Value != "0") {
			results = append(results, model.RuleFunctionResult{
				Message: fmt.Sprintf("'%s' must be falsy", context.RuleAction.Field),
			})
		}
	}

	return results
}
