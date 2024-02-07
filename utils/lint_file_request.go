// Copyright 2023-2024 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package utils

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"log/slog"
	"sync"
)

type LintFileRequest struct {
	FileName                 string
	BaseFlag                 string
	MultiFile                bool
	Remote                   bool
	SkipCheckFlag            bool
	Silent                   bool
	DetailsFlag              bool
	TimeFlag                 bool
	NoMessageFlag            bool
	AllResultsFlag           bool
	FailSeverityFlag         string
	CategoryFlag             string
	SnippetsFlag             bool
	ErrorsFlag               bool
	TotalFiles               int
	FileIndex                int
	TimeoutFlag              int
	IgnoreArrayCircleRef     bool
	IgnorePolymorphCircleRef bool
	DefaultRuleSets          rulesets.RuleSets
	SelectedRS               *rulesets.RuleSet
	Functions                map[string]model.RuleFunction
	Lock                     *sync.Mutex
	Logger                   *slog.Logger
}
