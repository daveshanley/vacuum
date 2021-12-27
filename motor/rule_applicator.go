package motor

import (
	"fmt"
	"github.com/daveshanley/vaccum/functions"
	"github.com/daveshanley/vaccum/model"
	"github.com/daveshanley/vaccum/utils"
	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v3"
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

		var ruleAction model.RuleAction
		err = mapstructure.Decode(rule.Then, &ruleAction)

		if err == nil {

			ruleResults = buildResults(builtinFunctions, ruleAction, ruleResults, nodes)

		} else {
			var ruleActions []model.RuleAction
			err = mapstructure.Decode(rule.Then, &ruleActions)

			if err == nil {
				for _, rAction := range ruleActions {
					ruleResults = buildResults(builtinFunctions, rAction, ruleResults, nodes)
				}
			}
		}

	}
	return ruleResults, nil
}

func buildResults(builtinFunctions functions.Functions, ruleAction model.RuleAction,
	ruleResults []model.RuleFunctionResult, nodes []*yaml.Node) []model.RuleFunctionResult {
	ruleFunction := builtinFunctions.FindFunction(ruleAction.Function)

	if ruleFunction != nil {

		rfc := model.RuleFunctionContext{
			Options:    ruleAction.FunctionOptions,
			RuleAction: &ruleAction,
		}

		// validate the rule is configured correctly before running it.
		res, errs := model.ValidateRuleFunctionContextAgainstSchema(ruleFunction, rfc)
		if !res {
			for _, e := range errs {
				ruleResults = append(ruleResults, model.RuleFunctionResult{Message: e})
			}
		} else {
			ruleResults = append(ruleResults, ruleFunction.RunRule(nodes, rfc)...)
		}
	}
	return ruleResults
}
