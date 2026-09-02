// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package motor

import (
	"fmt"
	"sort"

	"github.com/daveshanley/vacuum/functions"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
)

// UnknownRuleFunctionError reports a function referenced by a rule that is not
// available as either a built-in or a supplied custom function.
type UnknownRuleFunctionError struct {
	RuleID   string
	Function string
}

// Error returns a human-readable ruleset configuration error.
func (e *UnknownRuleFunctionError) Error() string {
	if e == nil {
		return "unknown rule function"
	}
	return fmt.Sprintf("unknown function '%s' in rule '%s'", e.Function, e.RuleID)
}

// ValidateRuleSetFunctions verifies that every rule action references an
// available built-in or supplied custom function. Errors are returned in
// ruleset map-key order so callers receive deterministic diagnostics.
func ValidateRuleSetFunctions(
	ruleSet *rulesets.RuleSet,
	customFunctions map[string]model.RuleFunction,
) []error {
	return validateRuleSetFunctions(ruleSet, functions.MapBuiltinFunctions(), customFunctions)
}

func validateRuleSetFunctions(
	ruleSet *rulesets.RuleSet,
	builtinFunctions functions.Functions,
	customFunctions map[string]model.RuleFunction,
) []error {
	if ruleSet == nil || len(ruleSet.Rules) == 0 {
		return nil
	}

	ruleIDs := make([]string, 0, len(ruleSet.Rules))
	for ruleID := range ruleSet.Rules {
		ruleIDs = append(ruleIDs, ruleID)
	}
	sort.Strings(ruleIDs)

	var validationErrors []error
	for _, mapRuleID := range ruleIDs {
		rule := ruleSet.Rules[mapRuleID]
		if rule == nil {
			continue
		}
		ruleID := rule.Id
		if ruleID == "" {
			ruleID = mapRuleID
		}

		unknownFunctions := make(map[string]struct{})
		forEachRuleAction(rule, func(action model.RuleAction) bool {
			if builtinFunctions.FindFunction(action.Function) != nil || customFunctions[action.Function] != nil {
				return false
			}
			if _, reported := unknownFunctions[action.Function]; reported {
				return false
			}
			unknownFunctions[action.Function] = struct{}{}
			validationErrors = append(validationErrors, &UnknownRuleFunctionError{
				RuleID:   ruleID,
				Function: action.Function,
			})
			return false
		})
	}
	return validationErrors
}
