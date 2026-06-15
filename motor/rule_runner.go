// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package motor

import (
	"context"
	"log/slog"
	"sort"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
)

const (
	// Keep rule execution independent from GOMAXPROCS without flooding shared
	// spec/index lookup state with hundreds of concurrent rules.
	maxRuleConcurrency = 32
)

type ruleContextBuilder func(rule *model.Rule) ruleContext

type ruleContextResult struct {
	index          int
	ruleResults    []model.RuleFunctionResult
	ignoredResults []model.RuleFunctionResult
	fixedResults   []model.RuleFunctionResult
	errors         []error
}

type ruleJob struct {
	index int
	rule  *model.Rule
}

func runRuleContexts(
	execution *RuleSetExecution,
	rules []*model.Rule,
	logger *slog.Logger,
	buildContext ruleContextBuilder,
) ([]model.RuleFunctionResult, []model.RuleFunctionResult, []model.RuleFunctionResult, []error) {
	var ruleResults []model.RuleFunctionResult
	var ignoredResults []model.RuleFunctionResult
	var fixedResults []model.RuleFunctionResult
	var errs []error

	if execution == nil || len(rules) == 0 {
		return ruleResults, ignoredResults, fixedResults, errs
	}
	if execution.Timeout <= 0 {
		execution.Timeout = time.Second * 5
	}
	if execution.NodeLookupTimeout <= 0 {
		execution.NodeLookupTimeout = time.Millisecond * 500
	}

	workerCount := ruleConcurrencyLimit(len(rules))
	jobs := make(chan ruleJob)
	done := make(chan ruleContextResult, len(rules))

	for i := 0; i < workerCount; i++ {
		go func() {
			for job := range jobs {
				result := executeRuleContext(execution, job.rule, logger, buildContext)
				result.index = job.index
				done <- result
			}
		}()
	}

	for i, rule := range rules {
		jobs <- ruleJob{index: i, rule: rule}
	}
	close(jobs)

	resultsByRule := make([]ruleContextResult, len(rules))
	for completed := 0; completed < len(rules); completed++ {
		result := <-done
		resultsByRule[result.index] = result
	}
	for _, result := range resultsByRule {
		ruleResults = append(ruleResults, result.ruleResults...)
		ignoredResults = append(ignoredResults, result.ignoredResults...)
		fixedResults = append(fixedResults, result.fixedResults...)
		errs = append(errs, result.errors...)
	}
	return ruleResults, ignoredResults, fixedResults, errs
}

func ruleConcurrencyLimit(ruleCount int) int {
	if ruleCount <= 0 {
		return 0
	}
	if ruleCount < maxRuleConcurrency {
		return ruleCount
	}
	return maxRuleConcurrency
}

func executeRuleContext(
	execution *RuleSetExecution,
	rule *model.Rule,
	logger *slog.Logger,
	buildContext ruleContextBuilder,
) ruleContextResult {
	ctx := buildContext(rule)
	if ctx.logger == nil {
		ctx.logger = logger
	}

	timeoutCtx, ruleCancel := context.WithTimeout(context.Background(), execution.Timeout)
	defer ruleCancel()
	doneChan := make(chan struct{})

	localResults := []model.RuleFunctionResult{}
	localIgnored := []model.RuleFunctionResult{}
	localFixed := []model.RuleFunctionResult{}
	localErrs := []error{}
	localCtx := ctx
	localCtx.ruleResults = &localResults
	localCtx.ignoredResults = &localIgnored
	localCtx.fixedResults = &localFixed
	localCtx.errors = &localErrs

	go runRule(localCtx, doneChan)
	select {
	case <-timeoutCtx.Done():
		if ctx.logger != nil {
			ctx.logger.Error("Rule timed out, skipping", "rule", rule.Id, "timeout", execution.Timeout)
		}
		// runRule is not cancellable; on timeout its goroutine may finish later,
		// writing only to these orphaned local slices.
		return ruleContextResult{}
	case <-doneChan:
		return ruleContextResult{
			ruleResults:    localResults,
			ignoredResults: localIgnored,
			fixedResults:   localFixed,
			errors:         localErrs,
		}
	}
}

func applicableRulesForFormat(ruleSet *rulesets.RuleSet, format string) []*model.Rule {
	if ruleSet == nil {
		return nil
	}
	applicable := make([]*model.Rule, 0, len(ruleSet.Rules))
	for _, rule := range ruleSet.Rules {
		if rule == nil {
			continue
		}
		ruleFormats := applicableRuleFormats(ruleSet, rule)
		if len(ruleFormats) == 0 && model.FormatMatches(model.AsyncAPI3, format) {
			continue
		}
		if len(ruleFormats) > 0 && format != "" {
			matches := false
			for _, ruleFormat := range ruleFormats {
				if model.FormatMatches(ruleFormat, format) {
					matches = true
					break
				}
			}
			if !matches {
				continue
			}
		}
		applicable = append(applicable, rule)
	}
	sort.SliceStable(applicable, func(i, j int) bool {
		return applicable[i].Id < applicable[j].Id
	})
	return applicable
}

func applicableRuleFormats(ruleSet *rulesets.RuleSet, rule *model.Rule) []string {
	if len(rule.Formats) > 0 {
		return rule.Formats
	}
	if ruleSet != nil && len(ruleSet.Formats) > 0 {
		return ruleSet.Formats
	}
	return nil
}

func resolveExecutionAliases(ruleSet *rulesets.RuleSet, format string, logger *slog.Logger) map[string][]string {
	if ruleSet == nil || ruleSet.ParsedAliases == nil {
		return nil
	}
	resolved := rulesets.ResolveAliasesForFormat(ruleSet.ParsedAliases, format)
	expanded, err := rulesets.ExpandAliasReferences(resolved)
	if err != nil {
		if logger != nil {
			logger.Error("alias expansion error", "error", err)
		}
		return nil
	}
	return expanded
}
