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
