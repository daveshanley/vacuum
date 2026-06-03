// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package rulesets

import (
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDefaultRuleSets_AsyncAPIRecommended(t *testing.T) {
	defaultRS := BuildDefaultRuleSets()
	recommended := defaultRS.GenerateAsyncAPIRecommendedRuleSet()

	require.NotNil(t, recommended)
	assert.Contains(t, recommended.Rules, AsyncAPI3DocumentResolved)
	assert.Contains(t, recommended.Rules, AsyncAPI3OperationSecurity)
	assert.Contains(t, recommended.Rules, AsyncAPIOperationChannel)
	assert.Contains(t, recommended.Rules, AsyncAPIUnusedComponents)
	assert.NotContains(t, recommended.Rules, AsyncAPI3ServerNotExampleCom)
	for _, rule := range recommended.Rules {
		assert.Contains(t, rule.Formats, model.AsyncAPI3)
	}
}

func TestGenerateRuleSetFromSuppliedRuleSet_AsyncAPIExtends(t *testing.T) {
	defaultRS := BuildDefaultRuleSets()
	ruleSet := defaultRS.GenerateRuleSetFromSuppliedRuleSet(&RuleSet{
		Extends: []interface{}{[]interface{}{VacuumAsyncAPI, VacuumRecommended}},
	})

	require.NotNil(t, ruleSet)
	assert.Contains(t, ruleSet.Rules, AsyncAPI3DocumentUnresolved)
	assert.Contains(t, ruleSet.Rules, AsyncAPIChannelParameters)
}

func TestGenerateRuleSetFromSuppliedRuleSet_AsyncAPIRecommendedStandaloneExtends(t *testing.T) {
	defaultRS := BuildDefaultRuleSets()
	ruleSet := defaultRS.GenerateRuleSetFromSuppliedRuleSet(&RuleSet{
		Extends: VacuumAsyncAPIRecommended,
	})

	require.NotNil(t, ruleSet)
	assert.Contains(t, ruleSet.Rules, AsyncAPI3DocumentResolved)
	assert.Contains(t, ruleSet.Rules, AsyncAPIChannelParameters)
	assert.NotContains(t, ruleSet.Rules, AsyncAPI3ServerNotExampleCom)
}

func TestAsyncAPIDocumentRulesPreserveResolvedMode(t *testing.T) {
	rules := GetAsyncAPIRecommendedRules()

	require.NotNil(t, rules[AsyncAPI3DocumentResolved])
	require.NotNil(t, rules[AsyncAPI3DocumentUnresolved])
	assert.True(t, rules[AsyncAPI3DocumentResolved].Resolved)
	assert.False(t, rules[AsyncAPI3DocumentUnresolved].Resolved)
}

func TestAsyncAPICustomRulesUseBatchMode(t *testing.T) {
	for ruleID, rule := range GetAsyncAPIRecommendedRules() {
		for _, action := range asyncAPIRuleActions(rule.Then) {
			if !strings.HasPrefix(action.Function, "asyncApi") {
				continue
			}
			options, ok := action.FunctionOptions.(map[string]interface{})
			require.True(t, ok, "rule %s should use map options", ruleID)
			batch, ok := options["batch"].(bool)
			require.True(t, ok, "rule %s should set a boolean batch option", ruleID)
			assert.True(t, batch, "rule %s should run in batch mode", ruleID)
		}
	}
}

func asyncAPIRuleActions(then any) []model.RuleAction {
	switch typed := then.(type) {
	case model.RuleAction:
		return []model.RuleAction{typed}
	case []model.RuleAction:
		return typed
	default:
		return nil
	}
}
