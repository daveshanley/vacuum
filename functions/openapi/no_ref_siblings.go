// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

// NoRefSiblings will check for anything placed next to a $ref (like a description) and will throw some shade if
// something is found. This rule is there to prevent us from  adding useless properties to a $ref child.
type NoRefSiblings struct {
}

// GetCategory returns the category of the NoRefSiblings rule.
func (nrs NoRefSiblings) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the NoRefSiblings rule.
func (nrs NoRefSiblings) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "refSiblings",
	}
}

// RunRule will execute the NoRefSiblings rule, based on supplied context and a supplied []*yaml.Node slice.
func (nrs NoRefSiblings) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult
	siblings := context.Index.GetReferencesWithSiblings()
	for _, ref := range siblings {

		key, val := utils.FindKeyNode("$ref", ref.Node.Content)
		results = append(results, model.RuleFunctionResult{
			Message:   "a `$ref` cannot be placed next to any other properties",
			StartNode: key,
			EndNode:   vacuumUtils.BuildEndNode(val),
			Path:      ref.Path,
			Rule:      context.Rule,
		})
	}
	return results
}
