// Copyright 2025 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"go.yaml.in/yaml/v4"
)

// CamelCaseProperties checks that all schema property names are in camelCase format.
type CamelCaseProperties struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the CamelCaseProperties rule.
func (ccp CamelCaseProperties) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "oasCamelCaseProperties",
	}
}

// GetCategory returns the category of the CamelCaseProperties rule.
func (ccp CamelCaseProperties) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the CamelCaseProperties rule, based on supplied context and a supplied []*yaml.Node slice.
func (ccp CamelCaseProperties) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

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

	checkSchema := func(schema *v3.Schema) {
		if schema == nil || schema.Value == nil || schema.Value.Properties == nil {
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

		// check all property names
		for propertyName, prop := range schema.Value.Properties.FromOldest() {
			if !ccp.isCamelCase(propertyName) {
				caseType := ccp.identifyCaseType(propertyName)
				path := fmt.Sprintf("%s.properties['%s']", schema.GenerateJSONPath(), propertyName)
				message := fmt.Sprintf("property `%s` is `%s` not `camelCase`", propertyName, caseType)
				results = append(results, buildResult(message, path, prop.GoLow().GetKeyNode(), schema))
			}
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

// isCamelCase checks if a string is in camelCase format.
func (ccp CamelCaseProperties) isCamelCase(s string) bool {
	if s == "" {
		return false
	}

	// must start with lowercase letter
	if !unicode.IsLower(rune(s[0])) {
		return false
	}

	// must contain only letters and numbers, no underscores, hyphens, or other special chars
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}

	return true
}

// identifyCaseType determines what case type a string is using.
func (ccp CamelCaseProperties) identifyCaseType(s string) string {
	if s == "" {
		return "unknown"
	}

	// check for special characters first
	hasUnderscore := strings.Contains(s, "_")
	hasHyphen := strings.Contains(s, "-")
	hasSpecialChars := false
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' {
			hasSpecialChars = true
			break
		}
	}

	if hasSpecialChars {
		return "unknown"
	}

	// check for all uppercase/lowercase
	isAllUpper := strings.ToUpper(s) == s && strings.ToLower(s) != s // has letters and is all upper
	isAllLower := strings.ToLower(s) == s && strings.ToUpper(s) != s // has letters and is all lower

	// classify based on patterns
	switch {
	case hasUnderscore && isAllUpper:
		return "SCREAMING_SNAKE_CASE"
	case hasHyphen && isAllUpper:
		return "SCREAMING-KEBAB-CASE"
	case hasUnderscore && isAllLower:
		return "snake_case"
	case hasUnderscore:
		return "Snake_Case"
	case hasHyphen && isAllLower:
		return "kebab-case"
	case hasHyphen:
		return "Kebab-Case"
	case isAllUpper && !hasUnderscore && !hasHyphen:
		return "UPPERCASE"
	case isAllLower && !hasUnderscore && !hasHyphen:
		return "lowercase"
	case unicode.IsUpper(rune(s[0])) && !hasUnderscore && !hasHyphen:
		return "PascalCase"
	case unicode.IsLower(rune(s[0])) && !hasUnderscore && !hasHyphen:
		// this is called when isCamelCase failed, so it must have some issue
		return "mixedCase"
	default:
		return "unknown"
	}
}
