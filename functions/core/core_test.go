package core

import "github.com/daveshanley/vacuum/model"

func buildCoreTestRule(given, severity, function, field string, functionOptions map[string]any) model.Rule {
	return model.Rule{
		Given:       given,
		Severity:    severity,
		Description: "test rule",
		Then: &model.RuleAction{
			Field:           field,
			Function:        function,
			FunctionOptions: functionOptions,
		},
	}
}

func buildCoreTestContext(action *model.RuleAction, options map[string]any) model.RuleFunctionContext {
	return model.RuleFunctionContext{
		RuleAction: action,
		Options:    options,
	}
}

func buildCoreTestContextFromRule(action *model.RuleAction, rule model.Rule) model.RuleFunctionContext {
	ruleAction := model.CastToRuleAction(rule.Then)
	return model.RuleFunctionContext{
		Rule:       &rule,
		RuleAction: action,
		Options:    ruleAction.FunctionOptions,
	}
}
