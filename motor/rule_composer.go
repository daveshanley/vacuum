package motor

import (
	"errors"
	"fmt"
	"github.com/daveshanley/vaccum/functions"
	"github.com/daveshanley/vaccum/model"
)

type RuleComposer struct {
}

func CreateRuleComposer() *RuleComposer {
	return &RuleComposer{}
}

func (rc *RuleComposer) ComposeRuleSet(ruleset []byte) (*model.RuleSet, error) {
	rs, err := model.CreateRuleSetUsingJSON(ruleset)
	if err != nil {
		return nil, err
	}

	// check for rules length
	if len(rs.Rules) <= 0 {
		return nil, errors.New("no rules defined in ruleset, cannot continue")
	}

	// load builtinFunctions
	builtinFunctions := functions.MapBuiltinFunctions()

	// check builtinFunctions exist for rules defined
	for k, v := range rs.Rules {
		if v.Then != nil && v.Then.FunctionName != "" {
			f := builtinFunctions.FindFunction(v.Then.FunctionName)
			if f == nil {
				return nil, fmt.Errorf("unable to locate function '%s' for rule '%s", v.Then.FunctionName, k)
			}
		}
	}

	return rs, nil
}
