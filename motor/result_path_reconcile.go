package motor

import (
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/libopenapi/index"
	"go.yaml.in/yaml/v4"
)

type resultPathPosition struct {
	line   int
	column int
}

type resultPathCache struct {
	nodePaths          map[*yaml.Node]string
	positionPaths      map[resultPathPosition]string
	precisePositionMap map[resultPathPosition]string
}

func needsResultPathReconciliation(results []model.RuleFunctionResult) bool {
	for i := range results {
		if resultPathNeedsReconciliation(&results[i]) {
			return true
		}
	}
	return false
}

func resultPathNeedsReconciliation(result *model.RuleFunctionResult) bool {
	if result == nil {
		return false
	}
	return result.Path == "" || result.Path == "unknown" || strings.Contains(result.Path, "*")
}

func newResultPathCache(root *yaml.Node) *resultPathCache {
	cache := &resultPathCache{
		nodePaths:          make(map[*yaml.Node]string),
		positionPaths:      make(map[resultPathPosition]string),
		precisePositionMap: make(map[resultPathPosition]string),
	}
	cache.indexNode(root, "$")
	return cache
}

func (c *resultPathCache) reconcile(result *model.RuleFunctionResult) {
	if c == nil || result == nil || !resultPathNeedsReconciliation(result) {
		return
	}

	if path, found := c.canonicalPathForResult(result); found {
		result.Path = path
	}
}

func ruleUsesTerminalKeySelector(rule *model.Rule) bool {
	if rule == nil {
		return false
	}

	check := func(path string) bool {
		return strings.HasSuffix(strings.TrimSpace(path), "~")
	}

	switch given := rule.Given.(type) {
	case string:
		return check(given)
	case []string:
		for _, path := range given {
			if check(path) {
				return true
			}
		}
	case []interface{}:
		for _, raw := range given {
			if path, ok := raw.(string); ok && check(path) {
				return true
			}
		}
	}
	return false
}

func needsTerminalKeySelectorPathUpgrade(results []model.RuleFunctionResult) bool {
	for i := range results {
		if ruleUsesTerminalKeySelector(results[i].Rule) {
			return true
		}
	}
	return false
}

func (c *resultPathCache) canonicalPathForResult(result *model.RuleFunctionResult) (string, bool) {
	if c == nil || result == nil {
		return "", false
	}

	if result.Origin != nil {
		if path, found := c.lookupNodePath(result.Origin.Node); found {
			return path, true
		}
		if path, found := c.lookupNodePath(result.Origin.ValueNode); found {
			return path, true
		}
		if path, found := c.lookupPositionPath(result.Origin.Line, result.Origin.Column); found {
			return path, true
		}
		if path, found := c.lookupPositionPath(result.Origin.LineValue, result.Origin.ColumnValue); found {
			return path, true
		}
	}

	if path, found := c.lookupNodePath(result.StartNode); found {
		return path, true
	}
	if path, found := c.lookupPositionPathForNode(result.StartNode); found {
		return path, true
	}
	return "", false
}

func collapseAliasedResults(results []model.RuleFunctionResult, cache *resultPathCache) []model.RuleFunctionResult {
	if len(results) <= 1 {
		return results
	}

	groupedIndexes := make(map[string]int, len(results))
	collapsed := make([]model.RuleFunctionResult, 0, len(results))

	for i := range results {
		result := results[i]
		key, ok := aliasedResultKey(&result)
		if !ok {
			collapsed = append(collapsed, result)
			continue
		}

		if existingIndex, seen := groupedIndexes[key]; seen {
			mergeAliasedResult(&collapsed[existingIndex], &result, cache)
			continue
		}

		groupedIndexes[key] = len(collapsed)
		collapsed = append(collapsed, result)
	}

	return collapsed
}

func completeAliasedResultPathsFromGiven(results []model.RuleFunctionResult, root *yaml.Node, rolodex *index.Rolodex, expandedAliases map[string][]string) {
	if len(results) == 0 || root == nil || !needsAliasedResultPathCompletion(results) {
		return
	}

	candidateCache := make(map[string]*resultPathCandidateIndex)
	givenPathCache := make(map[*model.Rule][]string)
	originCache := make(map[*yaml.Node]*index.NodeOrigin)

	for i := range results {
		result := &results[i]
		if !shouldCompleteAliasedResultPaths(result) {
			continue
		}

		var aliasPaths []string
		givenPaths, ok := givenPathCache[result.Rule]
		if !ok {
			givenPaths, _ = resolveRuleGivenPaths(result.Rule, expandedAliases)
			givenPathCache[result.Rule] = givenPaths
		}
		for _, givenPath := range givenPaths {
			candidateIndex, ok := candidateCache[givenPath]
			if !ok {
				candidates, truncated := collectResultPathCandidates(root, givenPath)
				if truncated {
					candidateCache[givenPath] = nil
					continue
				}
				candidateIndex = newResultPathCandidateIndex(candidates)
				candidateCache[givenPath] = candidateIndex
			}
			if candidateIndex == nil {
				continue
			}
			aliasPaths = append(aliasPaths, candidateIndex.matchingPaths(result, rolodex, originCache)...)
		}

		mergeResultPathCandidates(result, aliasPaths)
	}
}

func needsAliasedResultPathCompletion(results []model.RuleFunctionResult) bool {
	for i := range results {
		if shouldCompleteAliasedResultPaths(&results[i]) {
			return true
		}
	}
	return false
}

func shouldCompleteAliasedResultPaths(result *model.RuleFunctionResult) bool {
	if result == nil || result.Rule == nil || result.StartNode == nil {
		return false
	}
	if len(result.Paths) <= 1 && !resultPathNeedsReconciliation(result) {
		return false
	}
	_, ok := aliasedResultKey(result)
	return ok
}

func resultGivenPaths(rule *model.Rule) []string {
	if rule == nil {
		return nil
	}

	switch given := rule.Given.(type) {
	case string:
		return []string{given}
	case []string:
		return given
	case []interface{}:
		paths := make([]string, 0, len(given))
		for _, item := range given {
			if path, ok := item.(string); ok {
				paths = append(paths, path)
			}
		}
		return paths
	default:
		return nil
	}
}

type resultPathCandidate struct {
	path string
	node *yaml.Node
}

type resultPathCandidateIndex struct {
	byNode     map[*yaml.Node][]resultPathCandidate
	byPosition map[resultPathPosition][]resultPathCandidate
}

type resultReferenceAliasKey struct {
	index *index.SpecIndex
	path  string
}

const maxResultReferenceAliasDepth = 16

func newResultPathCandidateIndex(candidates []resultPathCandidate) *resultPathCandidateIndex {
	candidateIndex := &resultPathCandidateIndex{
		byNode:     make(map[*yaml.Node][]resultPathCandidate, len(candidates)),
		byPosition: make(map[resultPathPosition][]resultPathCandidate, len(candidates)),
	}
	for _, candidate := range candidates {
		node := resultPathNodeAlias(candidate.node)
		if node == nil {
			continue
		}
		candidate.node = node
		candidateIndex.byNode[node] = append(candidateIndex.byNode[node], candidate)
		if node.Line > 0 && node.Column > 0 {
			position := resultPathPosition{line: node.Line, column: node.Column}
			candidateIndex.byPosition[position] = append(candidateIndex.byPosition[position], candidate)
		}
	}
	return candidateIndex
}

func (c *resultPathCandidateIndex) matchingPaths(result *model.RuleFunctionResult, rolodex *index.Rolodex, originCache map[*yaml.Node]*index.NodeOrigin) []string {
	if c == nil || result == nil {
		return nil
	}

	var paths []string
	seen := make(map[string]struct{})
	addCandidate := func(candidate resultPathCandidate) {
		if candidate.path == "" {
			return
		}
		if _, ok := seen[candidate.path]; ok {
			return
		}
		if !resultPathCandidateMatchesResult(candidate.node, result, rolodex, originCache) {
			return
		}
		seen[candidate.path] = struct{}{}
		paths = append(paths, candidate.path)
	}

	if startNode := resultPathNodeAlias(result.StartNode); startNode != nil {
		for _, candidate := range c.byNode[startNode] {
			addCandidate(candidate)
		}
	}
	line, column := resultPathResultLineColumn(result)
	if line > 0 && column > 0 {
		for _, candidate := range c.byPosition[resultPathPosition{line: line, column: column}] {
			addCandidate(candidate)
		}
	}
	return paths
}

func completeAliasedResultPathsFromReferences(results []model.RuleFunctionResult, rolodex *index.Rolodex) {
	if len(results) == 0 || rolodex == nil || rolodex.GetRootIndex() == nil || rolodex.GetRootIndex().GetRootNode() == nil {
		return
	}

	pathIndexes := make(map[*index.SpecIndex]*vacuumUtils.NodePathIndex)
	aliasCache := make(map[resultReferenceAliasKey][]string)

	for i := range results {
		result := &results[i]
		if !shouldCompleteAliasedResultPathsFromReferences(result) {
			continue
		}

		targetIndex, targetPath := targetPathForResult(result, rolodex, pathIndexes)
		if targetIndex == nil || targetPath == "" {
			continue
		}
		if targetIndex == rolodex.GetRootIndex() {
			continue
		}

		cacheKey := resultReferenceAliasKey{index: targetIndex, path: targetPath}
		aliasPaths, ok := aliasCache[cacheKey]
		if !ok {
			aliasPaths = expandResultReferenceAliasPaths(rolodex.GetRootIndex(), targetIndex, targetPath, pathIndexes)
			aliasCache[cacheKey] = aliasPaths
		}
		if len(aliasPaths) <= 1 {
			continue
		}

		mergeResultPathCandidates(result, applyResultPathSuffix(result, aliasPaths))
	}
}

func shouldCompleteAliasedResultPathsFromReferences(result *model.RuleFunctionResult) bool {
	if result == nil || result.Rule == nil || result.StartNode == nil {
		return false
	}
	if len(result.Paths) <= 1 && !resultPathNeedsReconciliation(result) {
		return false
	}
	if !ruleUsesRecursiveDescent(result.Rule) {
		return false
	}
	_, ok := aliasedResultKey(result)
	return ok
}

func ruleUsesRecursiveDescent(rule *model.Rule) bool {
	for _, givenPath := range resultGivenPaths(rule) {
		if strings.Contains(givenPath, "$..") {
			return true
		}
	}
	return false
}

func targetPathForResult(
	result *model.RuleFunctionResult,
	rolodex *index.Rolodex,
	pathIndexes map[*index.SpecIndex]*vacuumUtils.NodePathIndex,
) (*index.SpecIndex, string) {
	if result == nil || rolodex == nil {
		return nil, ""
	}

	origin := result.Origin
	if origin == nil && result.StartNode != nil {
		origin = rolodex.FindNodeOrigin(result.StartNode)
	}
	if origin == nil || origin.Index == nil {
		return nil, ""
	}

	pathIndex := resultPathIndexForSpec(origin.Index, pathIndexes)
	if path, ok := pathIndex.Lookup(origin.Node); ok && path != "" {
		return origin.Index, path
	}
	if path, ok := pathIndex.Lookup(origin.ValueNode); ok && path != "" {
		return origin.Index, path
	}
	if path, ok := pathIndex.Lookup(result.StartNode); ok && path != "" {
		return origin.Index, path
	}
	if origin.Line > 0 && origin.Column > 0 {
		if node, found := origin.Index.GetNode(origin.Line, origin.Column); found {
			if path, ok := pathIndex.Lookup(node); ok && path != "" {
				return origin.Index, path
			}
		}
	}
	if origin.LineValue > 0 && origin.ColumnValue > 0 {
		if node, found := origin.Index.GetNode(origin.LineValue, origin.ColumnValue); found {
			if path, ok := pathIndex.Lookup(node); ok && path != "" {
				return origin.Index, path
			}
		}
	}
	return nil, ""
}

func resultPathIndexForSpec(specIndex *index.SpecIndex, pathIndexes map[*index.SpecIndex]*vacuumUtils.NodePathIndex) *vacuumUtils.NodePathIndex {
	if specIndex == nil {
		return nil
	}
	if pathIndex, ok := pathIndexes[specIndex]; ok {
		return pathIndex
	}
	pathIndex := vacuumUtils.BuildNodePathIndex(specIndex.GetRootNode())
	pathIndexes[specIndex] = pathIndex
	return pathIndex
}

func expandResultReferenceAliasPaths(
	sourceIndex *index.SpecIndex,
	targetIndex *index.SpecIndex,
	targetPath string,
	pathIndexes map[*index.SpecIndex]*vacuumUtils.NodePathIndex,
) []string {
	if sourceIndex == nil || targetIndex == nil || targetPath == "" {
		return nil
	}

	sourcePathIndex := resultPathIndexForSpec(sourceIndex, pathIndexes)
	targetPathIndex := resultPathIndexForSpec(targetIndex, pathIndexes)
	if sourcePathIndex == nil || targetPathIndex == nil {
		return nil
	}

	var paths []string
	for _, ref := range sourceIndex.GetAllSequencedReferences() {
		if ref == nil || ref.Node == nil || ref.Path == "" {
			continue
		}
		if !referenceTargetsIndex(ref, targetIndex) {
			continue
		}
		sourcePath, ok := sourcePathIndex.Lookup(ref.Node)
		if !ok || sourcePath == "" {
			continue
		}
		paths = append(paths, expandResultReferenceAliasPath(
			ref.Path,
			sourcePath,
			targetPath,
			targetIndex.GetAllSequencedReferences(),
			targetPathIndex,
			targetIndex,
			nil,
			0,
		)...)
	}
	return uniqueSortedResultPaths(paths)
}

func expandResultReferenceAliasPath(
	currentTargetPath string,
	currentSourcePath string,
	targetPath string,
	targetReferences []*index.Reference,
	targetPathIndex *vacuumUtils.NodePathIndex,
	targetIndex *index.SpecIndex,
	seen map[string]struct{},
	depth int,
) []string {
	if currentTargetPath == "" || currentSourcePath == "" || targetPath == "" {
		return nil
	}
	if seen == nil {
		seen = make(map[string]struct{})
	}

	seenKey := currentTargetPath + "\x00" + currentSourcePath
	if _, ok := seen[seenKey]; ok {
		return nil
	}
	seen[seenKey] = struct{}{}
	defer delete(seen, seenKey)

	var paths []string
	if suffix, ok := trimAliasPathPrefix(targetPath, currentTargetPath); ok {
		paths = append(paths, canonicalizeResultAliasPath(currentSourcePath+suffix))
	}
	if depth >= maxResultReferenceAliasDepth {
		return paths
	}

	for _, nestedRef := range targetReferences {
		if nestedRef == nil || nestedRef.Node == nil || nestedRef.Path == "" {
			continue
		}
		if !referenceTargetsIndex(nestedRef, targetIndex) {
			continue
		}

		nestedSourcePath, ok := targetPathIndex.Lookup(nestedRef.Node)
		if !ok || nestedSourcePath == "" {
			continue
		}
		nestedSuffix, ok := trimAliasPathPrefix(nestedSourcePath, currentTargetPath)
		if !ok {
			continue
		}

		nextSourcePath := canonicalizeResultAliasPath(currentSourcePath + nestedSuffix)
		paths = append(paths, expandResultReferenceAliasPath(
			nestedRef.Path,
			nextSourcePath,
			targetPath,
			targetReferences,
			targetPathIndex,
			targetIndex,
			seen,
			depth+1,
		)...)
	}

	return paths
}

func referenceTargetsIndex(ref *index.Reference, targetIndex *index.SpecIndex) bool {
	if ref == nil || targetIndex == nil {
		return false
	}
	if strings.HasPrefix(ref.FullDefinition, "#") {
		return ref.Index == targetIndex
	}

	targetDocumentPath := targetIndex.GetSpecAbsolutePath()
	if targetDocumentPath == "" || ref.FullDefinition == "" {
		return true
	}
	if ref.RemoteLocation == targetDocumentPath {
		return true
	}
	if !strings.HasPrefix(ref.FullDefinition, targetDocumentPath) {
		return false
	}
	return len(ref.FullDefinition) == len(targetDocumentPath) ||
		ref.FullDefinition[len(targetDocumentPath)] == '#'
}

func applyResultPathSuffix(result *model.RuleFunctionResult, aliasPaths []string) []string {
	if result == nil || len(aliasPaths) == 0 {
		return aliasPaths
	}

	suffix := resultPathSuffix(result.Path, aliasPaths)
	if suffix == "" {
		for _, path := range result.Paths {
			suffix = resultPathSuffix(path, aliasPaths)
			if suffix != "" {
				break
			}
		}
	}
	if suffix == "" {
		return aliasPaths
	}

	paths := make([]string, 0, len(aliasPaths))
	for _, path := range aliasPaths {
		paths = append(paths, path+suffix)
	}
	return paths
}

func resultPathSuffix(path string, aliasPaths []string) string {
	if path == "" {
		return ""
	}
	for _, aliasPath := range aliasPaths {
		if len(path) <= len(aliasPath) {
			continue
		}
		if suffix, ok := trimAliasPathPrefix(path, aliasPath); ok {
			return suffix
		}
	}
	return ""
}

func uniqueSortedResultPaths(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(paths))
	unique := make([]string, 0, len(paths))
	for _, path := range paths {
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		unique = append(unique, path)
	}
	sort.Strings(unique)
	return unique
}

func trimAliasPathPrefix(path, prefix string) (string, bool) {
	if resultPathHasPrefix(path, prefix) {
		return strings.TrimPrefix(path, prefix), true
	}

	normalizedPath := normalizeSimpleBracketResultPath(path)
	normalizedPrefix := normalizeSimpleBracketResultPath(prefix)
	if !resultPathHasPrefix(normalizedPath, normalizedPrefix) {
		return "", false
	}
	return strings.TrimPrefix(normalizedPath, normalizedPrefix), true
}

func normalizeSimpleBracketResultPath(path string) string {
	var b strings.Builder
	b.Grow(len(path))
	for i := 0; i < len(path); {
		if i+3 < len(path) && path[i] == '[' && (path[i+1] == '\'' || path[i+1] == '"') {
			quote := path[i+1]
			end := i + 2
			for end < len(path) && path[end] != quote {
				end++
			}
			if end+1 < len(path) && path[end+1] == ']' {
				key := path[i+2 : end]
				if isSimpleResultPathKey(key) {
					b.WriteByte('.')
					b.WriteString(key)
					i = end + 2
					continue
				}
			}
		}
		b.WriteByte(path[i])
		i++
	}
	return b.String()
}

func canonicalizeResultAliasPath(path string) string {
	for _, marker := range []string{".properties.", ".patternProperties."} {
		for {
			idx := strings.Index(path, marker)
			if idx < 0 {
				break
			}
			keyStart := idx + len(marker)
			keyEnd := keyStart
			for keyEnd < len(path) && path[keyEnd] != '.' && path[keyEnd] != '[' {
				keyEnd++
			}
			if keyEnd == keyStart {
				break
			}
			key := path[keyStart:keyEnd]
			path = path[:idx+len(marker)-1] + "['" + key + "']" + path[keyEnd:]
		}
	}
	return path
}

func resultPathHasPrefix(path, prefix string) bool {
	if path == prefix {
		return true
	}
	if prefix == "" || len(path) <= len(prefix) || !strings.HasPrefix(path, prefix) {
		return false
	}
	next := path[len(prefix)]
	return next == '.' || next == '['
}

type resultPathStepKind int

const (
	resultPathStepName resultPathStepKind = iota
	resultPathStepWildcard
	resultPathStepIndex
	resultPathStepUnion
)

type resultPathStep struct {
	kind  resultPathStepKind
	name  string
	names []string
	index int
}

// maxResultPathCandidates bounds optional alias-completion walks so broad
// wildcard selectors cannot dominate rule execution.
const maxResultPathCandidates = 65536

func collectResultPathCandidates(root *yaml.Node, givenPath string) ([]resultPathCandidate, bool) {
	steps, ok := parseResultPathSteps(givenPath)
	if !ok {
		return nil, false
	}

	root = resultPathNodeAlias(root)
	if root != nil && root.Kind == yaml.DocumentNode && len(root.Content) == 1 {
		root = resultPathNodeAlias(root.Content[0])
	}
	if root == nil {
		return nil, false
	}

	candidates := make([]resultPathCandidate, 0, resultPathCandidateCapacityHint(steps))
	if !walkResultPathCandidates(root, "$", steps, &candidates) {
		return nil, true
	}
	return candidates, false
}

func resultPathCandidateCapacityHint(steps []resultPathStep) int {
	hint := 8
	for _, step := range steps {
		multiplier := 1
		switch step.kind {
		case resultPathStepWildcard:
			multiplier = 8
		case resultPathStepUnion:
			if len(step.names) > 0 {
				multiplier = len(step.names)
			}
		}
		if hint > maxResultPathCandidates/multiplier {
			return maxResultPathCandidates
		}
		hint *= multiplier
	}
	return hint
}

// parseResultPathSteps intentionally supports only deterministic child selectors
// needed for result-path completion: names, quoted names, indexes, wildcards,
// and simple key unions. Recursive descent, filters, and expressions are left
// to the rule engine and skip this optional completion path.
func parseResultPathSteps(path string) ([]resultPathStep, bool) {
	if path == "" || path[0] != '$' {
		return nil, false
	}

	steps := make([]resultPathStep, 0, 8)
	for i := 1; i < len(path); {
		switch path[i] {
		case '.':
			i++
			if i >= len(path) || path[i] == '.' {
				return nil, false
			}
			if path[i] == '*' {
				steps = append(steps, resultPathStep{kind: resultPathStepWildcard})
				i++
				continue
			}
			start := i
			for i < len(path) && path[i] != '.' && path[i] != '[' {
				switch path[i] {
				case '?', '(', ')', ',', ' ':
					return nil, false
				}
				i++
			}
			if start == i {
				return nil, false
			}
			steps = append(steps, resultPathStep{kind: resultPathStepName, name: path[start:i]})
		case '[':
			step, next, ok := parseResultPathBracketStep(path, i)
			if !ok {
				return nil, false
			}
			steps = append(steps, step)
			i = next
		default:
			return nil, false
		}
	}
	return steps, true
}

func parseResultPathBracketStep(path string, start int) (resultPathStep, int, bool) {
	if start+2 >= len(path) {
		return resultPathStep{}, 0, false
	}
	if path[start+1] == '*' && path[start+2] == ']' {
		return resultPathStep{kind: resultPathStepWildcard}, start + 3, true
	}
	if path[start+1] == '\'' || path[start+1] == '"' {
		return parseResultPathQuotedBracketStep(path, start)
	}

	i := start + 1
	for i < len(path) && path[i] != ']' {
		i++
	}
	if i >= len(path) || i == start+1 {
		return resultPathStep{}, 0, false
	}
	token := path[start+1 : i]
	if idx, err := strconv.Atoi(token); err == nil {
		return resultPathStep{kind: resultPathStepIndex, index: idx}, i + 1, true
	}
	names := parseResultPathKeyUnion(token)
	if len(names) == 0 {
		return resultPathStep{}, 0, false
	}
	if len(names) == 1 {
		return resultPathStep{kind: resultPathStepName, name: names[0]}, i + 1, true
	}
	return resultPathStep{kind: resultPathStepUnion, names: names}, i + 1, true
}

func parseResultPathQuotedBracketStep(path string, start int) (resultPathStep, int, bool) {
	names := make([]string, 0, 2)
	i := start + 1
	for {
		for i < len(path) && path[i] == ' ' {
			i++
		}
		if i >= len(path) || (path[i] != '\'' && path[i] != '"') {
			return resultPathStep{}, 0, false
		}

		quote := path[i]
		i++
		nameStart := i
		for i < len(path) && path[i] != quote {
			i++
		}
		if i >= len(path) || i == nameStart {
			return resultPathStep{}, 0, false
		}
		names = append(names, path[nameStart:i])
		i++

		for i < len(path) && path[i] == ' ' {
			i++
		}
		if i >= len(path) {
			return resultPathStep{}, 0, false
		}
		if path[i] == ']' {
			if len(names) == 1 {
				return resultPathStep{kind: resultPathStepName, name: names[0]}, i + 1, true
			}
			return resultPathStep{kind: resultPathStepUnion, names: names}, i + 1, true
		}
		if path[i] != ',' {
			return resultPathStep{}, 0, false
		}
		i++
	}
}

func parseResultPathKeyUnion(token string) []string {
	parts := strings.Split(token, ",")
	names := make([]string, 0, len(parts))
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" || strings.ContainsAny(name, "[]'\"()") {
			return nil
		}
		names = append(names, name)
	}
	return names
}

func walkResultPathCandidates(node *yaml.Node, path string, steps []resultPathStep, candidates *[]resultPathCandidate) bool {
	node = resultPathNodeAlias(node)
	if node == nil {
		return true
	}
	if len(*candidates) >= maxResultPathCandidates {
		return false
	}
	if len(steps) == 0 {
		*candidates = append(*candidates, resultPathCandidate{path: path, node: node})
		return len(*candidates) < maxResultPathCandidates
	}

	step := steps[0]
	remaining := steps[1:]
	switch step.kind {
	case resultPathStepName:
		if node.Kind != yaml.MappingNode {
			return true
		}
		for i := 0; i+1 < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]
			if keyNode != nil && keyNode.Value == step.name {
				childPath := appendCollectedResultPathSegment(path, step.name)
				return walkResultPathCandidates(valueNode, childPath, remaining, candidates)
			}
		}
	case resultPathStepWildcard:
		switch node.Kind {
		case yaml.MappingNode:
			for i := 0; i+1 < len(node.Content); i += 2 {
				keyNode := node.Content[i]
				valueNode := node.Content[i+1]
				if keyNode == nil {
					continue
				}
				childPath := appendCollectedResultPathSegment(path, keyNode.Value)
				if !walkResultPathCandidates(valueNode, childPath, remaining, candidates) {
					return false
				}
			}
		case yaml.SequenceNode:
			for i, child := range node.Content {
				childPath := appendResultPathIndex(path, i)
				if !walkResultPathCandidates(child, childPath, remaining, candidates) {
					return false
				}
			}
		}
	case resultPathStepIndex:
		if node.Kind != yaml.SequenceNode || step.index < 0 || step.index >= len(node.Content) {
			return true
		}
		childPath := appendResultPathIndex(path, step.index)
		return walkResultPathCandidates(node.Content[step.index], childPath, remaining, candidates)
	case resultPathStepUnion:
		if node.Kind != yaml.MappingNode {
			return true
		}
		for _, name := range step.names {
			for i := 0; i+1 < len(node.Content); i += 2 {
				keyNode := node.Content[i]
				valueNode := node.Content[i+1]
				if keyNode != nil && keyNode.Value == name {
					childPath := appendCollectedResultPathSegment(path, name)
					if !walkResultPathCandidates(valueNode, childPath, remaining, candidates) {
						return false
					}
					break
				}
			}
		}
	}
	return true
}

func resultPathNodeAlias(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.AliasNode {
		return node.Alias
	}
	return node
}

func appendCollectedResultPathSegment(basePath, key string) string {
	if resultPathShouldBracketCollectedSegment(basePath) {
		return basePath + "['" + key + "']"
	}
	return appendResultPathSegment(basePath, key)
}

func resultPathShouldBracketCollectedSegment(basePath string) bool {
	switch {
	case strings.HasSuffix(basePath, ".properties"),
		strings.HasSuffix(basePath, ".patternProperties"),
		strings.HasSuffix(basePath, ".schemas"),
		strings.HasSuffix(basePath, ".responses"),
		strings.HasSuffix(basePath, ".parameters"),
		strings.HasSuffix(basePath, ".requestBodies"),
		strings.HasSuffix(basePath, ".headers"),
		strings.HasSuffix(basePath, ".securitySchemes"),
		strings.HasSuffix(basePath, ".examples"),
		strings.HasSuffix(basePath, ".links"),
		strings.HasSuffix(basePath, ".callbacks"),
		strings.HasSuffix(basePath, ".pathItems"):
		return true
	default:
		return false
	}
}

func resultPathCandidateMatchesResult(candidate *yaml.Node, result *model.RuleFunctionResult, rolodex *index.Rolodex, originCache map[*yaml.Node]*index.NodeOrigin) bool {
	if result == nil {
		return false
	}
	candidate = resultPathNodeAlias(candidate)
	startNode := resultPathNodeAlias(result.StartNode)
	if candidate == nil {
		return false
	}
	if candidate == startNode {
		return true
	}

	line, column := resultPathResultLineColumn(result)
	if line <= 0 || column <= 0 || candidate.Line != line || candidate.Column != column {
		return false
	}

	if result.Origin == nil || result.Origin.AbsoluteLocation == "" || rolodex == nil {
		return true
	}

	origin, ok := originCache[candidate]
	if !ok {
		origin = rolodex.FindNodeOrigin(candidate)
		originCache[candidate] = origin
	}
	if origin == nil {
		return true
	}

	return sameResultPathLocation(result.Origin.AbsoluteLocation, origin.AbsoluteLocation) ||
		sameResultPathLocation(result.Origin.AbsoluteLocation, origin.AbsoluteLocationValue)
}

func resultPathResultLineColumn(result *model.RuleFunctionResult) (int, int) {
	if result == nil {
		return 0, 0
	}
	if result.Origin != nil {
		if result.Origin.Line > 0 && result.Origin.Column > 0 {
			return result.Origin.Line, result.Origin.Column
		}
		if result.Origin.LineValue > 0 && result.Origin.ColumnValue > 0 {
			return result.Origin.LineValue, result.Origin.ColumnValue
		}
	}
	if result.StartNode != nil {
		return result.StartNode.Line, result.StartNode.Column
	}
	return 0, 0
}

func sameResultPathLocation(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	if strings.Contains(a, "://") || strings.Contains(b, "://") {
		return strings.TrimRight(a, "/") == strings.TrimRight(b, "/")
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

func mergeResultPathCandidates(result *model.RuleFunctionResult, candidatePaths []string) {
	if result == nil || len(candidatePaths) == 0 {
		return
	}
	needsReconciliation := resultPathNeedsReconciliation(result)
	if !needsReconciliation && !resultPathCandidatesOverlap(result, candidatePaths) {
		return
	}

	mergedResult := model.RuleFunctionResult{Path: result.Path}
	if needsReconciliation {
		mergedResult.Path = ""
	}
	mergedResult.Paths = make([]string, 0, len(result.Paths)+len(candidatePaths))
	mergedResult.Paths = append(mergedResult.Paths, result.Paths...)
	mergedResult.Paths = append(mergedResult.Paths, candidatePaths...)
	canonicalPath := result.Path
	if uniqueResultPathCount(canonicalPath, mergedResult.Paths) > 1 && !strings.HasPrefix(canonicalPath, "$.components.") {
		canonicalPath = ""
	}

	paths := buildMergedResultPaths(canonicalPath, &mergedResult)
	if len(paths) > 0 {
		result.Path = paths[0]
	}
	if len(paths) > 1 {
		result.Paths = paths
	} else {
		result.Paths = nil
	}
}

func resultPathCandidatesOverlap(result *model.RuleFunctionResult, candidatePaths []string) bool {
	if result == nil {
		return false
	}

	if result.Path != "" {
		for _, candidate := range candidatePaths {
			if result.Path == candidate {
				return true
			}
		}
	}
	for _, existing := range result.Paths {
		for _, candidate := range candidatePaths {
			if existing == candidate {
				return true
			}
		}
	}
	return false
}

func uniqueResultPathCount(path string, paths []string) int {
	seen := make(map[string]struct{}, len(paths)+1)
	if path != "" {
		seen[path] = struct{}{}
	}
	for _, candidate := range paths {
		if candidate != "" {
			seen[candidate] = struct{}{}
		}
	}
	return len(seen)
}

func aliasedResultKey(result *model.RuleFunctionResult) (string, bool) {
	if result == nil {
		return "", false
	}

	ruleID := result.RuleId
	if ruleID == "" && result.Rule != nil {
		ruleID = result.Rule.Id
	}

	if result.Origin != nil && result.Origin.AbsoluteLocation != "" && result.Origin.Line > 0 && result.Origin.Column > 0 {
		return ruleID + "\x00" + result.Message + "\x00" + result.Origin.AbsoluteLocation + "\x00" +
			strconv.Itoa(result.Origin.Line) + "\x00" + strconv.Itoa(result.Origin.Column), true
	}

	if result.StartNode != nil && result.StartNode.Line > 0 && result.StartNode.Column > 0 {
		return ruleID + "\x00" + result.Message + "\x00" +
			strconv.Itoa(result.StartNode.Line) + "\x00" + strconv.Itoa(result.StartNode.Column), true
	}

	return "", false
}

func mergeAliasedResult(primary, duplicate *model.RuleFunctionResult, cache *resultPathCache) {
	if primary == nil || duplicate == nil {
		return
	}

	canonicalPath := primary.Path
	if cache != nil {
		if path, found := cache.canonicalPathForResult(primary); found {
			canonicalPath = path
		} else if path, found := cache.canonicalPathForResult(duplicate); found {
			canonicalPath = path
		}
	}

	paths := buildMergedResultPaths(canonicalPath, primary, duplicate)
	if len(paths) > 0 {
		primary.Path = paths[0]
	}

	if len(paths) > 1 {
		primary.Paths = paths
	} else {
		primary.Paths = nil
	}

	if primary.Origin == nil && duplicate.Origin != nil {
		primary.Origin = duplicate.Origin
	}
}

func upgradeTerminalKeySelectorPaths(results []model.RuleFunctionResult, cache *resultPathCache) {
	for i := range results {
		result := &results[i]
		if result == nil || !ruleUsesTerminalKeySelector(result.Rule) || result.StartNode == nil || result.StartNode.Value == "" {
			continue
		}
		if cache != nil {
			if precisePath, found := cache.lookupPrecisePositionPathForNode(result.StartNode); found {
				result.Path = precisePath
				continue
			}
		}
		if !hasTerminalKeyPathSegment(result.Path, result.StartNode.Value) &&
			result.Path != "" && result.Path != "unknown" && !strings.Contains(result.Path, "*") {
			result.Path = appendResultPathSegment(result.Path, result.StartNode.Value)
		}
	}
}

func hasTerminalKeyPathSegment(path, key string) bool {
	if path == "" || key == "" {
		return false
	}
	return strings.HasSuffix(path, "."+key) || strings.HasSuffix(path, "['"+key+"']")
}

func buildMergedResultPaths(canonicalPath string, results ...*model.RuleFunctionResult) []string {
	seen := make(map[string]struct{}, len(results)*2+1)
	candidates := make([]string, 0, len(results)*2+1)

	addCandidate := func(path string) {
		if path == "" {
			return
		}
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		candidates = append(candidates, path)
	}

	if canonicalPath != "" {
		addCandidate(canonicalPath)
	}

	for _, result := range results {
		if result == nil {
			continue
		}
		addCandidate(result.Path)
		for _, path := range result.Paths {
			addCandidate(path)
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	sort.Strings(candidates)
	primaryPath := selectPrimaryResultPath(canonicalPath, candidates)
	merged := make([]string, 0, len(candidates))
	merged = append(merged, primaryPath)
	for _, path := range candidates {
		if path == primaryPath {
			continue
		}
		if strings.HasPrefix(primaryPath, "$.components.") && strings.HasPrefix(path, "$.components.") {
			continue
		}
		if isAncestorJSONPath(path, primaryPath) {
			continue
		}
		merged = append(merged, path)
	}
	return merged
}

func selectPrimaryResultPath(canonicalPath string, candidates []string) string {
	longestComponentPath := ""
	for _, path := range candidates {
		if !strings.HasPrefix(path, "$.components.") {
			continue
		}
		if len(path) > len(longestComponentPath) {
			longestComponentPath = path
		}
	}
	if longestComponentPath != "" {
		return longestComponentPath
	}
	return candidates[0]
}

func isAncestorJSONPath(candidate, descendant string) bool {
	if candidate == "" || descendant == "" || candidate == descendant || len(candidate) >= len(descendant) {
		return false
	}
	if !strings.HasPrefix(descendant, candidate) {
		return false
	}
	next := descendant[len(candidate)]
	return next == '.' || next == '['
}

func (c *resultPathCache) lookupNodePath(node *yaml.Node) (string, bool) {
	if c == nil || node == nil {
		return "", false
	}
	path, found := c.nodePaths[node]
	return path, found
}

func (c *resultPathCache) lookupPositionPathForNode(node *yaml.Node) (string, bool) {
	if node == nil {
		return "", false
	}
	return c.lookupPositionPath(node.Line, node.Column)
}

func (c *resultPathCache) lookupPositionPath(line, column int) (string, bool) {
	if c == nil || line <= 0 || column <= 0 {
		return "", false
	}
	path, found := c.positionPaths[resultPathPosition{line: line, column: column}]
	return path, found
}

func (c *resultPathCache) lookupPrecisePositionPathForNode(node *yaml.Node) (string, bool) {
	if node == nil {
		return "", false
	}
	return c.lookupPrecisePositionPath(node.Line, node.Column)
}

func (c *resultPathCache) lookupPrecisePositionPath(line, column int) (string, bool) {
	if c == nil || line <= 0 || column <= 0 {
		return "", false
	}
	path, found := c.precisePositionMap[resultPathPosition{line: line, column: column}]
	return path, found
}

func (c *resultPathCache) indexNode(node *yaml.Node, path string) {
	if node == nil {
		return
	}

	c.storeNodePath(node, path)

	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			c.indexNode(child, path)
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]
			childPath := vacuumUtils.AppendResultPathSegment(path, keyNode.Value)
			c.storeNodePath(keyNode, path)
			c.storeNodePath(valueNode, path)
			c.storePrecisePositionPath(keyNode, childPath)
			c.storePrecisePositionPath(valueNode, childPath)
			c.indexNode(valueNode, childPath)
		}
	case yaml.SequenceNode:
		for i, child := range node.Content {
			childPath := vacuumUtils.AppendResultPathIndex(path, i)
			c.storeNodePath(child, childPath)
			c.storePrecisePositionPath(child, childPath)
			c.indexNode(child, childPath)
		}
	}
}

func (c *resultPathCache) storeNodePath(node *yaml.Node, path string) {
	if node == nil {
		return
	}
	if _, exists := c.nodePaths[node]; !exists {
		c.nodePaths[node] = path
	}
	if node.Line > 0 && node.Column > 0 {
		position := resultPathPosition{line: node.Line, column: node.Column}
		if _, exists := c.positionPaths[position]; !exists {
			c.positionPaths[position] = path
		}
	}
}

func (c *resultPathCache) storePrecisePositionPath(node *yaml.Node, path string) {
	if c == nil || node == nil || node.Line <= 0 || node.Column <= 0 {
		return
	}
	position := resultPathPosition{line: node.Line, column: node.Column}
	if existing, exists := c.precisePositionMap[position]; !exists || len(path) > len(existing) {
		c.precisePositionMap[position] = path
	}
}

func appendResultPathSegment(basePath, key string) string {
	return vacuumUtils.AppendResultPathSegment(basePath, key)
}

func appendResultPathIndex(basePath string, index int) string {
	return vacuumUtils.AppendResultPathIndex(basePath, index)
}

func isSimpleResultPathKey(key string) bool {
	return vacuumUtils.IsSimpleResultPathKey(key)
}
