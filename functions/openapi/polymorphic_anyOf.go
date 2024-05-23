// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"gopkg.in/yaml.v3"
)

// PolymorphicAnyOf checks that there is no polymorphism used, in particular 'anyOf'
type PolymorphicAnyOf struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the PolymorphicAnyOf rule.
func (pm PolymorphicAnyOf) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "oasPolymorphicAnyOf",
	}
}

// GetCategory returns the category of the PolymorphicAnyOf rule.
func (pm PolymorphicAnyOf) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the PolymorphicAnyOf rule, based on supplied context and a supplied []*yaml.Node slice.
func (pm PolymorphicAnyOf) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	// no need to search! the index already has what we need.
	refs := context.Index.GetPolyAnyOfReferences()

	for _, ref := range refs {
		results = append(results, model.RuleFunctionResult{
			Message:   fmt.Sprintf("`anyOf` polymorphic reference: %s", context.Rule.Description),
			StartNode: ref.Node,
			EndNode:   vacuumUtils.BuildEndNode(ref.Node),
			Path:      ref.Path,
			Rule:      context.Rule,
		})
	}

	return results
}
