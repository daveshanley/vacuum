// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package motor

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"path/filepath"
	"strings"
	"sync"

	asyncapi_context "github.com/daveshanley/vacuum/asyncapi"
	"github.com/daveshanley/vacuum/functions"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/libasyncapi"
	"go.yaml.in/yaml/v4"
)

// ApplyAsyncAPIRulesToRuleSet handles AsyncAPI documents before the shared
// applicator enters libopenapi's OpenAPI document builder. The boolean return
// tells the caller whether the document was AsyncAPI-shaped and fully handled.
func ApplyAsyncAPIRulesToRuleSet(
	execution *RuleSetExecution,
	opts *ExecutionOptions,
	builtinFunctions functions.Functions,
) (*RuleSetExecutionResult, bool) {
	format := execution.SpecFormat
	if format == "" {
		detected, err := asyncapi_context.DetectFormat(execution.Spec)
		if err != nil {
			if errors.Is(err, libasyncapi.ErrAsyncAPI2NotSupported) ||
				errors.Is(err, libasyncapi.ErrInvalidAsyncAPIVersion) ||
				errors.Is(err, libasyncapi.ErrNoAsyncAPIVersion) {
				return &RuleSetExecutionResult{RuleSetExecution: execution, Errors: []error{err}}, true
			}
			if asyncapi_context.HasMarker(execution.Spec) {
				return &RuleSetExecutionResult{RuleSetExecution: execution, Errors: []error{err}}, true
			}
			return nil, false
		}
		format = detected
	}
	if !isAsyncAPIFormat(format) {
		return nil, false
	}

	if opts == nil {
		opts = &ExecutionOptions{}
	}

	logger := execution.Logger
	if logger == nil {
		if execution.SilenceLogs {
			logger = slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
		} else {
			logger = slog.New(slog.NewTextHandler(io.Discard, nil))
		}
	}

	config := libasyncapi.NewDocumentConfiguration()
	config.Logger = logger
	config.LocalFS = execution.RolodexFS
	config.AllowFileReferences = execution.AllowLookup
	config.AllowRemoteReferences = execution.AllowLookup
	config.ExtractRefsSequentially = execution.ExtractReferencesSequentially
	config.SkipCircularReferenceCheck = execution.SkipCircularCheck

	if execution.Base != "" {
		if strings.HasPrefix(execution.Base, "http") {
			base := execution.Base
			if !strings.HasSuffix(base, "/") {
				base += "/"
			}
			u, _ := url.Parse(base)
			config.BaseURL = u
			config.AllowRemoteReferences = true
		} else {
			config.BasePath = execution.Base
			config.AllowFileReferences = true
		}
	} else if execution.AllowLookup && execution.SpecFileName != "" && !strings.Contains(execution.SpecFileName, "://") {
		specDir := filepath.Dir(execution.SpecFileName)
		if specDir == "" {
			specDir = "."
		}
		config.BasePath = specDir
		config.AllowFileReferences = true
	}

	if vacuumUtils.ShouldUseCustomHTTPClient(execution.HTTPClientConfig) {
		httpClient, httpErr := vacuumUtils.CreateCustomHTTPClient(execution.HTTPClientConfig)
		if httpErr != nil {
			return &RuleSetExecutionResult{RuleSetExecution: execution, Errors: []error{fmt.Errorf("failed to create custom HTTP client: %w", httpErr)}}, true
		}
		config.RemoteURLHandler = vacuumUtils.CreateRemoteURLHandler(httpClient)
	}

	asyncCtx, err := asyncapi_context.NewContext(execution.Spec, execution.SpecFileName, config)
	if err != nil {
		return &RuleSetExecutionResult{RuleSetExecution: execution, Errors: []error{err}}, true
	}
	asyncCtx.Format = format
	if asyncCtx.SpecInfo != nil {
		asyncCtx.SpecInfo.SpecFormat = format
	}
	execution.AsyncAPI = asyncCtx
	execution.SpecFormat = format
	execution.CanonicalDocument = asyncCtx.RootNode
	execution.IndexResolved = asyncCtx.Index
	execution.IndexUnresolved = asyncCtx.Index

	documentResults := asyncAPIDocumentErrorResults(asyncCtx, asyncAPIDocumentErrorRule(execution.RuleSet))
	ruleResults, ignoredResults, fixedResults, errs := runAsyncAPIRules(execution, opts, builtinFunctions, asyncCtx, logger)
	if len(documentResults) > 0 {
		ruleResults = append(documentResults, ruleResults...)
		ruleResults = dedupeAsyncAPIResults(ruleResults)
	}

	if asyncCtx.Index == nil {
		rule := asyncAPIIndexBuildRule()
		ruleResults = append(ruleResults, model.RuleFunctionResult{
			RuleId:    rule.Id,
			Rule:      rule,
			StartNode: &yaml.Node{Line: 1, Column: 1},
			EndNode:   &yaml.Node{Line: 1, Column: 2},
			Message:   "unable to parse the AsyncAPI document, no index was created.",
			Path:      "$",
		})
	}

	filesProcessed := 0
	fileSize := int64(0)
	if asyncCtx.Rolodex != nil {
		filesProcessed = asyncCtx.Rolodex.RolodexTotalFiles()
		fileSize = asyncCtx.Rolodex.RolodexFileSize()
	}
	populateResultOrigins(ruleResults, asyncCtx.Rolodex, asyncCtx.Rolodex, execution.SpecFileName)
	ruleResults = finalizeResultPaths(
		ruleResults,
		asyncCtx.RootNode,
		execution.CanonicalDocument,
		execution.SpecFileName,
		asyncCtx.Rolodex,
		resolveExecutionAliases(execution.RuleSet, asyncCtx.Format, logger),
		false,
	)

	return &RuleSetExecutionResult{
		RuleSetExecution: execution,
		Results:          ruleResults,
		IgnoredResults:   ignoredResults,
		FixedResults:     fixedResults,
		Index:            asyncCtx.Index,
		SpecInfo:         asyncCtx.SpecInfo,
		Errors:           errs,
		FilesProcessed:   filesProcessed,
		FileSize:         fileSize,
		AsyncAPI:         asyncCtx,
	}, true
}

func asyncAPIDocumentErrorRule(ruleSet *rulesets.RuleSet) *model.Rule {
	if ruleSet != nil && ruleSet.Rules != nil {
		if rule := ruleSet.Rules[rulesets.AsyncAPI3DocumentResolved]; rule != nil {
			return rule
		}
	}
	if rule := rulesets.GetAllAsyncAPIRules()[rulesets.AsyncAPI3DocumentResolved]; rule != nil {
		return rule
	}
	return &model.Rule{
		Id:           rulesets.AsyncAPI3DocumentResolved,
		Name:         "Check resolved AsyncAPI v3 document structure",
		Severity:     model.SeverityError,
		RuleCategory: model.RuleCategories[model.CategoryValidation],
	}
}

func asyncAPIDocumentErrorResults(asyncCtx *asyncapi_context.Context, rule *model.Rule) []model.RuleFunctionResult {
	if asyncCtx == nil || rule == nil {
		return nil
	}
	errors := asyncCtx.DocumentErrors()
	if len(errors) == 0 {
		return nil
	}
	node := asyncAPIDocumentResultNode(asyncCtx.RootNode)
	results := make([]model.RuleFunctionResult, 0, len(errors))
	for _, err := range errors {
		if err == nil {
			continue
		}
		results = append(results, model.RuleFunctionResult{
			RuleId:       rule.Id,
			RuleSeverity: rule.Severity,
			Rule:         rule,
			StartNode:    node,
			EndNode:      vacuumUtils.BuildEndNode(node),
			Message:      err.Error(),
			Path:         "$",
		})
	}
	return results
}

func asyncAPIDocumentResultNode(root *yaml.Node) *yaml.Node {
	if root != nil && root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		return root.Content[0]
	}
	if root != nil {
		return root
	}
	return &yaml.Node{Line: 1, Column: 1}
}

func dedupeAsyncAPIResults(results []model.RuleFunctionResult) []model.RuleFunctionResult {
	if len(results) <= 1 {
		return results
	}
	seen := make(map[string]struct{}, len(results))
	deduped := make([]model.RuleFunctionResult, 0, len(results))
	for _, result := range results {
		key := strings.Join([]string{result.RuleId, result.Path, result.Message}, "\x00")
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, result)
	}
	return deduped
}

func runAsyncAPIRules(
	execution *RuleSetExecution,
	opts *ExecutionOptions,
	builtinFunctions functions.Functions,
	asyncCtx *asyncapi_context.Context,
	logger *slog.Logger,
) ([]model.RuleFunctionResult, []model.RuleFunctionResult, []model.RuleFunctionResult, []error) {
	var ruleResults []model.RuleFunctionResult
	var ignoredResults []model.RuleFunctionResult
	var fixedResults []model.RuleFunctionResult
	var errs []error
	if execution.RuleSet == nil || asyncCtx.Index == nil {
		return ruleResults, ignoredResults, fixedResults, errs
	}

	ignoreIdx := buildInlineIgnoreIndex(execution.CanonicalDocument)
	specHasInlineIgnores := ignoreIdx != nil
	resolvedAliases := resolveExecutionAliases(execution.RuleSet, asyncCtx.Format, logger)

	applicableRules := applicableRulesForFormat(execution.RuleSet, asyncCtx.Format)
	totalRules := len(applicableRules)
	if totalRules == 0 {
		return ruleResults, ignoredResults, fixedResults, errs
	}

	var schemaPathCache sync.Map
	runResults, runIgnored, runFixed, runErrs := runRuleContexts(
		execution,
		applicableRules,
		logger,
		func(rule *model.Rule) ruleContext {
			ruleResolved := opts.ResolveAllRefs || rule.Resolved
			return ruleContext{
				rule:               rule,
				specNode:           asyncCtx.RootNode,
				specNodeUnresolved: asyncCtx.RootNode,
				builtinFunctions:   builtinFunctions,
				specInfo:           asyncCtx.SpecInfo,
				index:              asyncCtx.Index,
				indexUnresolved:    asyncCtx.Index,
				asyncAPI:           asyncCtx,
				customFunctions:    execution.CustomFunctions,
				autoFixFunctions:   execution.AutoFixFunctions,
				panicFunc:          execution.PanicFunction,
				silenceLogs:        execution.SilenceLogs,
				skipDocumentCheck:  execution.SkipDocumentCheck,
				logger:             logger,
				nodeLookupTimeout:  execution.NodeLookupTimeout,
				applyAutoFixes:     execution.ApplyAutoFixes,
				resolvedExecution:  ruleResolved,
				fetchConfig:        execution.FetchConfig,
				turboMode:          execution.TurboMode,
				hasInlineIgnores:   specHasInlineIgnores,
				ignoreIndex:        ignoreIdx,
				schemaPathCache:    &schemaPathCache,
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

func asyncAPIIndexBuildRule() *model.Rule {
	return &model.Rule{
		Name:         "Check that an index can be created from the AsyncAPI document",
		Id:           "build-index",
		Description:  "vacuum must be able to index the AsyncAPI document; if it cannot then it cannot be linted",
		Given:        "$",
		Resolved:     false,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryValidation],
		Type:         "validation",
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "blank",
		},
		HowToFix: "An index is required to use vacuum. If an index cannot be created then the AsyncAPI document cannot be read. Check the document syntax.",
	}
}

func isAsyncAPIFormat(format string) bool {
	return format == model.AsyncAPI3 ||
		format == model.AsyncAPI30 ||
		format == model.AsyncAPI31
}
