// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package motor

import (
	"github.com/daveshanley/vacuum/functions"
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"
	"log/slog"
	"sync"
)

// runDocumentRules runs parser-neutral documents through the shared scheduler.
func runDocumentRules(
	execution *RuleSetExecution,
	opts *ExecutionOptions,
	builtinFunctions functions.Functions,
	specInfo *datamodel.SpecInfo,
	specIndex *index.SpecIndex,
	logger *slog.Logger,
	control *executionControl,
) ([]model.RuleFunctionResult, []model.RuleFunctionResult, []model.RuleFunctionResult, []error) {
	var ruleResults []model.RuleFunctionResult
	var ignoredResults []model.RuleFunctionResult
	var fixedResults []model.RuleFunctionResult
	var errs []error
	if execution.RuleSet == nil || specIndex == nil {
		return ruleResults, ignoredResults, fixedResults, errs
	}

	ignoreIdx := buildInlineIgnoreIndex(execution.CanonicalDocument)
	specHasInlineIgnores := ignoreIdx != nil
	var sourceIgnores map[string]sourceInlineIgnore
	if execution.Arazzo != nil {
		sourceIgnores = make(map[string]sourceInlineIgnore, len(execution.Arazzo.Sources))
		for origin, root := range execution.Arazzo.Sources {
			sourceIgnores[origin] = sourceInlineIgnore{root: root, index: buildInlineIgnoreIndex(root)}
		}
	}
	resolvedAliases := resolveExecutionAliases(execution.RuleSet, specInfo.SpecFormat, logger)

	applicableRules := applicableRulesForFormat(execution.RuleSet, specInfo.SpecFormat)
	totalRules := len(applicableRules)
	if totalRules == 0 {
		return ruleResults, ignoredResults, fixedResults, errs
	}

	var schemaPathCache sync.Map
	var ruleJSONPathCache sync.Map
	runResults, runIgnored, runFixed, runErrs := runRuleContexts(
		control,
		execution,
		applicableRules,
		logger,
		func(rule *model.Rule) ruleContext {
			ruleResolved := opts.ResolveAllRefs || rule.Resolved
			return ruleContext{
				rule:               rule,
				specNode:           specInfo.RootNode,
				specNodeUnresolved: specInfo.RootNode,
				builtinFunctions:   builtinFunctions,
				specInfo:           specInfo,
				index:              specIndex,
				indexUnresolved:    specIndex,
				asyncAPI:           execution.AsyncAPI,
				arazzo:             execution.Arazzo,
				customFunctions:    execution.CustomFunctions,
				autoFixFunctions:   execution.AutoFixFunctions,
				panicFunc:          execution.PanicFunction,
				silenceLogs:        execution.SilenceLogs,
				skipDocumentCheck:  execution.SkipDocumentCheck,
				logger:             logger,
				nodeLookupTimeout:  execution.NodeLookupTimeout,
				ruleTimeout:        execution.Timeout,
				applyAutoFixes:     execution.ApplyAutoFixes,
				resolvedExecution:  ruleResolved,
				fetchConfig:        execution.FetchConfig,
				turboMode:          execution.TurboMode,
				hasInlineIgnores:   specHasInlineIgnores,
				ignoreIndex:        ignoreIdx,
				sourceIgnores:      sourceIgnores,
				schemaPathCache:    &schemaPathCache,
				ruleJSONPathCache:  &ruleJSONPathCache,
				expandedAliases:    resolvedAliases,
			}
		},
	)
	ruleResults = append(ruleResults, runResults...)
	ignoredResults = append(ignoredResults, runIgnored...)
	fixedResults = append(fixedResults, runFixed...)
	errs = append(errs, runErrs...)
	return ruleResults, ignoredResults, fixedResults, errs
}

// sourceInlineIgnore keeps an external finding in its owning document's scope.
type sourceInlineIgnore struct {
	root  *yaml.Node
	index *inlineIgnoreIndex
}

func resultHasInlineIgnore(ctx ruleContext, result model.RuleFunctionResult) bool {
	if result.Path == "" {
		return false
	}
	if ctx.arazzo != nil && result.Origin != nil {
		source := ctx.sourceIgnores[result.Origin.AbsoluteLocation]
		return checkInlineIgnoreByPathIndexed(source.index, source.root, result.Path, ctx.rule.Id)
	}
	return ctx.hasInlineIgnores && checkInlineIgnoreByPathIndexed(ctx.ignoreIndex, ctx.specNodeUnresolved, result.Path, ctx.rule.Id)
}
