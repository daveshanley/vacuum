package functions

import (
	"github.com/daveshanley/vacuum/functions/openapi"
	"github.com/daveshanley/vacuum/model"
	"sync"
)

const (
	warn  = "warn"
	error = "error"
	info  = "info"
	hint  = "hint"
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
	rules["oasOpSuccessResponse"] = &model.Rule{
		Description: "Operation must have at least one 2xx or a 3xx response.",
		Given:       openapi.GetAllOperationsJSONPath(),
		Resolved:    true,
		Recommended: true,
		Severity:    warn,
		Then: model.RuleAction{
			Field:    "responses",
			Function: "oasOpSuccessResponse",
		},
	}

	set := &model.RuleSet{
		DocumentationURI: "https://quobix.com/vaccum/rules/oasOpSuccessResponse",
		Rules:            rules,
	}

	return set

}
