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
	// MainSpecPath selects the root specification for multi-file language-server diagnostics.
	// An empty value lints each open document independently.
	MainSpecPath             string
	FileName                 string
	BaseFlag                 string
	MultiFile                bool
	Remote                   bool
	SkipCheckFlag            bool
	Silent                   bool
	DetailsFlag              bool
	TimeFlag                 bool
	NoMessageFlag            bool
	ExtensionRefs            bool
	AllResultsFlag           bool
	FailSeverityFlag         string
	CategoryFlag             string
	SnippetsFlag             bool
	ErrorsFlag               bool
	TotalFiles               int
	FileIndex                int
	TimeoutFlag              int
	LookupTimeoutFlag        int
	IgnoreArrayCircleRef     bool
	IgnorePolymorphCircleRef bool
	NoClip                   bool
	IgnoredResults           model.IgnoredItems
	IgnoreFiles              []string // Explicit CLI ignore paths, overriding language-server file config.
	DefaultRuleSets          rulesets.RuleSets
	SelectedRS               *rulesets.RuleSet
	Functions                map[string]model.RuleFunction
	Lock                     *sync.Mutex
	Logger                   *slog.Logger
	PipelineOutput           bool
	ShowRules                bool
	HTTPClientConfig         HTTPClientConfig
}
