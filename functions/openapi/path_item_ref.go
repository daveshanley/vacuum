// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/base"
	"gopkg.in/yaml.v3"
)

// PathItemReferences will check that path items are not using references, as they are not allowed
// although many folks do it. This is a common mistake, this will help catch it.
type PathItemReferences struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the PathItemReferences rule.
func (id PathItemReferences) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "pathItemReferences",
	}
}

// GetCategory returns the category of the InfoContact rule.
func (id PathItemReferences) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the InfoContact rule, based on supplied context and a supplied []*yaml.Node slice.
func (id PathItemReferences) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	paths := context.DrDocument.V3Document.Paths

	if paths != nil {

		for _, path := range paths.PathItems.FromOldest() {

			if path.Value.GoLow().IsReference() {
				res := model.RuleFunctionResult{
					Message: vacuumUtils.SuppliedOrDefault(context.Rule.Message,
						fmt.Sprintf("path `%s` item uses a $ref, it's technically not allowed", path.Key)),
					StartNode: path.Value.GoLow().KeyNode,
					EndNode:   vacuumUtils.BuildEndNode(path.Value.GoLow().KeyNode),
					Path:      path.GenerateJSONPath(),
					Rule:      context.Rule,
				}
				results = append(results, res)
				path.AddRuleFunctionResult(base.ConvertRuleResult(&res))
			}
		}
	}

	return results
}
