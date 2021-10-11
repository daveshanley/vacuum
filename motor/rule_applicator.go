package motor

import (
	"fmt"
	"github.com/daveshanley/vaccum/functions"
	"github.com/daveshanley/vaccum/model"
	"github.com/daveshanley/vaccum/utils"
)

func ApplyRules(ruleSet *model.RuleSet, spec []byte) ([]model.RuleFunctionResult, error) {

	builtinFunctions := functions.MapBuiltinFunctions()

	//// decode using YAML as it handles JSON well, and we're going to need the AST nodes.
	//var yamlData map[string]interface{}
	//yaml.Unmarshal(spec, &yamlData)

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
			ruleResults = append(ruleResults, ruleFunction.RunRule(nodes, rule.Then.FunctionOptions, nil)...)
		}

	}
	return ruleResults, nil
}
