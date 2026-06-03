// Copyright 2025 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"go.yaml.in/yaml/v4"
)

const (
	propertyCaseCamel       = "camel"
	propertyCasePascal      = "pascal"
	propertyCasePascalKebab = "pascal-kebab"
	propertyCaseKebab       = "kebab"
	propertyCaseCobol       = "cobol"
	propertyCaseSnake       = "snake"
	propertyCaseMacro       = "macro"
	propertyCaseFlat        = "flat"
)

var propertyCasePatterns = map[string]*regexp.Regexp{
	propertyCaseCamel:       regexp.MustCompile(`^[a-z][a-z0-9]*(?:[A-Z0-9](?:[a-z0-9]+|$))*$`),
	propertyCasePascal:      regexp.MustCompile(`^[A-Z][a-z0-9]*(?:[A-Z0-9](?:[a-z0-9]+|$))*$`),
	propertyCasePascalKebab: regexp.MustCompile(`^[A-Z][a-z0-9]*(-[A-Z][a-z0-9]*)*$`),
	propertyCaseKebab:       regexp.MustCompile(`^[a-z0-9-]+$`),
	propertyCaseCobol:       regexp.MustCompile(`^[A-Z0-9-]+$`),
	propertyCaseSnake:       regexp.MustCompile(`^[a-z0-9_]+$`),
	propertyCaseMacro:       regexp.MustCompile(`^[A-Z0-9_]+$`),
	propertyCaseFlat:        regexp.MustCompile(`^[a-z][a-z0-9]*$`),
}

var propertyCaseAliases = map[string]string{
	"camel":                propertyCaseCamel,
	"camelcase":            propertyCaseCamel,
	"pascal":               propertyCasePascal,
	"pascalcase":           propertyCasePascal,
	"pascal-kebab":         propertyCasePascalKebab,
	"pascalkebab":          propertyCasePascalKebab,
	"kebab":                propertyCaseKebab,
	"kebab-case":           propertyCaseKebab,
	"cobol":                propertyCaseCobol,
	"cobol-case":           propertyCaseCobol,
	"snake":                propertyCaseSnake,
	"snake_case":           propertyCaseSnake,
	"macro":                propertyCaseMacro,
	"macro-case":           propertyCaseMacro,
	"screaming_snake_case": propertyCaseMacro,
	"flat":                 propertyCaseFlat,
	"flatcase":             propertyCaseFlat,
	"lowercase":            propertyCaseFlat,
}

var propertyCaseDisplayNames = map[string]string{
	propertyCaseCamel:       "camelCase",
	propertyCasePascal:      "PascalCase",
	propertyCasePascalKebab: "Pascal-Kebab-Case",
	propertyCaseKebab:       "kebab-case",
	propertyCaseCobol:       "SCREAMING-KEBAB-CASE",
	propertyCaseSnake:       "snake_case",
	propertyCaseMacro:       "SCREAMING_SNAKE_CASE",
	propertyCaseFlat:        "lowercase",
}

// CamelCaseProperties checks that all schema property names are in a configured case format.
type CamelCaseProperties struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the CamelCaseProperties rule.
func (ccp CamelCaseProperties) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name:          "oasCamelCaseProperties",
		MaxProperties: 1,
		Properties: []model.RuleFunctionProperty{
			{
				Name:        "type",
				Description: "optional property casing style. Supported values: camel, snake, pascal, kebab, macro, cobol, flat, pascal-kebab. Defaults to camel",
			},
		},
		ErrorMessage: "'oasCamelCaseProperties' function accepts an optional 'type' value such as 'camel' or 'snake'",
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

	expectedCase := ccp.resolveExpectedCase(context)
	expectedCaseDisplay := ccp.displayCaseType(expectedCase)
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

	checkSchema := func(schema *v3.Schema) {
		if schema == nil || schema.Value == nil {
			return
		}

		key := schema.GenerateJSONPath()
		if seen[key] {
			return
		}
		seen[key] = true

		for propertyName, prop := range schema.Value.Properties.FromOldest() {
			if !ccp.matchesConfiguredCase(propertyName, expectedCase) {
				caseType := ccp.identifyCaseType(propertyName)
				path := fmt.Sprintf("%s.properties['%s']", schema.GenerateJSONPath(), propertyName)
				message := ccp.formatCaseMismatchMessage(propertyName, caseType, expectedCaseDisplay)
				results = append(results, buildResult(message, path, prop.GoLow().GetKeyNode(), schema))
			}
		}
	}

	if context.DrDocument.Schemas != nil {
		for i := range context.DrDocument.Schemas {
			checkSchema(context.DrDocument.Schemas[i])
		}
	}

	return results
}

func (ccp CamelCaseProperties) formatCaseMismatchMessage(propertyName, caseType, expectedCaseDisplay string) string {
	if caseType == "unknown" {
		return fmt.Sprintf("property '%s' is using an unclassified case, not %s", propertyName, expectedCaseDisplay)
	}
	return fmt.Sprintf("property `%s` is `%s` not `%s`", propertyName, caseType, expectedCaseDisplay)
}

// isCamelCase checks if a string is in camelCase format.
func (ccp CamelCaseProperties) isCamelCase(s string) bool {
	return ccp.matchesConfiguredCase(s, propertyCaseCamel)
}

func (ccp CamelCaseProperties) resolveExpectedCase(context model.RuleFunctionContext) string {
	if context.Options == nil {
		return propertyCaseCamel
	}
	if normalized, ok := normalizePropertyCaseType(context.GetOptionsStringMap()["type"]); ok {
		return normalized
	}
	return propertyCaseCamel
}

func (ccp CamelCaseProperties) matchesConfiguredCase(s, expectedCase string) bool {
	pattern, ok := propertyCasePatterns[expectedCase]
	if !ok || s == "" {
		return false
	}
	return pattern.MatchString(s)
}

// identifyCaseType determines what case type a string is using.
func (ccp CamelCaseProperties) identifyCaseType(s string) string {
	if s == "" {
		return "unknown"
	}

	hasUnderscore := strings.Contains(s, "_")
	hasHyphen := strings.Contains(s, "-")
	hasSpecialChars := false
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			hasSpecialChars = true
			break
		}
	}
	if hasSpecialChars {
		return "unknown"
	}

	isAllUpper := strings.ToUpper(s) == s && strings.ToLower(s) != s
	isAllLower := strings.ToLower(s) == s && strings.ToUpper(s) != s

	switch {
	case hasUnderscore && isAllUpper:
		return "SCREAMING_SNAKE_CASE"
	case hasHyphen && isAllUpper:
		return "SCREAMING-KEBAB-CASE"
	case hasUnderscore && isAllLower:
		return "snake_case"
	case hasHyphen && isAllLower:
		return "kebab-case"
	case hasHyphen && ccp.matchesConfiguredCase(s, propertyCasePascalKebab):
		return ccp.displayCaseType(propertyCasePascalKebab)
	case hasUnderscore:
		return "unknown"
	case hasHyphen:
		return "unknown"
	case isAllUpper:
		return "UPPERCASE"
	case isAllLower:
		return "lowercase"
	case ccp.matchesConfiguredCase(s, propertyCaseCamel):
		return ccp.displayCaseType(propertyCaseCamel)
	case ccp.matchesConfiguredCase(s, propertyCasePascal):
		return ccp.displayCaseType(propertyCasePascal)
	default:
		return "unknown"
	}
}

func (ccp CamelCaseProperties) displayCaseType(caseType string) string {
	if display, ok := propertyCaseDisplayNames[caseType]; ok {
		return display
	}
	return caseType
}

func normalizePropertyCaseType(caseType string) (string, bool) {
	key := strings.ToLower(strings.TrimSpace(caseType))
	if key == "" {
		return propertyCaseCamel, true
	}
	normalized, ok := propertyCaseAliases[key]
	return normalized, ok
}
