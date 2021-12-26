package core

import "github.com/daveshanley/vaccum/model"

const (
	severityError = "error"
	severityWarn  = "warn"
)

func buildCoreTestRule(given, severity, function, field string, functionOptions map[string]string) model.Rule {
	return model.Rule{
		Given:    given,
		Severity: severity,
		Then: &model.RuleAction{
			Field:           field,
			FunctionName:    function,
			FunctionOptions: functionOptions,
		},
	}
}

func buildCoreTestContext(action *model.RuleAction, options map[string]string) model.RuleFunctionContext {
	return model.RuleFunctionContext{
		RuleAction: action,
		Options:    options,
	}
}

func buildCoreTestContextFromRule(action *model.RuleAction, rule model.Rule) model.RuleFunctionContext {
	return model.RuleFunctionContext{
		RuleAction: action,
		Options:    rule.Then.FunctionOptions,
	}
}
