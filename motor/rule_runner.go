// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package motor

import (
	"context"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
)

type ruleContextBuilder func(rule *model.Rule) ruleContext

type ruleContextOutcome uint8

const (
	ruleContextUnknown ruleContextOutcome = iota
	ruleContextCompleted
	ruleContextTimedOut
	ruleContextCancelled
)

type ruleContextResult struct {
	index          int
	outcome        ruleContextOutcome
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
	control *executionControl,
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

	workerCount := ruleConcurrencyLimit(len(rules), control.MaxRuleConcurrency())
	jobs := make(chan ruleJob)
	completedResults := make(chan ruleContextResult, len(rules))
	var workers sync.WaitGroup

	for i := 0; i < workerCount; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for {
				select {
				case <-control.Done():
					return
				case job, ok := <-jobs:
					if !ok {
						return
					}
					// The receive and cancellation may become ready together.
					// Do not start the rule if cancellation already won.
					select {
					case <-control.Done():
						return
					default:
					}
					result := executeRuleContext(control, execution, job.rule, logger, buildContext)
					if result.outcome != ruleContextCompleted {
						if result.outcome == ruleContextCancelled {
							return
						}
						continue
					}
					result.index = job.index
					completedResults <- result
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for i, rule := range rules {
			select {
			case <-control.Done():
				return
			case jobs <- ruleJob{index: i, rule: rule}:
			}
		}
	}()

	go func() {
		workers.Wait()
		close(completedResults)
	}()

	resultsByRule := make([]ruleContextResult, len(rules))
	for result := range completedResults {
		resultsByRule[result.index] = result
	}
	for _, result := range resultsByRule {
		if result.outcome != ruleContextCompleted {
			continue
		}
		ruleResults = append(ruleResults, result.ruleResults...)
		ignoredResults = append(ignoredResults, result.ignoredResults...)
		fixedResults = append(fixedResults, result.fixedResults...)
		errs = append(errs, result.errors...)
	}
	return ruleResults, ignoredResults, fixedResults, errs
}

func ruleConcurrencyLimit(ruleCount, configuredMaximum int) int {
	if ruleCount <= 0 {
		return 0
	}
	if configuredMaximum <= 0 {
		configuredMaximum = defaultMaxRuleConcurrency
	}
	if ruleCount < configuredMaximum {
		return ruleCount
	}
	return configuredMaximum
}

func executeRuleContext(
	control *executionControl,
	execution *RuleSetExecution,
	rule *model.Rule,
	logger *slog.Logger,
	buildContext ruleContextBuilder,
) ruleContextResult {
	ctx := buildContext(rule)
	if ctx.logger == nil {
		ctx.logger = logger
	}

	timeoutCtx, ruleCancel := context.WithTimeout(control.Context(), execution.Timeout)
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
	runGuard := &ruleRunGuard{}
	localCtx.runGuard = runGuard
	localCtx.executionContext = control.Context()
	if control != nil {
		localCtx.autoFixGate = control.autoFixGate
	}

	control.startRuleRun()
	go func() {
		defer control.finishRuleRun()
		runRule(localCtx, doneChan)
	}()
	select {
	case <-timeoutCtx.Done():
		// Prefer a rule that has already crossed its complete boundary when
		// completion and cancellation become observable together.
		select {
		case <-doneChan:
			return ruleContextResult{
				outcome:        ruleContextCompleted,
				ruleResults:    localResults,
				ignoredResults: localIgnored,
				fixedResults:   localFixed,
				errors:         localErrs,
			}
		default:
		}
		if runGuard.abandon() {
			// Once the bounded auto-fix publication phase starts, wait for it to
			// finish so returned fixed results and ModifiedSpec cannot diverge.
			<-doneChan
			return ruleContextResult{
				outcome:        ruleContextCompleted,
				ruleResults:    localResults,
				ignoredResults: localIgnored,
				fixedResults:   localFixed,
				errors:         localErrs,
			}
		}
		if control.Err() != nil {
			// runRule is not cancellable; it may finish after this call returns.
			// Its output remains isolated in these private local slices.
			return ruleContextResult{outcome: ruleContextCancelled}
		}
		if ctx.logger != nil {
			ctx.logger.Error("Rule timed out, skipping", "rule", rule.Id, "timeout", execution.Timeout)
		}
		// runRule is not cancellable; on timeout its goroutine may finish later,
		// writing only to these orphaned local slices.
		return ruleContextResult{outcome: ruleContextTimedOut}
	case <-doneChan:
		return ruleContextResult{
			outcome:        ruleContextCompleted,
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
