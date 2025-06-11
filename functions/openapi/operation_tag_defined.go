// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
    "fmt"
    "github.com/daveshanley/vacuum/model"
    vacuumUtils "github.com/daveshanley/vacuum/utils"
    "github.com/pb33f/doctor/model/high/v3"
    "gopkg.in/yaml.v3"
    "strings"
)

// TagDefined is a rule that checks if an operation uses a tag, it's also defined in the global tag definitions.
type TagDefined struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the TagDefined rule.
func (td TagDefined) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "oasTagDefined",
	}
}

// GetCategory returns the category of the TagDefined rule.
func (td TagDefined) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the TagDefined rule, based on supplied context and a supplied []*yaml.Node slice.
func (td TagDefined) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	globalTags := context.DrDocument.V3Document.Tags
	globalTagMap := make(map[string]*v3.Tag)
	for _, tag := range globalTags {
		globalTagMap[tag.Value.Name] = tag
	}

	paths := context.DrDocument.V3Document.Paths
	if paths != nil && paths.PathItems != nil {
		for pathItemPairs := paths.PathItems.First(); pathItemPairs != nil; pathItemPairs = pathItemPairs.Next() {
			path := pathItemPairs.Key()
			v := pathItemPairs.Value()
			for opPairs := v.GetOperations().First(); opPairs != nil; opPairs = opPairs.Next() {
				method := opPairs.Key()
				op := opPairs.Value()
				for i, tag := range op.Value.Tags {
					if _, ok := globalTagMap[tag]; !ok {
						res := model.RuleFunctionResult{
							Message: vacuumUtils.SuppliedOrDefault(context.Rule.Message,
								fmt.Sprintf("tag `%s` for `%s` operation is not defined as a global tag",
									tag, strings.ToUpper(method))),
							StartNode: op.Value.GoLow().Tags.Value[i].ValueNode,
							EndNode:   vacuumUtils.BuildEndNode(op.Value.GoLow().Tags.Value[i].ValueNode),
							Path:      fmt.Sprintf("$.paths['%s'].%s.tags[%v]", path, method, i),
							Rule:      context.Rule,
						}
						results = append(results, res)
						op.AddRuleFunctionResult(v3.ConvertRuleResult(&res))
					}
				}
			}
		}
	}
	return results
}
