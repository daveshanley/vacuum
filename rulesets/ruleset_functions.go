package rulesets

import "github.com/daveshanley/vacuum/model"

// GetContactPropertiesRule will return a rule configured to look at contact properties of a spec.
// it uses the in-built 'truthy' function
func GetContactPropertiesRule() *model.Rule {
	return &model.Rule{
		Description: "Contact object must be complete'",
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
