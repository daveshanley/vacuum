// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/base"
	"gopkg.in/yaml.v3"
)

// TagDescription will check that all tags have a description.
type TagDescription struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the TagDescription rule.
func (td TagDescription) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "tagDescription",
	}
}

// GetCategory returns the category of the TagDescription rule.
func (td TagDescription) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the TagDescription rule, based on supplied context and a supplied []*yaml.Node slice.
func (td TagDescription) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	tags := context.DrDocument.V3Document.Tags

	for x, tag := range tags {
		if tag.Value.Description == "" {
			res := model.RuleFunctionResult{
				Message:   vacuumUtils.SuppliedOrDefault(context.Rule.Message, fmt.Sprintf("tag `%s` must have a description", tag.Value.Name)),
				StartNode: tag.Value.GoLow().RootNode,
				EndNode:   vacuumUtils.BuildEndNode(tag.Value.GoLow().RootNode),
				Path:      fmt.Sprintf("$.tags[%d]", x),
				Rule:      context.Rule,
			}
			results = append(results, res)
			tag.AddRuleFunctionResult(base.ConvertRuleResult(&res))
		}
	}
	return results
}
