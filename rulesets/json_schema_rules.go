// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package rulesets

import "github.com/daveshanley/vacuum/model"

func GenerateDefaultJSONSchemaRuleSet() *RuleSet {
	return &RuleSet{
		DocumentationURI: "https://quobix.com/vacuum/rulesets/json-schema-recommended",
		Formats:          model.JSONSchemaAllFormats,
		Description:      "Recommended rules for high quality JSON Schema documents.",
		Rules:            GetJSONSchemaRecommendedRules(),
		Extends:          map[string]string{VacuumJSONSchema: VacuumRecommended},
	}
}

func GetJSONSchemaRecommendedRules() map[string]*model.Rule {
	return map[string]*model.Rule{
		JsonSchemaValid:                     GetJSONSchemaValidRule(),
		JsonSchemaRefValid:                  GetJSONSchemaRefValidRule(),
		JsonSchemaTypeConstraintCompatible:  GetJSONSchemaTypeConstraintRule(),
		JsonSchemaEnumValuesCompatible:      GetJSONSchemaEnumValuesRule(),
		JsonSchemaRequiredPropertiesDefined: GetJSONSchemaRequiredPropertiesRule(),
		JsonSchemaDependentRequiredDefined:  GetJSONSchemaDependentRequiredRule(),
		JsonSchemaPatternsValid:             GetJSONSchemaPatternsRule(),
		JsonSchemaCompositionSanity:         GetJSONSchemaCompositionRule(),
		JsonSchemaTitleDescriptionType:      GetJSONSchemaQualityRule(),
		JsonSchemaExamplesValid:             GetJSONSchemaExamplesRule(),
	}
}

func GetJSONSchemaValidRule() *model.Rule {
	return jsonSchemaRule(JsonSchemaValid, "Validate JSON Schema against its metaschema", "JSON Schema must be structurally valid for its dialect.", model.SeverityError, "jsonSchemaValid", nil)
}

func GetJSONSchemaRefValidRule() *model.Rule {
	return jsonSchemaRule(JsonSchemaRefValid, "Check JSON Schema references", "$ref values must resolve successfully.", model.SeverityError, "jsonSchemaRefValid", nil)
}

func GetJSONSchemaTypeConstraintRule() *model.Rule {
	return jsonSchemaRule(JsonSchemaTypeConstraintCompatible, "Check type and constraint compatibility", "JSON Schema type-specific constraints must match the declared type.", model.SeverityError, "jsonSchemaSanity", map[string]string{"check": "type"})
}

func GetJSONSchemaEnumValuesRule() *model.Rule {
	return jsonSchemaRule(JsonSchemaEnumValuesCompatible, "Check enum and const compatibility", "enum and const values must be compatible with schema type and each other.", model.SeverityError, "jsonSchemaSanity", map[string]string{"check": "enumConst"})
}

func GetJSONSchemaRequiredPropertiesRule() *model.Rule {
	return jsonSchemaRule(JsonSchemaRequiredPropertiesDefined, "Check required properties", "required values must be unique and refer to defined properties.", model.SeverityError, "jsonSchemaSanity", map[string]string{"check": "required"})
}

func GetJSONSchemaDependentRequiredRule() *model.Rule {
	return jsonSchemaRule(JsonSchemaDependentRequiredDefined, "Check dependent required properties", "dependentRequired values must refer to defined properties.", model.SeverityError, "jsonSchemaSanity", map[string]string{"check": "dependent"})
}

func GetJSONSchemaPatternsRule() *model.Rule {
	return jsonSchemaRule(JsonSchemaPatternsValid, "Check regular expression patterns", "JSON Schema patterns must be valid regular expressions.", model.SeverityError, "jsonSchemaSanity", map[string]string{"check": "patterns"})
}

func GetJSONSchemaCompositionRule() *model.Rule {
	return jsonSchemaRule(JsonSchemaCompositionSanity, "Check shallow composition contradictions", "Composition keywords must not contain obvious shallow contradictions.", model.SeverityError, "jsonSchemaSanity", map[string]string{"check": "composition"})
}

func GetJSONSchemaQualityRule() *model.Rule {
	return jsonSchemaRule(JsonSchemaTitleDescriptionType, "Check JSON Schema quality annotations", "Schemas should define title, description and type, and constrain object/array shapes.", model.SeverityWarn, "jsonSchemaSanity", map[string]string{"check": "quality"})
}

func GetJSONSchemaExamplesRule() *model.Rule {
	return jsonSchemaRule(JsonSchemaExamplesValid, "Check examples and defaults", "default and examples values should validate against their schema.", model.SeverityWarn, "jsonSchemaSanity", map[string]string{"check": "examples"})
}

func jsonSchemaRule(id, name, description, severity, function string, options map[string]string) *model.Rule {
	return &model.Rule{
		Name:         name,
		Id:           id,
		Formats:      model.JSONSchemaAllFormats,
		Description:  description,
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategorySchemas],
		Type:         Validation,
		Severity:     severity,
		Then: model.RuleAction{
			Function:        function,
			FunctionOptions: options,
		},
	}
}
