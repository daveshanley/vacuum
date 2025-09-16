// Copyright 2025 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"strings"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	libopenapi_base "github.com/pb33f/libopenapi/datamodel/high/base"
	"go.yaml.in/yaml/v4"
)

// UnnecessaryCombinator checks for schema combinators (allOf, anyOf, oneOf) that have only a single item,
// which makes them unnecessary and should be replaced with the single item directly.
type UnnecessaryCombinator struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the UnnecessaryCombinator rule.
func (uc UnnecessaryCombinator) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "oasUnnecessaryCombinator",
	}
}

// GetCategory returns the category of the UnnecessaryCombinator rule.
func (uc UnnecessaryCombinator) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the UnnecessaryCombinator rule, based on supplied context and a supplied []*yaml.Node slice.
func (uc UnnecessaryCombinator) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	// cache to prevent checking the same schema twice
	seen := make(map[string]bool)

	buildResult := func(message, path string, node *yaml.Node, component v3.AcceptsRuleResults) model.RuleFunctionResult {
		// try to find all paths for this node if it's a schema
		var allPaths []string
		if schema, ok := component.(*v3.Schema); ok {
			_, allPaths = vacuumUtils.LocateSchemaPropertyPaths(context, schema, node, node)
		}

		result := model.RuleFunctionResult{
			Message:   message,
			StartNode: node,
			EndNode:   vacuumUtils.BuildEndNode(node),
			Path:      path,
			Rule:      context.Rule,
		}

		// set the Paths array if we found multiple locations
		if len(allPaths) > 1 {
			result.Paths = allPaths
		}

		component.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
		return result
	}

	checkCombinator := func(schema *v3.Schema, combinatorName string, combinatorSlice []*libopenapi_base.SchemaProxy,
		keyNode *yaml.Node) {
		if len(combinatorSlice) == 1 {
			path := fmt.Sprintf("%s.%s", schema.GenerateJSONPath(), combinatorName)
			message := fmt.Sprintf("schema with `%s` combinator containing only one item "+
				"should be replaced with the item directly", combinatorName)
			result := buildResult(message, path, keyNode, schema)
			results = append(results, result)
		}
	}

	checkSchema := func(schema *v3.Schema) {
		if schema == nil || schema.Value == nil {
			return
		}

		// create cache key to prevent duplicate processing
		var cacheKey strings.Builder
		cacheKey.WriteString(schema.GenerateJSONPath())
		key := cacheKey.String()

		if seen[key] {
			return
		}
		seen[key] = true

		// check allOf combinator
		if schema.Value.AllOf != nil && len(schema.Value.AllOf) > 0 {
			checkCombinator(schema, "allOf", schema.Value.AllOf, schema.Value.GoLow().AllOf.GetKeyNode())
		}

		// check anyOf combinator
		if schema.Value.AnyOf != nil && len(schema.Value.AnyOf) > 0 {
			checkCombinator(schema, "anyOf", schema.Value.AnyOf, schema.Value.GoLow().AnyOf.GetKeyNode())
		}

		// check oneOf combinator
		if schema.Value.OneOf != nil && len(schema.Value.OneOf) > 0 {
			checkCombinator(schema, "oneOf", schema.Value.OneOf, schema.Value.GoLow().OneOf.GetKeyNode())
		}
	}

	// check all schemas in the document
	if context.DrDocument.Schemas != nil {
		for i := range context.DrDocument.Schemas {
			checkSchema(context.DrDocument.Schemas[i])
		}
	}

	return results
}
