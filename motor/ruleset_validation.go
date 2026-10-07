package motor

import (
	"fmt"
	"sort"

	"github.com/daveshanley/vacuum/functions"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
)

// UnknownFunctionError identifies a rule that cannot run because its function
// is neither built in nor registered as a custom function.
type UnknownFunctionError struct {
	RuleID   string
	Function string
}

func (e *UnknownFunctionError) Error() string {
	return fmt.Sprintf("rule %q uses unknown function %q", e.RuleID, e.Function)
}

// Validate before node selection: a selector with no matches must not hide an
// invalid function, and a rule's finding severity must not downgrade this error.
func validateRuleFunctions(rs *rulesets.RuleSet, builtin functions.Functions, custom map[string]model.RuleFunction) []error {
	if rs == nil {
		return nil
	}
	keys := make([]string, 0, len(rs.Rules))
	for key := range rs.Rules {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var errs []error
	for _, key := range keys {
		rule := rs.Rules[key]
		if rule == nil {
			continue
		}
		id := rule.Id
		if id == "" {
			id = key
		}
		forEachRuleAction(rule, func(action model.RuleAction) bool {
			if builtin.FindFunction(action.Function) == nil && custom[action.Function] == nil {
				errs = append(errs, &UnknownFunctionError{RuleID: id, Function: action.Function})
			}
			return false
		})
	}
	return errs
}
