// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	drV3 "github.com/pb33f/doctor/model/high/v3"
	"go.yaml.in/yaml/v4"
	"slices"
	"strings"
)

type NoNumericIds struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the DefineError rule.
func (ni NoNumericIds) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "owaspNoNumericIds"}
}

// GetCategory returns the category of the NoNumericIds rule.
func (ni NoNumericIds) GetCategory() string {
	return model.FunctionCategoryOWASP
}

// RunRule will execute the DefineError rule, based on supplied context and a supplied []*yaml.Node slice.
func (ni NoNumericIds) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	checkParam := func(param *drV3.Parameter) {
		if param != nil && param.Value != nil {
			if strings.ToLower(param.Value.Name) == "id" ||
				strings.HasSuffix(strings.ToLower(param.Value.Name), "_id") ||
				strings.HasSuffix(strings.ToLower(param.Value.Name), "id") ||
				strings.HasSuffix(strings.ToLower(param.Value.Name), "-id") {
				if param.Value.Schema != nil {
					if slices.Contains(param.SchemaProxy.Schema.Value.Type, "integer") {
						node := param.SchemaProxy.Schema.Value.GoLow().Type.KeyNode
						result := model.RuleFunctionResult{
							Message: vacuumUtils.SuppliedOrDefault(context.Rule.Message,
								"don't use numeric IDs, use random IDs that cannot be guessed like UUIDs"),
							StartNode: node,
							EndNode:   vacuumUtils.BuildEndNode(node),
							Path:      fmt.Sprintf("%s.%s", param.SchemaProxy.Schema.GenerateJSONPath(), "type"),
							Rule:      context.Rule,
						}
						param.SchemaProxy.Schema.AddRuleFunctionResult(drV3.ConvertRuleResult(&result))
						results = append(results, result)
					}
				}
			}
		}
	}
	for _, param := range context.DrDocument.Parameters {
		checkParam(param)
	}
	return results
}
