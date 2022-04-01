// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"gopkg.in/yaml.v3"
)

// NoRefSiblings will check for anything placed next to a $ref (like a description) and will throw some shade if
// something is found. This rule is there to prevent us from  adding useless properties to a $ref child.
type NoRefSiblings struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the NoRefSiblings rule.
func (nrs NoRefSiblings) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "no_ref_siblings",
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
		results = append(results, model.RuleFunctionResult{
			Message:   fmt.Sprintf("a $ref cannot be placed next to any other properties"),
			StartNode: ref.Node,
			EndNode:   ref.Node,
			Path:      ref.Path,
			Rule:      context.Rule,
		})
	}
	return results
}
