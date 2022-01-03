package rulesets

import (
	"github.com/daveshanley/vacuum/model"
	"sync"
)

const (
	warn       = "warn"
	error      = "error"
	info       = "info"
	hint       = "hint"
	style      = "style"
	validation = "validation"
)

type ruleSetsModel struct {
	openAPIRuleSet *model.RuleSet
}

// RuleSets is used to generate default RuleSets built into vacuum
type RuleSets interface {

	// GenerateOpenAPIDefaultRuleSet generates a ready to run pointer to a model.RuleSet containing all
	// OpenAPI rules supported by vacuum. Passing all these rules would be considered a good quality specification.
	GenerateOpenAPIDefaultRuleSet() *model.RuleSet
}

var rulesetsSingleton *ruleSetsModel
var openAPIRulesGrab sync.Once

func BuildDefaultRuleSets() RuleSets {
	openAPIRulesGrab.Do(func() {
		rulesetsSingleton = &ruleSetsModel{
			openAPIRuleSet: generateDefaultOpenAPIRuleSet(),
		}
	})

	return rulesetsSingleton
}

func (rsm ruleSetsModel) GenerateOpenAPIDefaultRuleSet() *model.RuleSet {
	return rsm.openAPIRuleSet
}

func generateDefaultOpenAPIRuleSet() *model.RuleSet {

	rules := make(map[string]*model.Rule)

	// add success response
	rules["operation-success-response"] = &model.Rule{
		Description: "Operation must have at least one 2xx or a 3xx response.",
		Given:       "$",
		Resolved:    true,
		Recommended: true,
		Type:        style,
		Severity:    warn,
		Then: model.RuleAction{
			Field:    "responses",
			Function: "oasOpSuccessResponse",
		},
	}

	// add unique operation ID rule
	rules["operation-operationId-unique"] = &model.Rule{
		Description: "Every operation must have unique \"operationId\".",
		Given:       "$.paths",
		Resolved:    true,
		Recommended: true,
		Type:        validation,
		Severity:    warn,
		Then: model.RuleAction{
			Function: "oasOpIdUnique",
		},
	}

	// add operation params rule
	rules["operation-parameters"] = &model.Rule{
		Description: "Operation parameters are unique and non-repeating.",
		Given:       "$.paths",
		Resolved:    true,
		Recommended: true,
		Type:        validation,
		Severity:    warn,
		Then: model.RuleAction{
			Function: "oasOpParams",
		},
	}

	// add operation tag defined rule
	rules["operation-tag-defined"] = &model.Rule{
		Description: "Operation tags must be defined in global tags.",
		Given:       "$",
		Resolved:    true,
		Recommended: true,
		Type:        validation,
		Severity:    warn,
		Then: model.RuleAction{
			Function: "oasTagDefined",
		},
	}

	// add operation tag defined rule
	rules["path-params"] = &model.Rule{
		Description: "Path parameters must be defined and valid.",
		Given:       "$",
		Resolved:    true,
		Recommended: true,
		Type:        validation,
		Severity:    error,
		Then: model.RuleAction{
			Function: "oasPathParam",
		},
	}

	set := &model.RuleSet{
		DocumentationURI: "https://quobix.com/vacuum/rules/openapi",
		Rules:            rules,
	}

	return set

}
