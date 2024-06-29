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

// OperationId is a rule that will check if each operation provides an operationId
type OperationId struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the OperationId rule.
func (oId OperationId) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "oasOpId",
	}
}

// GetCategory returns the category of the OperationId rule.
func (oId OperationId) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the OperationId rule, based on supplied context and a supplied []*yaml.Node slice.
func (oId OperationId) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

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

				if op.Value.OperationId == "" {
					res := model.RuleFunctionResult{
						Message: vacuumUtils.SuppliedOrDefault(context.Rule.Message, fmt.Sprintf("the `%s` operation does not contain an `operationId`",
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
	return results
}
