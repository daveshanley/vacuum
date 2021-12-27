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

		var givenPaths []string
		if x, ok := rule.Given.(string); ok {
			givenPaths = append(givenPaths, x)
		}

		if x, ok := rule.Given.([]interface{}); ok {
			for _, gpI := range x {
				if gp, ok := gpI.(string); ok {
					givenPaths = append(givenPaths, gp)
				}
				if gp, ok := gpI.(int); ok {
					givenPaths = append(givenPaths, fmt.Sprintf("%v", gp))
				}
			}

		}

		for _, givenPath := range givenPaths {

			nodes, err := utils.FindNodes(spec, givenPath)
			if err != nil {
				return nil, err
			}
			if len(nodes) <= 0 {
				continue
			}

			var ruleAction model.RuleAction
			err = mapstructure.Decode(rule.Then, &ruleAction)

			if err == nil {

				ruleResults = buildResults(rule, builtinFunctions, ruleAction, ruleResults, nodes)

			} else {
				var ruleActions []model.RuleAction
				err = mapstructure.Decode(rule.Then, &ruleActions)

				if err == nil {
					for _, rAction := range ruleActions {
						ruleResults = buildResults(rule, builtinFunctions, rAction, ruleResults, nodes)
					}
				}
			}
		}

	}
	return ruleResults, nil
}

func buildResults(rule *model.Rule, builtinFunctions functions.Functions, ruleAction model.RuleAction,
	ruleResults []model.RuleFunctionResult, nodes []*yaml.Node) []model.RuleFunctionResult {

	ruleFunction := builtinFunctions.FindFunction(ruleAction.Function)

	if ruleFunction != nil {

		rfc := model.RuleFunctionContext{
			Options:    ruleAction.FunctionOptions,
			RuleAction: &ruleAction,
			Rule:       rule,
		}

		// validate the rule is configured correctly before running it.
		res, errs := model.ValidateRuleFunctionContextAgainstSchema(ruleFunction, rfc)
		if !res {
			for _, e := range errs {
				ruleResults = append(ruleResults, model.RuleFunctionResult{Message: e})
			}
		} else {

			// iterate through nodes and supply them one at a time so we don't pollute each run
			// TODO: change this signature to be singular and not an array so this is handled permanently.

			for _, node := range nodes {
				ruleResults = append(ruleResults, ruleFunction.RunRule([]*yaml.Node{node}, rfc)...)
			}

		}
	}
	return ruleResults
}
