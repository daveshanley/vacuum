// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/base"
	"gopkg.in/yaml.v3"
	"strings"
)

// NoRequestBody is a rule that checks operations are using tags and they are not empty.
type NoRequestBody struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the NoRequestBody rule.
func (r NoRequestBody) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "noRequestBody",
	}
}

// GetCategory returns the category of the TagDefined rule.
func (r NoRequestBody) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the NoRequestBody rule, based on supplied context and a supplied []*yaml.Node slice.
func (r NoRequestBody) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	paths := context.DrDocument.V3Document.Paths
	if paths != nil {
		for pathItemPairs := paths.PathItems.First(); pathItemPairs != nil; pathItemPairs = pathItemPairs.Next() {
			path := pathItemPairs.Key()
			v := pathItemPairs.Value()

			for opPairs := v.GetOperations().First(); opPairs != nil; opPairs = opPairs.Next() {
				method := opPairs.Key()
				op := opPairs.Value()

				for _, checkedMethods := range []string{"GET", "DELETE"} {
					if strings.EqualFold(method, checkedMethods) {
						if op.RequestBody != nil {

							res := model.RuleFunctionResult{
								Message: vacuumUtils.SuppliedOrDefault(context.Rule.Message, fmt.Sprintf("`%s` operation should not have a requestBody defined",
									strings.ToUpper(method))),
								StartNode: op.Value.GoLow().KeyNode,
								EndNode:   vacuumUtils.BuildEndNode(op.Value.GoLow().KeyNode),
								Path:      fmt.Sprintf("$.paths['%s'].%s", path, method),
								Rule:      context.Rule,
							}
							results = append(results, res)
							op.AddRuleFunctionResult(base.ConvertRuleResult(&res))
						}
					}
				}
			}
		}
	}
	return results

}
