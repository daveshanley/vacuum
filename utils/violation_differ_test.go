// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package utils

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeResult(ruleId, path, message string) model.RuleFunctionResult {
	return model.RuleFunctionResult{
		RuleId:  ruleId,
		Path:    path,
		Message: message,
		Rule:    &model.Rule{Id: ruleId},
	}
}

func makeResultPtr(ruleId, path, message string) *model.RuleFunctionResult {
	r := makeResult(ruleId, path, message)
	return &r
}

func TestDiffViolationsValues_BothEmpty(t *testing.T) {
	result, stats := DiffViolationsValues(nil, nil)
	assert.Empty(t, result)
	assert.Equal(t, 0, stats.TotalResultsBefore)
	assert.Equal(t, 0, stats.TotalResultsAfter)
	assert.Equal(t, 0, stats.ResultsDropped)
}

func TestDiffViolationsValues_OriginalEmpty(t *testing.T) {
	newResults := []model.RuleFunctionResult{
		makeResult("rule-1", "$.paths./items.post", "missing description"),
		makeResult("rule-2", "$.info", "missing contact"),
	}
	result, stats := DiffViolationsValues(nil, newResults)
	assert.Len(t, result, 2)
	assert.Equal(t, 2, stats.TotalResultsBefore)
	assert.Equal(t, 2, stats.TotalResultsAfter)
	assert.Equal(t, 0, stats.ResultsDropped)
}

func TestDiffViolationsValues_NewEmpty(t *testing.T) {
	original := []model.RuleFunctionResult{
		makeResult("rule-1", "$.paths./items.post", "missing description"),
	}
	result, stats := DiffViolationsValues(original, nil)
	assert.Empty(t, result)
	assert.Equal(t, 0, stats.TotalResultsBefore)
	assert.Equal(t, 0, stats.TotalResultsAfter)
	assert.Equal(t, 0, stats.ResultsDropped)
}

func TestDiffViolationsValues_NoOverlap(t *testing.T) {
	original := []model.RuleFunctionResult{
		makeResult("rule-1", "$.paths./old", "old violation"),
	}
	newResults := []model.RuleFunctionResult{
		makeResult("rule-2", "$.paths./new", "new violation"),
		makeResult("rule-3", "$.info", "another new"),
	}
	result, stats := DiffViolationsValues(original, newResults)
	assert.Len(t, result, 2)
	assert.Equal(t, 0, stats.ResultsDropped)
}

func TestDiffViolationsValues_FullOverlap(t *testing.T) {
	violations := []model.RuleFunctionResult{
		makeResult("rule-1", "$.paths./items.post", "missing description"),
		makeResult("rule-2", "$.info.contact", "missing name"),
	}
	result, stats := DiffViolationsValues(violations, violations)
	assert.Empty(t, result)
	assert.Equal(t, 2, stats.ResultsDropped)
	assert.Len(t, stats.RulesFullyFiltered, 2)
}

func TestDiffViolationsValues_PartialOverlap(t *testing.T) {
	original := []model.RuleFunctionResult{
		makeResult("rule-1", "$.paths./items.post", "missing description"),
	}
	newResults := []model.RuleFunctionResult{
		makeResult("rule-1", "$.paths./items.post", "missing description"), // same — suppressed
		makeResult("rule-2", "$.info", "missing contact"),                  // new — kept
	}
	result, stats := DiffViolationsValues(original, newResults)
	assert.Len(t, result, 1)
	assert.Equal(t, "rule-2", result[0].RuleId)
	assert.Equal(t, 1, stats.ResultsDropped)
}

func TestDiffViolationsValues_SameRuleIdPathDifferentMessage(t *testing.T) {
	// info-contact-properties scenario: same (RuleId, Path), different Message
	original := []model.RuleFunctionResult{
		makeResult("info-contact-properties", "$.info.contact", "missing name"),
	}
	newResults := []model.RuleFunctionResult{
		makeResult("info-contact-properties", "$.info.contact", "missing name"), // same — suppressed
		makeResult("info-contact-properties", "$.info.contact", "missing url"),  // different message — kept
	}
	result, stats := DiffViolationsValues(original, newResults)
	assert.Len(t, result, 1)
	assert.Equal(t, "missing url", result[0].Message)
	assert.Equal(t, 1, stats.ResultsDropped)
	assert.Equal(t, 1, stats.RulesPartialFiltered["info-contact-properties"])
}

func TestDiffViolationsValues_SameRuleIdPathMessage_Suppressed(t *testing.T) {
	v := makeResult("rule-1", "$.paths./items.post", "missing description")
	original := []model.RuleFunctionResult{v}
	newResults := []model.RuleFunctionResult{v}
	result, stats := DiffViolationsValues(original, newResults)
	assert.Empty(t, result)
	assert.Equal(t, 1, stats.ResultsDropped)
}

func TestDiffViolationsValues_DuplicateCount(t *testing.T) {
	// Original has 2, new has 3 at same key → 1 reported
	v := makeResult("rule-1", "$.paths./items.post", "missing description")
	original := []model.RuleFunctionResult{v, v}
	newResults := []model.RuleFunctionResult{v, v, v}
	result, stats := DiffViolationsValues(original, newResults)
	assert.Len(t, result, 1)
	assert.Equal(t, 2, stats.ResultsDropped)
	assert.Equal(t, 2, stats.RulesPartialFiltered["rule-1"])
}

func TestDiffViolationsValues_EmptyPathFallsBackToPaths(t *testing.T) {
	original := []model.RuleFunctionResult{
		{
			RuleId:  "rule-1",
			Path:    "",
			Paths:   []string{"$.fallback.path"},
			Message: "test",
			Rule:    &model.Rule{Id: "rule-1"},
		},
	}
	newResults := []model.RuleFunctionResult{
		{
			RuleId:  "rule-1",
			Path:    "",
			Paths:   []string{"$.fallback.path"},
			Message: "test",
			Rule:    &model.Rule{Id: "rule-1"},
		},
	}
	result, _ := DiffViolationsValues(original, newResults)
	assert.Empty(t, result) // should be matched and suppressed
}

func TestDiffViolationsValues_StatsCorrect(t *testing.T) {
	original := []model.RuleFunctionResult{
		makeResult("rule-1", "$.a", "msg-a"),
		makeResult("rule-2", "$.b", "msg-b"),
		makeResult("rule-2", "$.c", "msg-c"),
	}
	newResults := []model.RuleFunctionResult{
		makeResult("rule-1", "$.a", "msg-a"),     // suppressed
		makeResult("rule-2", "$.b", "msg-b"),     // suppressed
		makeResult("rule-2", "$.c", "msg-c"),     // suppressed
		makeResult("rule-2", "$.d", "msg-d"),     // new
		makeResult("rule-3", "$.e", "msg-e"),     // new
	}
	result, stats := DiffViolationsValues(original, newResults)
	assert.Len(t, result, 2)
	assert.Equal(t, 5, stats.TotalResultsBefore)
	assert.Equal(t, 2, stats.TotalResultsAfter)
	assert.Equal(t, 3, stats.ResultsDropped)

	// rule-1: 1 before, 0 after (fully filtered)
	assert.Contains(t, stats.RulesFullyFiltered, "rule-1")

	// rule-2: 3 before, 1 after (partial: 2 dropped)
	assert.Equal(t, 2, stats.RulesPartialFiltered["rule-2"])
}

func TestDiffViolationsMixed_Basic(t *testing.T) {
	original := []model.RuleFunctionResult{
		makeResult("rule-1", "$.paths./items.post", "missing description"),
	}
	newResults := []*model.RuleFunctionResult{
		makeResultPtr("rule-1", "$.paths./items.post", "missing description"), // suppressed
		makeResultPtr("rule-2", "$.info", "missing contact"),                  // new
	}
	result, stats := DiffViolationsMixed(original, newResults)
	require.Len(t, result, 1)
	assert.Equal(t, "rule-2", result[0].RuleId)
	assert.Equal(t, 1, stats.ResultsDropped)
}

func TestDiffViolationsMixed_NilInNew(t *testing.T) {
	original := []model.RuleFunctionResult{}
	newResults := []*model.RuleFunctionResult{
		nil,
		makeResultPtr("rule-1", "$.a", "msg"),
	}
	result, stats := DiffViolationsMixed(original, newResults)
	// nil entries produce an empty key; since original also has no such entry, it stays
	assert.Len(t, result, 2)
	assert.Equal(t, 0, stats.ResultsDropped)
}

func TestExtractPath(t *testing.T) {
	assert.Equal(t, "$.a", extractPath("$.a", nil))
	assert.Equal(t, "$.a", extractPath("$.a", []string{"$.b"}))
	assert.Equal(t, "$.b", extractPath("", []string{"$.b"}))
	assert.Equal(t, "", extractPath("", nil))
	assert.Equal(t, "", extractPath("", []string{}))
}
