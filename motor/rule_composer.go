package motor

import (
	"errors"

	"github.com/daveshanley/vacuum/rulesets"
)

// RuleComposer will consume a ruleset specification into a *model.RuleSet
type RuleComposer struct {
}

// CreateRuleComposer will create a new RuleComposer and return a pointer to it.
func CreateRuleComposer() *RuleComposer {
	return &RuleComposer{}
}

// ComposeRuleSet compose a byte array ruleset specification into a *model.RuleSet
func (rc *RuleComposer) ComposeRuleSet(ruleset []byte) (*rulesets.RuleSet, error) {
	rs, err := rulesets.CreateRuleSetFromData(ruleset)
	if err != nil {
		return nil, err
	}

	// check for rules length
	if len(rs.Rules) <= 0 {
		return nil, errors.New("no rules defined in ruleset, cannot continue")
	}

	for k, v := range rs.Rules {
		if v.Id == "" {
			v.Id = k
		}
	}
	if functionErrors := ValidateRuleSetFunctions(rs, nil); len(functionErrors) > 0 {
		return nil, errors.Join(functionErrors...)
	}

	return rs, nil
}
