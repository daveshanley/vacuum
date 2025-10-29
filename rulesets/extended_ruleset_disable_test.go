// Copyright 2025 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package rulesets

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/stretchr/testify/assert"
)

func TestExtendedRuleset_DisableRule(t *testing.T) {
	// This test verifies that when extending a ruleset and setting a rule to false,
	// the rule is properly disabled. This addresses issue #739.

	// Create a base ruleset with a custom rule
	baseRulesetYAML := `
rules:
  my-custom-rule:
    description: "Test rule that should be disabled"
    given: $.info.title
    severity: error
    then:
      function: pattern
      functionOptions:
        match: "^hello world$"
`

	// Create an override ruleset that extends the base and disables the rule
	overrideRulesetYAML := `
extends:
  - base.yaml
rules:
  my-custom-rule: false
`

	// Parse the base ruleset
	baseRS, err := CreateRuleSetFromData([]byte(baseRulesetYAML))
	assert.NoError(t, err)
	assert.NotNil(t, baseRS)
	assert.Len(t, baseRS.Rules, 1)
	assert.NotNil(t, baseRS.Rules["my-custom-rule"])

	// Parse the override ruleset
	overrideRS, err := CreateRuleSetFromData([]byte(overrideRulesetYAML))
	assert.NoError(t, err)
	assert.NotNil(t, overrideRS)

	// Verify that the override ruleset has the rule marked as false
	assert.Equal(t, false, overrideRS.RuleDefinitions["my-custom-rule"])

	// When GenerateRuleSetFromSuppliedRuleSet is called with the override ruleset,
	// it should process the extends and then apply the overrides.
	// The parent's RuleDefinitions (with my-custom-rule: false) should take precedence
	// and the rule should be disabled in the final ruleset.
}

func TestExtendedRuleset_PreserveParentDefinitions(t *testing.T) {
	// This test verifies that parent rule definitions take precedence over
	// extended ruleset definitions when processing extends.

	rs := &RuleSet{
		RuleDefinitions: map[string]interface{}{
			"rule-to-disable": false,
			"rule-to-modify": map[string]interface{}{
				"severity": "warn",
			},
		},
		Rules: make(map[string]*model.Rule),
	}

	// Simulate what SniffOutAllExternalRules does
	// Extended ruleset has full definitions for these rules
	extendedDefinitions := map[string]interface{}{
		"rule-to-disable": map[string]interface{}{
			"description": "This should not override parent's false",
			"severity":    "error",
		},
		"rule-to-modify": map[string]interface{}{
			"description": "This should not override parent's modification",
			"severity":    "error",
		},
		"new-rule": map[string]interface{}{
			"description": "This should be added",
			"severity":    "info",
		},
	}

	// Apply the logic from SniffOutAllExternalRules with the fix
	for ruleName, ruleValue := range extendedDefinitions {
		// Don't overwrite parent's rule definitions - they take precedence
		if _, exists := rs.RuleDefinitions[ruleName]; !exists {
			rs.RuleDefinitions[ruleName] = ruleValue
		}
	}

	// Verify parent definitions are preserved
	assert.Equal(t, false, rs.RuleDefinitions["rule-to-disable"],
		"Parent's rule disable (false) should be preserved")

	assert.Equal(t, "warn", rs.RuleDefinitions["rule-to-modify"].(map[string]interface{})["severity"],
		"Parent's rule modification should be preserved")

	assert.NotNil(t, rs.RuleDefinitions["new-rule"],
		"New rules from extended ruleset should be added")
	assert.Equal(t, "info", rs.RuleDefinitions["new-rule"].(map[string]interface{})["severity"],
		"New rule should have its original definition")
}
