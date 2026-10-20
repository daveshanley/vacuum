package motor

import (
	"fmt"
	"sort"
	"strings"

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

// ValidateRuleSet checks function names and options before fixture selection.
// Harnesses use this stricter preflight so an invalid action cannot pass merely
// because its selector matched no nodes. Normal lint behavior is unchanged.
func ValidateRuleSet(rs *rulesets.RuleSet, custom map[string]model.RuleFunction) []error {
	builtin := functions.MapBuiltinFunctions()
	errs := validateRuleFunctions(rs, builtin, custom)
	if rs == nil {
		return errs
	}
	keys := make([]string, 0, len(rs.Rules))
	for key := range rs.Rules {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		rule := rs.Rules[key]
		if rule == nil {
			continue
		}

		paths := resultGivenPaths(rule)
		if len(paths) == 0 {
			errs = append(errs, fmt.Errorf("rule %q requires at least one selector", key))
		}
		for _, path := range paths {
			if strings.TrimSpace(path) == "" {
				errs = append(errs, fmt.Errorf("rule %q has an empty selector", key))
			}
		}
		if values, ok := rule.Given.([]interface{}); ok && len(values) != len(paths) {
			errs = append(errs, fmt.Errorf("rule %q selectors must be strings", key))
		}
		actions := 0
		forEachRuleAction(rule, func(action model.RuleAction) bool {
			actions++
			f := builtin.FindFunction(action.Function)
			if f == nil {
				f = custom[action.Function]
			}
			if f == nil {
				return false
			}
			valid, issues := model.ValidateRuleFunctionContextAgainstSchema(f, model.RuleFunctionContext{Rule: rule, RuleAction: &action, Options: action.FunctionOptions})
			if !valid || len(issues) > 0 {
				errs = append(errs, fmt.Errorf("rule %q action %q: %v", key, action.Function, issues))
			}
			return false
		})
		if actions == 0 {
			errs = append(errs, fmt.Errorf("rule %q requires at least one action", key))
		}
	}
	return errs
}
