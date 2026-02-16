// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"go.yaml.in/yaml/v4"
	"strings"
)

// PathItemReferences checks that operations within path items are not using $ref.
// $ref is only valid directly on the path item object itself, not on individual operations.
type PathItemReferences struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the PathItemReferences rule.
func (id PathItemReferences) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "pathItemReferences",
	}
}

// GetCategory returns the category of the PathItemReferences rule.
func (id PathItemReferences) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the PathItemReferences rule, based on supplied context and a supplied []*yaml.Node slice.
func (id PathItemReferences) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	paths := context.DrDocument.V3Document.Paths

	if paths != nil {

		for pathKey, pathItem := range paths.PathItems.FromOldest() {

			for method, op := range pathItem.GetOperations().FromOldest() {

				if op.Value.GoLow().IsReference() {
					res := model.RuleFunctionResult{
						Message: vacuumUtils.SuppliedOrDefault(context.Rule.Message,
							fmt.Sprintf("`%s` operation at path `%s` uses a $ref, which is not valid; "+
								"$ref is only allowed on the path item itself", strings.ToUpper(method), pathKey)),
						StartNode: op.Value.GoLow().KeyNode,
						EndNode:   vacuumUtils.BuildEndNode(op.Value.GoLow().KeyNode),
						Path:      fmt.Sprintf("$.paths['%s'].%s", pathKey, method),
						Rule:      context.Rule,
					}
					results = append(results, res)
					op.AddRuleFunctionResult(v3.ConvertRuleResult(&res))
				}
			}
		}
	}

	return results
}
