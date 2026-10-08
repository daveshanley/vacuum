// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package rulesets

import (
	"github.com/daveshanley/vacuum/model"
	validation "github.com/pb33f/libopenapi-validator/arazzo"
)

// Arazzo ruleset names accepted by extends and ruleset generation.
const (
	VacuumArazzo               = "vacuum:arazzo"
	VacuumArazzoRecommended    = "arazzo-recommended"
	SpectralArazzo             = "spectral:arazzo"
	ArazzoValidationIncomplete = model.ArazzoValidationIncomplete
)

// GenerateDefaultArazzoRuleSet returns all Arazzo conformance and authoring rules.
func GenerateDefaultArazzoRuleSet() *RuleSet {
	return &RuleSet{DocumentationURI: "https://github.com/daveshanley/vacuum/blob/main/ARAZZO.md",
		Formats: model.ArazzoAllFormats, Description: "Arazzo 1.0 and 1.1 validation and authoring rules.",
		Rules: GetAllArazzoRules(), Extends: map[string]string{VacuumArazzo: VacuumAll}}
}

// GetAllArazzoRules returns fresh rules backed by one shared validator execution.
func GetAllArazzoRules() map[string]*model.Rule {
	rules := make(map[string]*model.Rule)
	checks := []struct {
		code        validation.Code
		description string
	}{
		{validation.CodeStructure, "Arazzo document structure must be valid."},
		{validation.CodeDuplicateID, "Workflow, step, source and action identifiers must be unique in their scope."},
		{validation.CodeReference, "Workflow, step and source references must identify valid targets."},
		{validation.CodeParameter, "Parameters must be valid for their target operation."},
		{validation.CodeExpression, "Runtime expressions must be valid in their context."},
		{validation.CodeSelector, "Selectors and success criteria must use valid syntax."},
		{validation.CodeDependency, "Workflow prerequisites must be valid and free of cycles."},
		{validation.CodeInputSchema, "Workflow input schemas must be valid."},
		{validation.CodeSourceType, "Source descriptions must match the referenced document type."},
	}
	for _, check := range checks {
		id := string(check.code)
		rules[id] = arazzoRule(id, check.description, "$", "", "arazzoDocument", map[string]string{"code": id}, model.SeverityError, true)
	}
	rules[ArazzoValidationIncomplete] = arazzoRule(ArazzoValidationIncomplete, "All applicable static checks should complete.", "$", "", "arazzoDocument", map[string]string{"code": ArazzoValidationIncomplete}, model.SeverityWarn, true)
	advisory := string(validation.CodeAdvisory)
	rules[advisory] = arazzoRule(advisory, "Use portable identifiers and avoid forward step references.", "$", "", "arazzoDocument", map[string]string{"code": advisory}, model.SeverityWarn, false)
	for _, check := range []struct{ id, given, field, severity string }{
		{"arazzo-info-description", "$", "info.description", model.SeverityWarn},
		{"arazzo-info-summary", "$", "info.summary", model.SeverityHint},
		{"arazzo-workflow-description", "$.workflows[*]", "description", model.SeverityWarn},
		{"arazzo-workflow-summary", "$.workflows[*]", "summary", model.SeverityHint},
		{"arazzo-step-description", "$.workflows[*].steps[*]", "description", model.SeverityWarn},
	} {
		rules[check.id] = arazzoRule(check.id, "Provide a non-empty "+check.field+".", check.given, check.field, "truthy", nil, check.severity, true)
	}
	rules["arazzo-no-script-tags-in-markdown"] = arazzoRule("arazzo-no-script-tags-in-markdown", "Descriptions and titles must not contain script tags.", "$..[description,title]", "", "pattern", map[string]string{"notMatch": "(?i)<script\\b"}, model.SeverityWarn, true)
	rules["arazzo-step-operationPath"] = arazzoRule("arazzo-step-operationPath", "Prefer operationId for portable operation references.", "$.workflows[*].steps[*]", "operationPath", "falsy", nil, model.SeverityHint, false)
	return rules
}

func arazzoRule(id, description, given, field, function string, options any, severity string, recommended bool) *model.Rule {
	return &model.Rule{Id: id, Name: id, Description: description, Message: "{{error}}",
		Given: given, Formats: model.ArazzoAllFormats, Severity: severity, Recommended: recommended,
		RuleCategory: model.RuleCategories[model.CategoryValidation], Type: "validation",
		Then: model.RuleAction{Field: field, Function: function, FunctionOptions: options}}
}
