// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package rulesets

import "github.com/daveshanley/vacuum/model"

// GetContactPropertiesRule will return a rule configured to look at contact properties of a spec.
// it uses the in-built 'truthy' function
func GetContactPropertiesRule() *model.Rule {
	return &model.Rule{
		Description: "Contact details are incomplete",
		Given:       "$.info.contact",
		Resolved:    true,
		Recommended: true,
		Type:        validation,
		Severity:    error,
		Then: []model.RuleAction{
			{
				Field:    "name",
				Function: "truthy",
			},
			{
				Field:    "url",
				Function: "truthy",
			},
			{
				Field:    "email",
				Function: "truthy",
			},
		},
	}
}

// GetInfoContactRule Will return a rule that uses the truthy function to check if the
// info object contains a contact object
func GetInfoContactRule() *model.Rule {
	return &model.Rule{
		Description: "Info section is missing contact details",
		Given:       "$.info",
		Resolved:    true,
		Recommended: true,
		Type:        validation,
		Severity:    error,
		Then: model.RuleAction{
			Field:    "contact",
			Function: "truthy",
		},
	}
}

// GetInfoDescriptionRule Will return a rule that uses the truthy function to check if the
// info object contains a description
func GetInfoDescriptionRule() *model.Rule {
	return &model.Rule{
		Description: "Info section is missing a description",
		Given:       "$.info",
		Resolved:    true,
		Recommended: true,
		Type:        validation,
		Severity:    error,
		Then: model.RuleAction{
			Field:    "description",
			Function: "truthy",
		},
	}
}

// GetInfoLicenseRule will return a rule that uses the truthy function to check if the
// info object contains a license
func GetInfoLicenseRule() *model.Rule {
	return &model.Rule{
		Description: "Info section should contain a license",
		Given:       "$.info",
		Resolved:    true,
		Recommended: true,
		Type:        validation,
		Severity:    error,
		Then: model.RuleAction{
			Field:    "license",
			Function: "truthy",
		},
	}
}

// GetInfoLicenseUrlRule will return a rule that uses the truthy function to check if the
// info object contains a license with an url that is set.
func GetInfoLicenseUrlRule() *model.Rule {
	return &model.Rule{
		Description: "License should contain an url",
		Given:       "$.info.license",
		Resolved:    true,
		Recommended: true,
		Type:        validation,
		Severity:    error,
		Then: model.RuleAction{
			Field:    "url",
			Function: "truthy",
		},
	}
}

// GetNoEvalInMarkdownRule will return a rule that uses the pattern function to check if
// there is no eval statements markdown used in descriptions
func GetNoEvalInMarkdownRule() *model.Rule {

	fo := make(map[string]string)
	fo["notMatch"] = "eval\\("

	return &model.Rule{
		Description: "Markdown descriptions must not have 'eval('",
		Given:       "$..description",
		Resolved:    true,
		Recommended: true,
		Type:        validation,
		Severity:    error,
		Then: model.RuleAction{
			Function:        "pattern",
			FunctionOptions: fo,
		},
	}
}

// GetNoScriptTagsInMarkdown will return a rule that uses the pattern function to check if
// there is no script tags used in descriptions and the title.
func GetNoScriptTagsInMarkdown() *model.Rule {

	fo := make(map[string]string)
	fo["notMatch"] = "<script"

	return &model.Rule{
		Description: "Markdown descriptions must not contain '<script>' tags",
		Given:       "$..description",
		Resolved:    true,
		Recommended: true,
		Type:        validation,
		Severity:    error,
		Then: model.RuleAction{
			Function:        "pattern",
			FunctionOptions: fo,
		},
	}
}

// GetOpenApiTagsAlphabetical will return a rule that uses the alphabetical function to check if
// tags are in alphabetical order
func GetOpenApiTagsAlphabetical() *model.Rule {

	fo := make(map[string]string)
	fo["keyedBy"] = "name"

	return &model.Rule{
		Description: "Tags must be in alphabetical order",
		Given:       "$.tags",
		Resolved:    true,
		Recommended: true,
		Type:        validation,
		Severity:    error,
		Then: model.RuleAction{
			Function:        "alphabetical",
			FunctionOptions: fo,
		},
	}
}
