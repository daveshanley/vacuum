// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package motor

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	arazzo_context "github.com/daveshanley/vacuum/arazzo"
	"github.com/daveshanley/vacuum/functions"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	validation "github.com/pb33f/libopenapi-validator/arazzo"
)

func applyArazzoRulesToRuleSet(execution *RuleSetExecution, opts *ExecutionOptions, builtins functions.Functions, control *executionControl) (*RuleSetExecutionResult, bool) {
	if isJSONSchemaFormat(execution.SpecFormat) {
		return nil, false
	}
	format, err := arazzo_context.DetectFormat(execution.Spec)
	if format == "" && !model.FormatMatches(model.Arazzo, execution.SpecFormat) {
		if err == nil || !bytes.Contains(execution.Spec, []byte("arazzo")) {
			return nil, false
		}
	}
	if err != nil {
		return executionErrorResult(execution, control, err), true
	}
	ctx, err := arazzo_context.NewContext(execution.Spec, execution.SpecFileName)
	if err != nil {
		return executionErrorResult(execution, control, err), true
	}
	execution.Arazzo = ctx
	execution.SpecFormat = ctx.SpecInfo.SpecFormat
	execution.CanonicalDocument = ctx.RootNode
	execution.IndexResolved, execution.IndexUnresolved = ctx.Index, ctx.Index
	logger := execution.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	result := &RuleSetExecutionResult{RuleSetExecution: execution, Index: ctx.Index, SpecInfo: ctx.SpecInfo, Arazzo: ctx}
	for _, rule := range applicableRulesForFormat(execution.RuleSet, execution.SpecFormat) {
		if ruleUsesFunction(rule, "arazzoDocument") {
			resolver, uri, options, err := arazzoValidationOptions(execution)
			if err != nil {
				result.Errors = []error{err}
				return result, true
			}
			// Document validation includes source loading and is bounded by the
			// same timeout as an individual rule, plus the caller's run deadline.
			timeout := execution.Timeout
			if timeout <= 0 {
				timeout = 5 * time.Second
			}
			validationCtx, cancel := context.WithTimeout(control.Context(), timeout)
			err = ctx.Validate(validationCtx, uri, resolver.Sources, options...)
			cancel()
			result.FilesProcessed, result.FileSize = resolver.Files, resolver.Bytes
			if err != nil {
				result.Errors = []error{err}
				return result, true
			}
			break
		}
	}
	result.Results, result.IgnoredResults, result.FixedResults, result.Errors = runDocumentRules(execution, opts, builtins, ctx.SpecInfo, ctx.Index, logger, control)
	result.Results = finalizeResultPaths(result.Results, ctx.RootNode, ctx.RootNode, execution.SpecFileName, nil,
		resolveExecutionAliases(execution.RuleSet, ctx.SpecInfo.SpecFormat, logger), false)
	populateResultOrigins(result.Results, nil, nil, execution.SpecFileName)
	result.Errors = appendContextErrorToErrors(result.Errors, control.Err())
	return result, true
}

func arazzoValidationOptions(execution *RuleSetExecution) (*arazzo_context.Resolver, string, []validation.Option, error) {
	uri, err := arazzo_context.DocumentURI(execution.SpecFileName, execution.Base)
	if err != nil {
		return nil, "", nil, err
	}
	baseURL, _ := url.Parse(uri)
	resolver := &arazzo_context.Resolver{AllowFiles: execution.AllowLookup, AllowRemote: execution.AllowLookup,
		LocalFS: execution.RolodexFS, Sources: make(map[string]*arazzo_context.SourceDocument)}
	if baseURL != nil {
		resolver.BasePath = filepath.Dir(filepath.FromSlash(baseURL.Path))
	}
	if execution.Base != "" {
		if strings.HasPrefix(execution.Base, "http://") || strings.HasPrefix(execution.Base, "https://") {
			resolver.AllowRemote = true
		} else {
			resolver.AllowFiles = true
		}
	}
	resolver.Client, err = utils.CreateHTTPClientIfNeeded(execution.HTTPClientConfig)
	if err != nil {
		return nil, "", nil, err
	}
	options := []validation.Option{validation.WithResolver(resolver)}
	if execution.RuleSet != nil {
		if rule := execution.RuleSet.Rules[string(validation.CodeAdvisory)]; rule != nil && rule.Severity != model.SeverityNone {
			options = append(options, validation.WithAdvisories())
		}
	}
	return resolver, uri, options, nil
}
