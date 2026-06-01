// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package utils

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"go.yaml.in/yaml/v4"
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

type violationIdentity struct {
	key            violationKey
	paths          []string
	source         string
	sourceLine     int
	sourceColumn   int
	sourceFromRoot bool
}

type violationIdentityGroup struct {
	RuleId  string
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

func extractPathCandidates(path string, paths []string) []string {
	candidates := make([]string, 0, len(paths)+1)
	seen := make(map[string]struct{}, len(paths)+1)
	add := func(candidate string) {
		if candidate == "" {
			return
		}
		if _, ok := seen[candidate]; ok {
			return
		}
		seen[candidate] = struct{}{}
		candidates = append(candidates, candidate)
	}
	add(path)
	for _, candidate := range paths {
		add(candidate)
	}
	sort.Strings(candidates)
	return candidates
}

type canonicalOriginMapper struct {
	localRoot             string
	rootPath              string
	rootCanonicalLocation string
	canonicalByPath       map[string]string
	nodePathIndexes       map[*yaml.Node]*NodePathIndex
}

func newCanonicalOriginMapper(localRoot, rootPath, rootCanonicalLocation string, canonicalByPath map[string]string) canonicalOriginMapper {
	return canonicalOriginMapper{
		localRoot:             localRoot,
		rootPath:              rootPath,
		rootCanonicalLocation: rootCanonicalLocation,
		canonicalByPath:       canonicalByPath,
		nodePathIndexes:       make(map[*yaml.Node]*NodePathIndex),
	}
}

func newCanonicalOriginMapperFromSpecPath(specPath, rootCanonicalLocation string, canonicalByPath map[string]string, sharedParentDepth int) (canonicalOriginMapper, bool) {
	root, ok := originRootFromSpecPath(specPath)
	if !ok {
		return canonicalOriginMapper{}, false
	}
	rootPath, _ := normalizeLocalOriginPath(specPath)
	return newCanonicalOriginMapper(ascendPath(root, sharedParentDepth), rootPath, rootCanonicalLocation, canonicalByPath), true
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
		rootCanonicalLocation := "$root"
		originalMapper, _ := newCanonicalOriginMapperFromSpecPath(originalSpecPath, rootCanonicalLocation, originalInferred, sharedParentDepth)
		newMapper, _ := newCanonicalOriginMapperFromSpecPath(newSpecPath, rootCanonicalLocation, newInferred, sharedParentDepth)
		return originalMapper, newMapper
	}

	return newCanonicalOriginMapper("", "", "", originalInferred),
		newCanonicalOriginMapper("", "", "", newInferred)
}

func (m canonicalOriginMapper) canonicalLocation(location string) string {
	if location == "" || strings.Contains(location, "://") {
		return location
	}

	path, ok := normalizeLocalOriginPath(location)
	if !ok {
		return filepath.Clean(location)
	}
	if m.rootPath != "" && path == m.rootPath {
		return m.rootCanonicalLocation
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

func (m canonicalOriginMapper) isRootLocation(location string) bool {
	if m.rootPath == "" || location == "" || strings.Contains(location, "://") {
		return false
	}
	path, ok := normalizeLocalOriginPath(location)
	return ok && path == m.rootPath
}

func (m *canonicalOriginMapper) sourcePathIdentity(result model.RuleFunctionResult) string {
	if m == nil || result.Origin == nil || result.Origin.Index == nil {
		return ""
	}

	location := m.canonicalLocation(result.Origin.AbsoluteLocation)
	if location == "" {
		return ""
	}

	for _, node := range []*yaml.Node{
		result.Origin.Node,
		result.Origin.ValueNode,
		result.StartNode,
	} {
		if sourcePath := m.lookupOriginNodePath(result.Origin.Index.GetRootNode(), node); sourcePath != "" {
			return location + "#" + sourcePath
		}
	}
	return ""
}

func (m *canonicalOriginMapper) lookupOriginNodePath(root, node *yaml.Node) string {
	if root == nil || node == nil {
		return ""
	}
	pathIndex := m.nodePathIndex(root)
	if path, ok := pathIndex.Lookup(node); ok {
		return path
	}
	return ""
}

func (m *canonicalOriginMapper) nodePathIndex(root *yaml.Node) *NodePathIndex {
	if m.nodePathIndexes == nil {
		m.nodePathIndexes = make(map[*yaml.Node]*NodePathIndex)
	}
	if pathIndex, ok := m.nodePathIndexes[root]; ok {
		return pathIndex
	}
	pathIndex := BuildNodePathIndex(root)
	m.nodePathIndexes[root] = pathIndex
	return pathIndex
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
// JSONPath identity is the user-visible contract for --original filtering; line
// and column numbers are display metadata and can move when unrelated text is
// inserted earlier in the document. Origin is still useful for pathless results.
func extractIdentity(result model.RuleFunctionResult, originMapper *canonicalOriginMapper) string {
	if pathIdentity := extractPath(result.Path, result.Paths); pathIdentity != "" {
		return "path:" + pathIdentity
	}
	if sourcePath := originMapper.sourcePathIdentity(result); sourcePath != "" {
		return "source:" + sourcePath
	}
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
	return "path:"
}

func buildViolationIdentity(result model.RuleFunctionResult, originMapper *canonicalOriginMapper) violationIdentity {
	source := originMapper.sourcePathIdentity(result)
	sourceFromRoot, sourceLine, sourceColumn := false, 0, 0
	if result.Origin != nil {
		sourceFromRoot = originMapper.isRootLocation(result.Origin.AbsoluteLocation)
		sourceLine = result.Origin.Line
		sourceColumn = result.Origin.Column
		if sourceLine == 0 {
			sourceLine = result.Origin.LineValue
		}
		if sourceColumn == 0 {
			sourceColumn = result.Origin.ColumnValue
		}
	}
	return violationIdentity{
		key: violationKey{
			RuleId:  result.RuleId,
			Path:    extractIdentity(result, originMapper),
			Message: result.Message,
		},
		paths:          extractPathCandidates(result.Path, result.Paths),
		source:         source,
		sourceLine:     sourceLine,
		sourceColumn:   sourceColumn,
		sourceFromRoot: sourceFromRoot,
	}
}

// diffCore builds exact and aliased-path match state and returns which new indices survive filtering.
func diffCore(originalKeys []violationIdentity, newKeys []violationIdentity) (kept []int, stats *ChangeFilterStats) {
	stats = &ChangeFilterStats{
		TotalResultsBefore:   len(newKeys),
		RulesPartialFiltered: make(map[string]int),
	}

	// Track per-rule before/after counts
	ruleResultsBefore := make(map[string]int)
	ruleResultsAfter := make(map[string]int)

	originalAvailable := make([]bool, len(originalKeys))
	newMatched := make([]bool, len(newKeys))
	originalByKey := make(map[violationKey][]int, len(originalKeys))
	originalByGroup := make(map[violationIdentityGroup][]int, len(originalKeys))
	for i, k := range originalKeys {
		originalAvailable[i] = true
		originalByKey[k.key] = append(originalByKey[k.key], i)
		group := violationIdentityGroup{RuleId: k.key.RuleId, Message: k.key.Message}
		originalByGroup[group] = append(originalByGroup[group], i)
	}

	for i, k := range newKeys {
		if matchIndex := findExactOriginalViolationMatch(k, originalAvailable, originalByKey); matchIndex >= 0 {
			originalAvailable[matchIndex] = false
			newMatched[i] = true
		}
	}

	kept = make([]int, 0, len(newKeys))
	for i, k := range newKeys {
		ruleResultsBefore[k.key.RuleId]++
		if newMatched[i] {
			continue
		}
		if matchIndex := findAliasedOriginalViolationMatch(k, originalKeys, originalAvailable, originalByGroup); matchIndex >= 0 {
			// This violation existed in the original — suppress it.
			originalAvailable[matchIndex] = false
			continue
		}
		// New violation — keep it.
		kept = append(kept, i)
		ruleResultsAfter[k.key.RuleId]++
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

func findExactOriginalViolationMatch(
	newKey violationIdentity,
	originalAvailable []bool,
	originalByKey map[violationKey][]int,
) int {
	for _, idx := range originalByKey[newKey.key] {
		if originalAvailable[idx] {
			return idx
		}
	}
	return -1
}

func findAliasedOriginalViolationMatch(
	newKey violationIdentity,
	originalKeys []violationIdentity,
	originalAvailable []bool,
	originalByGroup map[violationIdentityGroup][]int,
) int {
	group := violationIdentityGroup{RuleId: newKey.key.RuleId, Message: newKey.key.Message}
	for _, idx := range originalByGroup[group] {
		if !originalAvailable[idx] {
			continue
		}
		if aliasedViolationIdentityMatches(originalKeys[idx], newKey) {
			return idx
		}
	}
	return -1
}

func aliasedViolationIdentityMatches(original, next violationIdentity) bool {
	if len(original.paths) == 0 || len(next.paths) == 0 {
		return false
	}
	if original.source != "" && next.source != "" {
		if original.source != next.source {
			return false
		}
		if original.sourceFromRoot && next.sourceFromRoot && rootSourcePositionMoved(original, next) {
			return true
		}
	}
	return pathCandidatesIntersect(original.paths, next.paths)
}

func rootSourcePositionMoved(original, next violationIdentity) bool {
	if original.sourceLine <= 0 || next.sourceLine <= 0 {
		return false
	}
	if original.sourceLine != next.sourceLine {
		return true
	}
	if original.sourceColumn <= 0 || next.sourceColumn <= 0 {
		return false
	}
	return original.sourceColumn != next.sourceColumn
}

func pathCandidatesIntersect(a, b []string) bool {
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] == b[j]:
			return true
		case a[i] < b[j]:
			i++
		default:
			j++
		}
	}
	return false
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
	originalKeys := make([]violationIdentity, len(original))
	for i := range original {
		originalKeys[i] = buildViolationIdentity(original[i], &originalOrigins)
	}

	newKeys := make([]violationIdentity, len(new))
	for i := range new {
		newKeys[i] = buildViolationIdentity(new[i], &newOrigins)
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
	originalKeys := make([]violationIdentity, len(original))
	for i := range original {
		originalKeys[i] = buildViolationIdentity(original[i], &originalOrigins)
	}

	newKeys := make([]violationIdentity, len(new))
	for i := range new {
		if new[i] == nil {
			continue
		}
		newKeys[i] = buildViolationIdentity(*new[i], &newOrigins)
	}

	kept, stats := diffCore(originalKeys, newKeys)

	result := make([]*model.RuleFunctionResult, len(kept))
	for i, idx := range kept {
		result[i] = new[idx]
	}
	return result, stats
}
