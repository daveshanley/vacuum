// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package motor

import (
	"context"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
)

type ruleContextBuilder func(rule *model.Rule) ruleContext

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

	ruleSem := make(chan struct{}, runtime.NumCPU())
	done := make(chan bool, len(rules))
	var resultLock sync.Mutex

	for _, rule := range rules {
		ruleSem <- struct{}{}
		go func(rule *model.Rule) {
			defer func() { <-ruleSem }()

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
				ctx.logger.Error("Rule timed out, skipping", "rule", rule.Id, "timeout", execution.Timeout)
			case <-doneChan:
				resultLock.Lock()
				ruleResults = append(ruleResults, localResults...)
				ignoredResults = append(ignoredResults, localIgnored...)
				fixedResults = append(fixedResults, localFixed...)
				errs = append(errs, localErrs...)
				resultLock.Unlock()
			}
			done <- true
		}(rule)
	}

	for completed := 0; completed < len(rules); completed++ {
		<-done
	}
	return ruleResults, ignoredResults, fixedResults, errs
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
