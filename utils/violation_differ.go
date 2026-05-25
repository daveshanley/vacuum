// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package utils

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/daveshanley/vacuum/model"
)

// violationKey identifies a violation across spec versions.
// Includes Message to distinguish rules that emit multiple distinct violations
// at the same path (e.g., info-contact-properties emits separate name/url/email
// failures all at $.info.contact).
type violationKey struct {
	RuleId  string
	Path    string
	Message string
}

// extractPath returns a stable JSONPath identity from a result.
// Some rules report a primary Path plus alternate Paths for the same resolved
// model. The primary can vary when external refs are resolved concurrently, so
// use the sorted unique set of known paths as the diff identity.
func extractPath(path string, paths []string) string {
	if len(paths) > 0 {
		candidates := make([]string, 0, len(paths)+1)
		seen := make(map[string]struct{}, len(paths)+1)
		if path != "" {
			seen[path] = struct{}{}
			candidates = append(candidates, path)
		}
		for _, candidate := range paths {
			if candidate == "" {
				continue
			}
			if _, ok := seen[candidate]; ok {
				continue
			}
			seen[candidate] = struct{}{}
			candidates = append(candidates, candidate)
		}
		sort.Strings(candidates)
		return strings.Join(candidates, "\x00")
	}
	if path != "" {
		return path
	}
	return ""
}

type canonicalOriginMapper struct {
	localRoot       string
	canonicalByPath map[string]string
}

func newCanonicalOriginMapperFromSpecPath(specPath string, canonicalByPath map[string]string, sharedParentDepth int) (canonicalOriginMapper, bool) {
	root, ok := originRootFromSpecPath(specPath)
	if !ok {
		return canonicalOriginMapper{}, false
	}
	return canonicalOriginMapper{
		localRoot:       ascendPath(root, sharedParentDepth),
		canonicalByPath: canonicalByPath,
	}, true
}

func newCanonicalOriginMappers(originalLocations, newLocations []string, originalSpecPath, newSpecPath string) (canonicalOriginMapper, canonicalOriginMapper) {
	originalPaths := normalizeOriginLocations(originalLocations)
	newPaths := normalizeOriginLocations(newLocations)
	originalCounts := suffixCounts(originalPaths)
	newCounts := suffixCounts(newPaths)
	originalInferred := inferCanonicalPaths(originalPaths, originalCounts, newCounts)
	newInferred := inferCanonicalPaths(newPaths, originalCounts, newCounts)

	originalRoot, originalOK := originRootFromSpecPath(originalSpecPath)
	newRoot, newOK := originRootFromSpecPath(newSpecPath)
	if originalOK && newOK {
		sharedParentDepth := maxInt(
			maxParentTraversalDepth(originalRoot, originalPaths),
			maxParentTraversalDepth(newRoot, newPaths),
		)
		originalMapper, _ := newCanonicalOriginMapperFromSpecPath(originalSpecPath, originalInferred, sharedParentDepth)
		newMapper, _ := newCanonicalOriginMapperFromSpecPath(newSpecPath, newInferred, sharedParentDepth)
		return originalMapper, newMapper
	}

	return canonicalOriginMapper{canonicalByPath: originalInferred},
		canonicalOriginMapper{canonicalByPath: newInferred}
}

func (m canonicalOriginMapper) canonicalLocation(location string) string {
	if location == "" || strings.Contains(location, "://") {
		return location
	}

	path, ok := normalizeLocalOriginPath(location)
	if !ok {
		return filepath.Clean(location)
	}
	if m.localRoot != "" {
		if rel, err := filepath.Rel(m.localRoot, path); err == nil && rel != "." && !isParentRelativePath(rel) {
			return filepath.ToSlash(rel)
		}
	}
	if canonical, ok := m.canonicalByPath[path]; ok {
		return canonical
	}
	return path
}

func collectOriginLocationsValues(results []model.RuleFunctionResult) []string {
	locations := make([]string, 0, len(results))
	for i := range results {
		if results[i].Origin != nil && results[i].Origin.AbsoluteLocation != "" {
			locations = append(locations, results[i].Origin.AbsoluteLocation)
		}
	}
	return locations
}

func collectOriginLocationsMixed(results []*model.RuleFunctionResult) []string {
	locations := make([]string, 0, len(results))
	for _, result := range results {
		if result != nil && result.Origin != nil && result.Origin.AbsoluteLocation != "" {
			locations = append(locations, result.Origin.AbsoluteLocation)
		}
	}
	return locations
}

func originRootFromSpecPath(specPath string) (string, bool) {
	if specPath == "" || specPath == "stdin" || strings.Contains(specPath, "://") {
		return "", false
	}
	abs, err := filepath.Abs(specPath)
	if err != nil {
		return "", false
	}
	return filepath.Dir(filepath.Clean(abs)), true
}

func maxParentTraversalDepth(root string, paths []string) int {
	maxDepth := 0
	for _, path := range paths {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			continue
		}
		if depth := parentTraversalDepth(rel); depth > maxDepth {
			maxDepth = depth
		}
	}
	return maxDepth
}

func parentTraversalDepth(rel string) int {
	if rel == "." || rel == "" {
		return 0
	}
	depth := 0
	for _, part := range strings.Split(filepath.Clean(rel), string(filepath.Separator)) {
		if part != ".." {
			break
		}
		depth++
	}
	return depth
}

func ascendPath(path string, depth int) string {
	root := filepath.Clean(path)
	for i := 0; i < depth; i++ {
		parent := filepath.Dir(root)
		if parent == root {
			return root
		}
		root = parent
	}
	return root
}

func isParentRelativePath(rel string) bool {
	return rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func normalizeLocalOriginPath(location string) (string, bool) {
	if location == "" || strings.Contains(location, "://") {
		return "", false
	}
	if abs, err := filepath.Abs(location); err == nil {
		return filepath.Clean(abs), true
	}
	return filepath.Clean(location), true
}

func normalizeOriginLocations(locations []string) []string {
	paths := make([]string, 0, len(locations))
	seen := make(map[string]struct{}, len(locations))
	for _, location := range locations {
		path, ok := normalizeLocalOriginPath(location)
		if !ok {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	return paths
}

func suffixCounts(paths []string) map[string]int {
	counts := make(map[string]int)
	for _, path := range paths {
		for _, suffix := range originPathSuffixes(path) {
			counts[suffix]++
		}
	}
	return counts
}

func inferCanonicalPaths(paths []string, originalCounts, newCounts map[string]int) map[string]string {
	canonical := make(map[string]string, len(paths))
	for _, path := range paths {
		for _, suffix := range originPathSuffixes(path) {
			if originalCounts[suffix] <= 1 && newCounts[suffix] <= 1 {
				canonical[path] = suffix
				break
			}
		}
		if canonical[path] == "" {
			canonical[path] = filepath.ToSlash(path)
		}
	}
	return canonical
}

func originPathSuffixes(path string) []string {
	clean := filepath.ToSlash(filepath.Clean(path))
	clean = strings.TrimLeft(clean, "/")
	if clean == "" || clean == "." {
		return nil
	}
	parts := strings.Split(clean, "/")
	start := len(parts) - 1
	if len(parts) > 1 {
		start = len(parts) - 2
	}
	suffixes := make([]string, 0, start+1)
	for i := start; i >= 0; i-- {
		suffixes = append(suffixes, strings.Join(parts[i:], "/"))
	}
	return suffixes
}

// extractIdentity returns the most stable identity available for a violation.
// Result paths can vary for resolved external references because the same source
// schema may be reachable through multiple root-document reference paths. Source
// origin is stable for that case, so prefer a private canonical file/line/column
// identity when it is known.
func extractIdentity(result model.RuleFunctionResult, originMapper canonicalOriginMapper) string {
	if result.Origin != nil {
		location := originMapper.canonicalLocation(result.Origin.AbsoluteLocation)
		line := result.Origin.Line
		column := result.Origin.Column
		if line == 0 && result.StartNode != nil {
			line = result.StartNode.Line
		}
		if column == 0 && result.StartNode != nil {
			column = result.StartNode.Column
		}
		if location != "" && line > 0 {
			return fmt.Sprintf("origin:%s:%d:%d", location, line, column)
		}
	}
	return "path:" + extractPath(result.Path, result.Paths)
}

// diffCore builds a count map from original keys and returns which new indices survive filtering.
func diffCore(originalKeys []violationKey, newKeys []violationKey) (kept []int, stats *ChangeFilterStats) {
	stats = &ChangeFilterStats{
		TotalResultsBefore:   len(newKeys),
		RulesPartialFiltered: make(map[string]int),
	}

	// Build occurrence count map from original results
	origCounts := make(map[violationKey]int, len(originalKeys))
	for _, k := range originalKeys {
		origCounts[k]++
	}

	// Track per-rule before/after counts
	ruleResultsBefore := make(map[string]int)
	ruleResultsAfter := make(map[string]int)

	kept = make([]int, 0, len(newKeys))
	for i, k := range newKeys {
		ruleResultsBefore[k.RuleId]++
		if origCounts[k] > 0 {
			// This violation existed in the original — suppress it
			origCounts[k]--
		} else {
			// New violation — keep it
			kept = append(kept, i)
			ruleResultsAfter[k.RuleId]++
		}
	}

	stats.TotalResultsAfter = len(kept)
	stats.ResultsDropped = stats.TotalResultsBefore - stats.TotalResultsAfter

	for ruleId, beforeCount := range ruleResultsBefore {
		afterCount := ruleResultsAfter[ruleId]
		dropped := beforeCount - afterCount
		if dropped > 0 {
			if afterCount == 0 {
				stats.RulesFullyFiltered = append(stats.RulesFullyFiltered, ruleId)
			} else {
				stats.RulesPartialFiltered[ruleId] = dropped
			}
		}
	}

	return kept, stats
}

// DiffViolationsValues compares original (value slice) and new (value slice) violations,
// returning only violations in new that don't exist in the original.
// Used by lint_cmd.go where both slices are values.
func DiffViolationsValues(original, new []model.RuleFunctionResult) ([]model.RuleFunctionResult, *ChangeFilterStats) {
	originalOrigins, newOrigins := newCanonicalOriginMappers(
		collectOriginLocationsValues(original),
		collectOriginLocationsValues(new),
		"",
		"",
	)

	return diffViolationsValuesWithMappers(original, new, originalOrigins, newOrigins)
}

func DiffViolationsValuesWithOriginBases(original, new []model.RuleFunctionResult, originalSpecPath, newSpecPath string) ([]model.RuleFunctionResult, *ChangeFilterStats) {
	originalOrigins, newOrigins := newCanonicalOriginMappers(
		collectOriginLocationsValues(original),
		collectOriginLocationsValues(new),
		originalSpecPath,
		newSpecPath,
	)

	return diffViolationsValuesWithMappers(original, new, originalOrigins, newOrigins)
}

func diffViolationsValuesWithMappers(original, new []model.RuleFunctionResult, originalOrigins, newOrigins canonicalOriginMapper) ([]model.RuleFunctionResult, *ChangeFilterStats) {
	originalKeys := make([]violationKey, len(original))
	for i := range original {
		originalKeys[i] = violationKey{
			RuleId:  original[i].RuleId,
			Path:    extractIdentity(original[i], originalOrigins),
			Message: original[i].Message,
		}
	}

	newKeys := make([]violationKey, len(new))
	for i := range new {
		newKeys[i] = violationKey{
			RuleId:  new[i].RuleId,
			Path:    extractIdentity(new[i], newOrigins),
			Message: new[i].Message,
		}
	}

	kept, stats := diffCore(originalKeys, newKeys)

	result := make([]model.RuleFunctionResult, len(kept))
	for i, idx := range kept {
		result[i] = new[idx]
	}
	return result, stats
}

// DiffViolationsMixed compares original (value slice from LintOriginalSpec) and new (pointer slice
// from report commands), returning only violations in new that don't exist in the original.
// Used by spectral-report, html-report, vacuum-report, and dashboard.
func DiffViolationsMixed(original []model.RuleFunctionResult, new []*model.RuleFunctionResult) ([]*model.RuleFunctionResult, *ChangeFilterStats) {
	originalOrigins, newOrigins := newCanonicalOriginMappers(
		collectOriginLocationsValues(original),
		collectOriginLocationsMixed(new),
		"",
		"",
	)

	return diffViolationsMixedWithMappers(original, new, originalOrigins, newOrigins)
}

func DiffViolationsMixedWithOriginBases(original []model.RuleFunctionResult, new []*model.RuleFunctionResult, originalSpecPath, newSpecPath string) ([]*model.RuleFunctionResult, *ChangeFilterStats) {
	originalOrigins, newOrigins := newCanonicalOriginMappers(
		collectOriginLocationsValues(original),
		collectOriginLocationsMixed(new),
		originalSpecPath,
		newSpecPath,
	)

	return diffViolationsMixedWithMappers(original, new, originalOrigins, newOrigins)
}

func diffViolationsMixedWithMappers(original []model.RuleFunctionResult, new []*model.RuleFunctionResult, originalOrigins, newOrigins canonicalOriginMapper) ([]*model.RuleFunctionResult, *ChangeFilterStats) {
	originalKeys := make([]violationKey, len(original))
	for i := range original {
		originalKeys[i] = violationKey{
			RuleId:  original[i].RuleId,
			Path:    extractIdentity(original[i], originalOrigins),
			Message: original[i].Message,
		}
	}

	newKeys := make([]violationKey, len(new))
	for i := range new {
		if new[i] == nil {
			continue
		}
		newKeys[i] = violationKey{
			RuleId:  new[i].RuleId,
			Path:    extractIdentity(*new[i], newOrigins),
			Message: new[i].Message,
		}
	}

	kept, stats := diffCore(originalKeys, newKeys)

	result := make([]*model.RuleFunctionResult, len(kept))
	for i, idx := range kept {
		result[i] = new[idx]
	}
	return result, stats
}
