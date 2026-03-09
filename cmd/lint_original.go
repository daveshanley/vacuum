// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
)

// LintOriginalSpec lints the original spec using the provided execution config as a template.
// All config fields (RuleSet, CustomFunctions, Timeout, etc.) are copied from the template
// to guarantee exact config parity. Only Spec, SpecFileName, and Base are replaced.
// Returns nil results (not an error) if the original spec has parse errors.
func LintOriginalSpec(originalPath string, template *motor.RuleSetExecution) ([]model.RuleFunctionResult, error) {
	originalBytes, err := os.ReadFile(originalPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read original spec file '%s': %w", originalPath, err)
	}

	// Resolve base path for original spec
	absPath, pathErr := filepath.Abs(originalPath)
	var resolvedBase string
	if pathErr == nil {
		resolvedBase = filepath.Dir(absPath)
	}

	// Clone the template execution with swapped spec-specific fields.
	// All other config fields are copied from the template to guarantee parity.
	exec := &motor.RuleSetExecution{
		RuleSet:                         template.RuleSet,
		Spec:                            originalBytes,
		SpecFileName:                    originalPath,
		CustomFunctions:                 template.CustomFunctions,
		AutoFixFunctions:                nil, // don't fix the original
		SilenceLogs:                     true,
		Base:                            resolvedBase,
		AllowLookup:                     template.AllowLookup,
		SkipDocumentCheck:               template.SkipDocumentCheck,
		Timeout:                         template.Timeout,
		NodeLookupTimeout:               template.NodeLookupTimeout,
		IgnoreCircularArrayRef:          template.IgnoreCircularArrayRef,
		IgnoreCircularPolymorphicRef:    template.IgnoreCircularPolymorphicRef,
		ExtractReferencesFromExtensions: template.ExtractReferencesFromExtensions,
		HTTPClientConfig:                template.HTTPClientConfig,
		FetchConfig:                     template.FetchConfig,
		TurboMode:                       template.TurboMode,
		BuildDeepGraph:                  template.BuildDeepGraph,
		SkipResolve:                     template.SkipResolve,
		SkipCircularCheck:               template.SkipCircularCheck,
		SkipSchemaErrors:                template.SkipSchemaErrors,
	}

	// Set a reasonable timeout if none was configured
	if exec.Timeout == 0 {
		exec.Timeout = 5 * time.Minute
	}

	result := motor.ApplyRulesToRuleSet(exec)

	// If original spec has parse errors, return nil — safe default means all new violations get reported
	if len(result.Errors) > 0 {
		return nil, nil
	}

	return result.Results, nil
}
