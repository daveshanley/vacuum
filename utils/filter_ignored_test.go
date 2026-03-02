// Copyright 2025 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package utils

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFilterIgnoredResults(t *testing.T) {
	results := []model.RuleFunctionResult{
		{Path: "a/b/c", Rule: &model.Rule{Id: "XXX"}},
		{Path: "a/b", Rule: &model.Rule{Id: "XXX"}},
		{Path: "a", Rule: &model.Rule{Id: "XXX"}},
		{Path: "a/b/c", Rule: &model.Rule{Id: "YYY"}},
		{Path: "a/b", Rule: &model.Rule{Id: "YYY"}},
		{Path: "a", Rule: &model.Rule{Id: "YYY"}},
		{Path: "a/b/c", Rule: &model.Rule{Id: "ZZZ"}},
		{Path: "a/b", Rule: &model.Rule{Id: "ZZZ"}},
		{Path: "a", Rule: &model.Rule{Id: "ZZZ"}},
	}

	igItems := model.IgnoredItems{
		"XXX": []string{"a/b/c"},
		"YYY": []string{"a/b"},
	}

	filtered := FilterIgnoredResults(results, igItems)

	assert.Len(t, filtered, 7)

	// Check that the ignored items are not in the result
	for _, r := range filtered {
		if r.Rule.Id == "XXX" {
			assert.NotEqual(t, "a/b/c", r.Path)
		}
		if r.Rule.Id == "YYY" {
			assert.NotEqual(t, "a/b", r.Path)
		}
	}
}

func TestFilterIgnoredResultsPtr(t *testing.T) {
	results := []*model.RuleFunctionResult{
		{Path: "a/b/c", Rule: &model.Rule{Id: "XXX"}},
		{Path: "a/b", Rule: &model.Rule{Id: "XXX"}},
		{Path: "a", Rule: &model.Rule{Id: "XXX"}},
		{Path: "a/b/c", Rule: &model.Rule{Id: "YYY"}},
		{Path: "a/b", Rule: &model.Rule{Id: "YYY"}},
		{Path: "a", Rule: &model.Rule{Id: "YYY"}},
		{Path: "a/b/c", Rule: &model.Rule{Id: "ZZZ"}},
		{Path: "a/b", Rule: &model.Rule{Id: "ZZZ"}},
		{Path: "a", Rule: &model.Rule{Id: "ZZZ"}},
	}

	igItems := model.IgnoredItems{
		"XXX": []string{"a/b/c"},
		"YYY": []string{"a/b"},
	}

	filtered := FilterIgnoredResultsPtr(results, igItems)

	assert.Len(t, filtered, 7)

	// Check that the ignored items are not in the result
	for _, r := range filtered {
		if r.Rule.Id == "XXX" {
			assert.NotEqual(t, "a/b/c", r.Path)
		}
		if r.Rule.Id == "YYY" {
			assert.NotEqual(t, "a/b", r.Path)
		}
	}
}

func TestFilterIgnoredResultsWithPaths(t *testing.T) {
	results := []model.RuleFunctionResult{
		{Path: "main", Paths: []string{"a/b/c", "d/e/f"}, Rule: &model.Rule{Id: "XXX"}},
		{Path: "main", Paths: []string{"g/h/i"}, Rule: &model.Rule{Id: "XXX"}},
		{Path: "main", Rule: &model.Rule{Id: "YYY"}},
	}

	igItems := model.IgnoredItems{
		"XXX": []string{"d/e/f"},
		"YYY": []string{"main"},
	}

	filtered := FilterIgnoredResults(results, igItems)

	// First result should be filtered because one of its paths matches
	// Second result should not be filtered
	// Third result should be filtered because its path matches
	assert.Len(t, filtered, 1)
	assert.Equal(t, "XXX", filtered[0].Rule.Id)
	assert.Contains(t, filtered[0].Paths, "g/h/i")
}
