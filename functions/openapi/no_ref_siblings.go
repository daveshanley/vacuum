// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"go.yaml.in/yaml/v4"
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
	results = applyNoRefSiblingRuleToIndex(context.Index, results, context)

	rolodex := context.Index.GetRolodex()

	if rolodex != nil {
		for _, idx := range rolodex.GetIndexes() {
			results = applyNoRefSiblingRuleToIndex(idx, results, context)
		}
	}

	return results
}

// applyNoRefSiblingRuleToIndex is a helper that applies the NoRefSiblings rule to a given index and returns the results.
func applyNoRefSiblingRuleToIndex(idx *index.SpecIndex, results []model.RuleFunctionResult, context model.RuleFunctionContext) []model.RuleFunctionResult {
	siblings := idx.GetReferencesWithSiblings()
	for _, ref := range siblings {

		key, val := utils.FindKeyNode("$ref", ref.Node.Content)
		results = append(results, model.RuleFunctionResult{
			Message:   "a `$ref` cannot be placed next to any other properties",
			StartNode: key,
			Origin:    idx.FindNodeOrigin(val),
			EndNode:   vacuumUtils.BuildEndNode(val),
			Path:      ref.Path,
			Rule:      context.Rule,
		})
	}
	return results
}
