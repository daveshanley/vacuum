// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/model/reports"
	"github.com/daveshanley/vacuum/rulesets"
)

// RenderDetailsOptions contains all options for rendering detailed results
type RenderDetailsOptions struct {
	Results      []*model.RuleFunctionResult
	SpecData     []string
	Snippets     bool
	Errors       bool
	Silent       bool
	NoMessage    bool
	AllResults   bool
	NoClip       bool
	FileName     string
	NoStyle      bool
}

// RenderSummaryOptions contains all options for rendering summary
type RenderSummaryOptions struct {
	RuleResultSet    *model.RuleResultSet
	RuleSet          *rulesets.RuleSet
	RuleCategories   []*model.RuleCategory
	Statistics       *reports.ReportStatistics
	Filename         string
	Silent           bool
	NoStyle          bool
	PipelineOutput   bool
	ShowRules        bool
	RenderRules      bool
	ReportStats      *reports.ReportStatistics
	TotalFiles       int
	Severity         string
}