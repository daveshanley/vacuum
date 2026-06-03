// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package asyncapi

import (
	"github.com/daveshanley/vacuum/model"
	"go.yaml.in/yaml/v4"
)

// Document validates the parsed AsyncAPI document build state.
type Document struct{}

// GetSchema returns the AsyncAPI document validation function schema.
func (d Document) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "asyncApiDocument"}
}

// GetCategory returns the AsyncAPI function category.
func (d Document) GetCategory() string {
	return model.FunctionCategoryAsyncAPI
}

// RunRule emits parse/build errors collected by libasyncapi as lint results.
func (d Document) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	if context.AsyncAPI == nil {
		return []model.RuleFunctionResult{result(context, firstNode(nodes), "$", "AsyncAPI document context was not built.")}
	}
	options := context.GetOptionsStringMap()
	if options["resolved"] == "false" {
		return nil
	}

	var results []model.RuleFunctionResult
	for _, err := range context.AsyncAPI.DocumentErrors() {
		if err == nil {
			continue
		}
		results = append(results, result(context, context.AsyncAPI.Root(), "$", err.Error()))
	}
	return results
}

func firstNode(nodes []*yaml.Node) *yaml.Node {
	if len(nodes) == 0 {
		return nil
	}
	return nodes[0]
}
