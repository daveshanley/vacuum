package motor

import (
	"errors"
	"fmt"
	"github.com/daveshanley/vacuum/functions"
	"github.com/daveshanley/vacuum/model"
	"github.com/mitchellh/mapstructure"
)

// RuleComposer will consume a ruleset specification into a *model.RuleSet
type RuleComposer struct {
}

func CreateRuleComposer() *RuleComposer {
	return &RuleComposer{}
}

// ComposeRuleSet compose a byte array ruleset specification into a *model.RuleSet
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

		if v.Then != nil {

			var ruleAction model.RuleAction
			err = mapstructure.Decode(v.Then, &ruleAction)

			if err == nil {

				if ruleAction.Function != "" {

					f := builtinFunctions.FindFunction(ruleAction.Function)
					if f == nil {
						return nil, fmt.Errorf("unable to locate function '%s' for rule '%s",
							ruleAction.Function, k)
					}
				}
			}

			// must be an array of then rule actions.
			var ruleActions []model.RuleAction
			err = mapstructure.Decode(v.Then, &ruleActions)

			if err == nil {

				for _, rAction := range ruleActions {
					if rAction.Function != "" {

						f := builtinFunctions.FindFunction(rAction.Function)
						if f == nil {
							return nil, fmt.Errorf("unable to locate function '%s' for rule '%s",
								rAction.Function, k)
						}
					}
				}
			}
		}
	}

	return rs, nil
}
