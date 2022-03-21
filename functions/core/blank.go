package core

import (
	"github.com/daveshanley/vacuum/model"
	"gopkg.in/yaml.v3"
)

// Blank is a pass through function that does nothing. Use this if you want a rig a rule, but don't want to check
// any logic, the logic is pre or post processed outside the main rule run.
type Blank struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the Blank rule.
func (b Blank) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "blank",
	}
}

// RunRule will execute the Blank rule, based on supplied context and a supplied []*yaml.Node slice.
func (b Blank) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	// return right away, nothing to do in here.
	return nil
}
