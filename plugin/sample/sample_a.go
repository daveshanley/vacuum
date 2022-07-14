package main

import (
	"github.com/daveshanley/vacuum/model"
	"gopkg.in/yaml.v3"
)

// SampleRuleFunction_A is an example custom rule that does nothing.
type SampleRuleFunction_A struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the Defined rule.
func (s SampleRuleFunction_A) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "uselessFunc",
	}
}

// RunRule will execute the Sample rule, based on supplied context and a supplied []*yaml.Node slice.
func (s SampleRuleFunction_A) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	// return a single result, for a made up linting failure.
	return []model.RuleFunctionResult{
		{
			Message:   "this is a useless function that will always error out.",
			StartNode: &yaml.Node{Line: 1, Column: 0},
			EndNode:   &yaml.Node{Line: 2, Column: 0},
			Path:      "$.i.do.not.exist",
			Rule:      context.Rule,
		},
	}
}
