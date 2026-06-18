// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/utils"
	drModel "github.com/pb33f/doctor/model"
	wcModel "github.com/pb33f/libopenapi/what-changed/model"
	"go.yaml.in/yaml/v4"
)

type lintOriginalResult struct {
	results []model.RuleFunctionResult
	release func()
}

type originalDiffWarningFunc func(error)

type originalResultSetDiffOptions struct {
	OriginalPath            string
	CurrentBytes            []byte
	CurrentPath             string
	ResultSet               *model.RuleResultSet
	Execution               *motor.RuleSetExecution
	ExecutionOptions        *motor.ExecutionOptions
	DrDocument              *drModel.DrDocument
	WarnOriginalLintFailure originalDiffWarningFunc
	WarnChangeReportFailure originalDiffWarningFunc
}

type originalValueDiffOptions struct {
	OriginalPath            string
	CurrentPath             string
	Results                 []model.RuleFunctionResult
	Execution               *motor.RuleSetExecution
	ExecutionOptions        *motor.ExecutionOptions
	ReuseCurrentResults     bool
	WarnOriginalLintFailure originalDiffWarningFunc
}

func (r *lintOriginalResult) releaseOwnedResources() {
	if r == nil || r.release == nil {
		return
	}
	r.release()
	r.release = nil
}

// LintOriginalSpec lints the original spec using the provided execution config as a template.
// All config fields (RuleSet, CustomFunctions, Timeout, etc.) are copied from the template
// to guarantee exact config parity. Only Spec, SpecFileName, and Base are replaced.
// Returns nil results (not an error) if the original spec has parse errors.
func LintOriginalSpec(originalPath string, template *motor.RuleSetExecution, executionOptions *motor.ExecutionOptions) ([]model.RuleFunctionResult, error) {
	result, err := lintOriginalSpecForDiff(originalPath, template, executionOptions)
	if result != nil {
		defer result.releaseOwnedResources()
	}
	if err != nil || result == nil {
		return nil, err
	}
	return result.results, nil
}

func lintOriginalSpecForDiff(originalPath string, template *motor.RuleSetExecution, executionOptions *motor.ExecutionOptions) (*lintOriginalResult, error) {
	originalBytes, err := os.ReadFile(originalPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read original spec file '%s': %w", originalPath, err)
	}

	// Resolve base path for original spec
	absPath, pathErr := filepath.Abs(originalPath)
	var resolvedBase string
	resolvedSpecPath := originalPath
	if pathErr == nil {
		resolvedBase = filepath.Dir(absPath)
		resolvedSpecPath = absPath
	}

	// Clone the template execution with swapped spec-specific fields.
	// All other config fields are copied from the template to guarantee parity.
	exec := &motor.RuleSetExecution{
		RuleSet:                         template.RuleSet,
		Spec:                            originalBytes,
		SpecFileName:                    resolvedSpecPath,
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

	result := motor.ApplyRulesToRuleSetWithOptions(exec, executionOptions)

	// If original spec has parse errors, return nil — safe default means all new violations get reported
	if len(result.Errors) > 0 {
		result.ReleaseOwnedResources()
		return nil, nil
	}

	return &lintOriginalResult{
		results: result.Results,
		release: result.ReleaseOwnedResources,
	}, nil
}

func originalSpecCanReuseCurrentResults(originalPath string, currentBytes []byte, currentPath string, currentBase string, customFunctions map[string]model.RuleFunction) bool {
	// Custom functions can observe runtime state or external inputs, so byte-equal
	// specs do not guarantee byte-equal lint results.
	if len(customFunctions) > 0 {
		return false
	}
	// A custom --base changes how identical relative refs resolve. Reuse results
	// only when the current spec is using the same spec-directory base that the
	// original lint path will use.
	if !currentSpecUsesDefaultBase(currentPath, currentBase) {
		return false
	}
	originalBytes, err := os.ReadFile(originalPath)
	if err != nil || !bytes.Equal(originalBytes, currentBytes) {
		return false
	}
	if sameLocalSpecFile(originalPath, currentPath) {
		return true
	}
	return externalReferenceFilesMatch(originalPath, currentPath, currentBytes)
}

func applyOriginalDiffToResultSet(opts originalResultSetDiffOptions) (*wcModel.DocumentChanges, bool) {
	if opts.OriginalPath == "" || opts.ResultSet == nil {
		return nil, false
	}

	var customFunctions map[string]model.RuleFunction
	var currentBase string
	if opts.Execution != nil {
		customFunctions = opts.Execution.CustomFunctions
		currentBase = opts.Execution.Base
	}
	if originalSpecCanReuseCurrentResults(opts.OriginalPath, opts.CurrentBytes, opts.CurrentPath, currentBase, customFunctions) {
		opts.ResultSet.Results = nil
		return emptyDocumentChanges(), true
	}

	if opts.Execution == nil {
		changeResult, changeErr := utils.GenerateChangeReportWithTree(opts.OriginalPath, opts.CurrentBytes, opts.CurrentPath)
		if changeErr != nil {
			if opts.WarnChangeReportFailure != nil {
				opts.WarnChangeReportFailure(changeErr)
			}
			return nil, false
		}
		if changeResult == nil {
			return nil, false
		}
		changeFilter := utils.NewChangeFilter(changeResult.DocumentChanges, opts.DrDocument)
		opts.ResultSet.Results = changeFilter.FilterResults(opts.ResultSet.Results)
		return changeResult.DocumentChanges, true
	}

	filtered := false
	originalLint, lintErr := lintOriginalSpecForDiff(opts.OriginalPath, opts.Execution, opts.ExecutionOptions)
	if lintErr != nil {
		if opts.WarnOriginalLintFailure != nil {
			opts.WarnOriginalLintFailure(lintErr)
		}
	} else {
		var originalResults []model.RuleFunctionResult
		if originalLint != nil {
			originalResults = originalLint.results
		}
		opts.ResultSet.Results, _ = utils.DiffViolationsMixedWithOriginBases(
			originalResults,
			opts.ResultSet.Results,
			opts.OriginalPath,
			opts.CurrentPath,
		)
		if originalLint != nil {
			originalLint.releaseOwnedResources()
		}
		filtered = true
	}

	changeResult, changeErr := utils.GenerateChangeReportWithTree(opts.OriginalPath, opts.CurrentBytes, opts.CurrentPath)
	if changeErr != nil {
		if opts.WarnChangeReportFailure != nil {
			opts.WarnChangeReportFailure(changeErr)
		}
		return nil, filtered
	}
	if changeResult == nil {
		return nil, filtered
	}
	return changeResult.DocumentChanges, filtered
}

func applyOriginalDiffToValues(opts originalValueDiffOptions) ([]model.RuleFunctionResult, *utils.ChangeFilterStats) {
	if opts.OriginalPath == "" {
		return opts.Results, nil
	}
	if opts.ReuseCurrentResults {
		// Current lint results are also the original results. Route through the
		// canonical differ so filtering stats stay identical to the full path.
		return utils.DiffViolationsValues(opts.Results, opts.Results)
	}
	if opts.Execution == nil {
		return opts.Results, nil
	}

	originalLint, lintErr := lintOriginalSpecForDiff(opts.OriginalPath, opts.Execution, opts.ExecutionOptions)
	if lintErr != nil {
		if opts.WarnOriginalLintFailure != nil {
			opts.WarnOriginalLintFailure(lintErr)
		}
		return opts.Results, nil
	}

	var originalResults []model.RuleFunctionResult
	if originalLint != nil {
		originalResults = originalLint.results
	}
	filteredResults, stats := utils.DiffViolationsValuesWithOriginBases(
		originalResults,
		opts.Results,
		opts.OriginalPath,
		opts.CurrentPath,
	)
	if originalLint != nil {
		originalLint.releaseOwnedResources()
	}
	return filteredResults, stats
}

func currentSpecUsesDefaultBase(currentPath string, currentBase string) bool {
	if currentBase == "" {
		return true
	}
	if currentPath == "" || currentPath == "stdin" || strings.Contains(currentPath, "://") {
		return false
	}

	currentAbs, currentErr := filepath.Abs(currentPath)
	baseAbs, baseErr := filepath.Abs(currentBase)
	if currentErr != nil || baseErr != nil {
		return false
	}
	return filepath.Clean(baseAbs) == filepath.Dir(filepath.Clean(currentAbs))
}

func emptyDocumentChanges() *wcModel.DocumentChanges {
	// Byte-identical specs have no structural changes, but libopenapi expects
	// the embedded PropertyChanges object to be initialized when stats are read.
	return &wcModel.DocumentChanges{
		PropertyChanges: wcModel.NewPropertyChanges(nil),
	}
}

func sameLocalSpecFile(leftPath, rightPath string) bool {
	if leftPath == "" || rightPath == "" ||
		strings.Contains(leftPath, "://") || strings.Contains(rightPath, "://") ||
		leftPath == "stdin" || rightPath == "stdin" {
		return false
	}

	leftInfo, leftErr := os.Stat(leftPath)
	rightInfo, rightErr := os.Stat(rightPath)
	if leftErr == nil && rightErr == nil && os.SameFile(leftInfo, rightInfo) {
		return true
	}

	leftAbs, leftAbsErr := filepath.Abs(leftPath)
	rightAbs, rightAbsErr := filepath.Abs(rightPath)
	return leftAbsErr == nil && rightAbsErr == nil && leftAbs == rightAbs
}

func externalReferenceFilesMatch(originalPath string, currentPath string, currentBytes []byte) bool {
	originalAbs, originalErr := filepath.Abs(originalPath)
	currentAbs, currentErr := filepath.Abs(currentPath)
	if originalErr != nil || currentErr != nil {
		return false
	}

	type refFilePair struct {
		originalPath string
		currentPath  string
		currentBytes []byte
	}
	queue := []refFilePair{{
		originalPath: originalAbs,
		currentPath:  currentAbs,
		currentBytes: currentBytes,
	}}
	seen := make(map[string]struct{})

	for len(queue) > 0 {
		pair := queue[0]
		queue = queue[1:]

		seenKey := pair.originalPath + "\x00" + pair.currentPath
		if _, ok := seen[seenKey]; ok {
			continue
		}
		seen[seenKey] = struct{}{}

		refFiles, ok := collectExternalReferenceFiles(pair.currentBytes)
		if !ok {
			return false
		}
		for _, refFile := range refFiles {
			nextOriginalPath := resolveReferenceFilePath(filepath.Dir(pair.originalPath), refFile)
			nextCurrentPath := resolveReferenceFilePath(filepath.Dir(pair.currentPath), refFile)

			originalRefBytes, originalErr := os.ReadFile(nextOriginalPath)
			currentRefBytes, currentErr := os.ReadFile(nextCurrentPath)
			if originalErr != nil || currentErr != nil || !bytes.Equal(originalRefBytes, currentRefBytes) {
				return false
			}
			queue = append(queue, refFilePair{
				originalPath: nextOriginalPath,
				currentPath:  nextCurrentPath,
				currentBytes: currentRefBytes,
			})
		}
	}
	return true
}

func resolveReferenceFilePath(baseDir string, refFile string) string {
	if filepath.IsAbs(refFile) {
		return filepath.Clean(refFile)
	}
	return filepath.Clean(filepath.Join(baseDir, refFile))
}

func collectExternalReferenceFiles(specBytes []byte) ([]string, bool) {
	var root yaml.Node
	if err := yaml.Unmarshal(specBytes, &root); err != nil {
		return nil, false
	}

	seen := make(map[string]struct{})
	var refs []string
	ok := collectExternalReferenceFilesFromNode(&root, seen, &refs)
	return refs, ok
}

func collectExternalReferenceFilesFromNode(node *yaml.Node, seen map[string]struct{}, refs *[]string) bool {
	if node == nil {
		return true
	}
	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			if !collectExternalReferenceFilesFromNode(child, seen, refs) {
				return false
			}
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]
			if keyNode != nil && keyNode.Value == "$ref" && valueNode != nil && valueNode.Kind == yaml.ScalarNode {
				ref := strings.TrimSpace(valueNode.Value)
				refFile, ok := externalReferenceFile(ref)
				if !ok {
					return false
				}
				if refFile != "" {
					if _, exists := seen[refFile]; !exists {
						seen[refFile] = struct{}{}
						*refs = append(*refs, refFile)
					}
				}
			}
			if !collectExternalReferenceFilesFromNode(valueNode, seen, refs) {
				return false
			}
		}
	case yaml.SequenceNode:
		for _, child := range node.Content {
			if !collectExternalReferenceFilesFromNode(child, seen, refs) {
				return false
			}
		}
	}
	return true
}

func externalReferenceFile(ref string) (string, bool) {
	if ref == "" || strings.HasPrefix(ref, "#") {
		return "", true
	}
	refFile, _, _ := strings.Cut(ref, "#")
	refFile = strings.TrimSpace(refFile)
	if refFile == "" {
		return "", true
	}
	if strings.Contains(refFile, "://") || strings.HasPrefix(refFile, "//") {
		return "", false
	}
	return refFile, true
}
