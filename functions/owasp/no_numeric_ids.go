// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/doctor/model/high/base"
	drV3 "github.com/pb33f/doctor/model/high/v3"
	"gopkg.in/yaml.v3"
	"slices"
	"strings"
)

type NoNumericIds struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the DefineError rule.
func (ni NoNumericIds) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "no_numeric_ids"}
}

// RunRule will execute the DefineError rule, based on supplied context and a supplied []*yaml.Node slice.
func (ni NoNumericIds) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	checkParam := func(param *drV3.Parameter) {
		if strings.ToLower(param.Value.Name) == "id" ||
			strings.HasSuffix(strings.ToLower(param.Value.Name), "_id") ||
			strings.HasSuffix(strings.ToLower(param.Value.Name), "-id") {
			if param.Value.Schema != nil {
				if slices.Contains(param.SchemaProxy.Schema.Value.Type, "integer") {
					node := param.SchemaProxy.Schema.Value.GoLow().Type.KeyNode
					result := model.RuleFunctionResult{
						Message:   "don't use numeric IDs, use random IDs that cannot be guessed like UUIDs",
						StartNode: node,
						EndNode:   node,
						Path:      fmt.Sprintf("%s.%s", param.SchemaProxy.Schema.GenerateJSONPath(), "type"),
						Rule:      context.Rule,
					}
					param.SchemaProxy.Schema.AddRuleFunctionResult(base.ConvertRuleResult(&result))
					results = append(results, result)
				}
			}
		}
	}
	if context.DrDocument.V3Document.Paths != nil && context.DrDocument.V3Document.Paths.PathItems != nil {
		for pathPairs := context.DrDocument.V3Document.Paths.PathItems.First(); pathPairs != nil; pathPairs = pathPairs.Next() {
			for _, param := range pathPairs.Value().Parameters {
				checkParam(param)
			}
			if pathPairs.Value().Get != nil {
				for _, param := range pathPairs.Value().Get.Parameters {
					checkParam(param)
				}
			}
			if pathPairs.Value().Post != nil {
				for _, param := range pathPairs.Value().Post.Parameters {
					checkParam(param)
				}
			}
			if pathPairs.Value().Put != nil {
				for _, param := range pathPairs.Value().Put.Parameters {
					checkParam(param)
				}
			}
			if pathPairs.Value().Patch != nil {
				for _, param := range pathPairs.Value().Patch.Parameters {
					checkParam(param)
				}
			}
			if pathPairs.Value().Delete != nil {
				for _, param := range pathPairs.Value().Delete.Parameters {
					checkParam(param)
				}
			}
			if pathPairs.Value().Head != nil {
				for _, param := range pathPairs.Value().Head.Parameters {
					checkParam(param)
				}
			}
			if pathPairs.Value().Trace != nil {
				for _, param := range pathPairs.Value().Trace.Parameters {
					checkParam(param)
				}
			}
			if pathPairs.Value().Options != nil {
				for _, param := range pathPairs.Value().Options.Parameters {
					checkParam(param)
				}
			}
			if pathPairs.Value().Delete != nil {
				for _, param := range pathPairs.Value().Delete.Parameters {
					checkParam(param)
				}
			}
		}
	}
	if context.DrDocument.V3Document.Components != nil && context.DrDocument.V3Document.Components.Parameters != nil {
		for paramPairs := context.DrDocument.V3Document.Components.Parameters.First(); paramPairs != nil; paramPairs = paramPairs.Next() {
			checkParam(paramPairs.Value())
		}
	}
	return results
}
