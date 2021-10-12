package motor

import (
	"fmt"
	"github.com/daveshanley/vaccum/functions"
	"github.com/daveshanley/vaccum/model"
	"github.com/daveshanley/vaccum/utils"
)

func ApplyRules(ruleSet *model.RuleSet, spec []byte) ([]model.RuleFunctionResult, error) {

	builtinFunctions := functions.MapBuiltinFunctions()
	var ruleResults []model.RuleFunctionResult

	for _, rule := range ruleSet.Rules {

		nodes, err := utils.FindNodes(spec, rule.Given)
		if err != nil {
			return nil, err
		}
		if len(nodes) <= 0 {
			return nil, fmt.Errorf("no nodes found matching path: '%s'", rule.Given)
		}

		ruleFunction := builtinFunctions.FindFunction(rule.Then.FunctionName)
		if ruleFunction != nil {

			rfc := model.RuleFunctionContext{
				Options:    rule.Then.FunctionOptions,
				RuleAction: rule.Then,
			}

			ruleResults = append(ruleResults, ruleFunction.RunRule(nodes, rfc)...)
		}

	}
	return ruleResults, nil
}
