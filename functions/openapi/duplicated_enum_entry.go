// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

// DuplicatedEnum will check enum values match the types provided
type DuplicatedEnum struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the DuplicatedEnum rule.
func (de DuplicatedEnum) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "duplicated_enum",
	}
}

// RunRule will execute the DuplicatedEnum rule, based on supplied context and a supplied []*yaml.Node slice.
func (de DuplicatedEnum) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	enums := context.Index.GetAllEnums()

	for _, enum := range enums {

		duplicates := utils.CheckEnumForDuplicates(enum.Node.Content)

		startNode := enum.Node
		endNode := enum.Node

		// iterate through duplicate results and add results.
		for _, res := range duplicates {
			results = append(results, model.RuleFunctionResult{
				Message:   fmt.Sprintf("enum contains a duplicate: %s", res.Value),
				StartNode: startNode,
				EndNode:   endNode,
				Path:      fmt.Sprintf("%v", context.Given),
				Rule:      context.Rule,
			})
		}
	}
	return results
}
