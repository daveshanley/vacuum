// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package motor

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/assert"
)

func TestApplicableRulesForFormatSkipsUnscopedRulesForAsyncAPI(t *testing.T) {
	ruleSet := &rulesets.RuleSet{
		Rules: map[string]*model.Rule{
			"unscoped": {Id: "unscoped"},
			"async":    {Id: "async", Formats: model.AsyncAPI3AllFormats},
			"oas":      {Id: "oas", Formats: model.OAS3AllFormat},
		},
	}

	assert.ElementsMatch(t, []string{"async"}, ruleIDs(applicableRulesForFormat(ruleSet, model.AsyncAPI31)))
}

func TestApplicableRulesForFormatUsesRuleSetFormatsForUnscopedRules(t *testing.T) {
	ruleSet := &rulesets.RuleSet{
		Formats: model.AsyncAPI3AllFormats,
		Rules: map[string]*model.Rule{
			"unscoped": {Id: "unscoped"},
		},
	}

	assert.Empty(t, applicableRulesForFormat(ruleSet, model.OAS31))
	assert.ElementsMatch(t, []string{"unscoped"}, ruleIDs(applicableRulesForFormat(ruleSet, model.AsyncAPI31)))
}

func TestApplicableRulesForFormatKeepsLegacyUnscopedRulesForOpenAPI(t *testing.T) {
	ruleSet := &rulesets.RuleSet{
		Rules: map[string]*model.Rule{
			"unscoped": {Id: "unscoped"},
		},
	}

	assert.ElementsMatch(t, []string{"unscoped"}, ruleIDs(applicableRulesForFormat(ruleSet, model.OAS31)))
}

func ruleIDs(rules []*model.Rule) []string {
	ids := make([]string, 0, len(rules))
	for _, rule := range rules {
		ids = append(ids, rule.Id)
	}
	return ids
}
