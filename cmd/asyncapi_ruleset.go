// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package cmd

import (
	arazzo_context "github.com/daveshanley/vacuum/arazzo"
	asyncapi_context "github.com/daveshanley/vacuum/asyncapi"
	"github.com/daveshanley/vacuum/rulesets"
)

func selectDefaultRuleSetForSpec(defaultRuleSets rulesets.RuleSets, specBytes []byte, hardMode bool) (*rulesets.RuleSet, string, bool) {
	if format, _ := arazzo_context.DetectFormat(specBytes); format != "" {
		if hardMode {
			return defaultRuleSets.GenerateArazzoDefaultRuleSet(), format, true
		}
		return defaultRuleSets.GenerateArazzoRecommendedRuleSet(), format, true
	}
	if format, err := asyncapi_context.DetectFormat(specBytes); err == nil && format != "" {
		if hardMode {
			return defaultRuleSets.GenerateAsyncAPIDefaultRuleSet(), format, true
		}
		return defaultRuleSets.GenerateAsyncAPIRecommendedRuleSet(), format, true
	}
	if hardMode {
		return defaultRuleSets.GenerateOpenAPIDefaultRuleSet(), "", false
	}
	return defaultRuleSets.GenerateOpenAPIRecommendedRuleSet(), "", false
}

func prepareDefaultRuleSetForSpec(defaultRuleSets rulesets.RuleSets, specBytes []byte, hardMode, turbo bool) (*rulesets.RuleSet, string, bool) {
	selectedRS, specFormat, nonOpenAPIDefault := selectDefaultRuleSetForSpec(defaultRuleSets, specBytes, hardMode)
	if hardMode && !nonOpenAPIDefault {
		MergeOWASPRulesToRuleSet(selectedRS, true)
	}
	if turbo {
		rulesets.FilterRulesForTurbo(selectedRS)
	}
	return selectedRS, specFormat, nonOpenAPIDefault
}
