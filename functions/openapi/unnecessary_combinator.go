// Copyright 2025 Princess Beef Heavy Industries / Dave Shanley
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

	// in OAS 3.0.x, $ref siblings are ignored, so allOf is the only way to add properties
	isOAS30 := false
	if context.SpecInfo != nil {
		isOAS30 = context.SpecInfo.VersionNumeric >= 3.0 && context.SpecInfo.VersionNumeric < 3.1
	}

	seen := make(map[string]bool)

	buildResult := func(message, path string, node *yaml.Node, component v3.AcceptsRuleResults) model.RuleFunctionResult {
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

		if len(allPaths) > 1 {
			result.Paths = allPaths
		}

		component.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
		return result
	}

	checkCombinator := func(schema *v3.Schema, combinatorName string, combinatorSlice []*libopenapi_base.SchemaProxy,
		keyNode *yaml.Node) {
		if len(combinatorSlice) == 1 {
			// OAS 3.0.x: allOf with single $ref + sibling properties is legitimate
			if isOAS30 && combinatorName == "allOf" &&
				hasSiblingProperties(schema) && hasRefInCombinator(combinatorSlice) {
				return
			}

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

		var cacheKey strings.Builder
		cacheKey.WriteString(schema.GenerateJSONPath())
		key := cacheKey.String()

		if seen[key] {
			return
		}
		seen[key] = true

		if schema.Value.AllOf != nil && len(schema.Value.AllOf) > 0 {
			checkCombinator(schema, "allOf", schema.Value.AllOf, schema.Value.GoLow().AllOf.GetKeyNode())
		}

		if schema.Value.AnyOf != nil && len(schema.Value.AnyOf) > 0 {
			checkCombinator(schema, "anyOf", schema.Value.AnyOf, schema.Value.GoLow().AnyOf.GetKeyNode())
		}

		if schema.Value.OneOf != nil && len(schema.Value.OneOf) > 0 {
			checkCombinator(schema, "oneOf", schema.Value.OneOf, schema.Value.GoLow().OneOf.GetKeyNode())
		}
	}

	if context.DrDocument.Schemas != nil {
		for i := range context.DrDocument.Schemas {
			checkSchema(context.DrDocument.Schemas[i])
		}
	}

	return results
}

func hasSiblingProperties(schema *v3.Schema) bool {
	if schema == nil || schema.Value == nil {
		return false
	}
	v := schema.Value

	if v.Description != "" || v.Title != "" {
		return true
	}
	if v.Default != nil || v.Example != nil || v.ExternalDocs != nil {
		return true
	}
	if v.Nullable != nil || v.ReadOnly != nil || v.WriteOnly != nil || v.Deprecated != nil {
		return true
	}
	if v.XML != nil {
		return true
	}
	if len(v.Enum) > 0 {
		return true
	}
	lowSchema := v.GoLow()
	if lowSchema != nil && lowSchema.Extensions != nil && lowSchema.Extensions.Len() > 0 {
		return true
	}
	return false
}

func hasRefInCombinator(combinatorSlice []*libopenapi_base.SchemaProxy) bool {
	if len(combinatorSlice) != 1 || combinatorSlice[0] == nil {
		return false
	}
	lowProxy := combinatorSlice[0].GoLow()
	return lowProxy != nil && lowProxy.GetReference() != ""
}
