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
	rootLocation       string
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
	path := strings.TrimSpace(result.Path)
	return path == "" ||
		path == "unknown" ||
		strings.Contains(path, "*") ||
		(result.PathFromRuleGiven && resultPathHasRecursiveOrFilterSelector(path))
}

func resultPathHasSelectorSyntax(path string) bool {
	path = strings.TrimSpace(path)
	return strings.Contains(path, "*") ||
		resultPathHasRecursiveOrFilterSelector(path)
}

func resultPathHasRecursiveOrFilterSelector(path string) bool {
	path = strings.TrimSpace(path)
	return strings.Contains(path, "..") ||
		strings.Contains(path, "[?")
}

func resultPathIsFallbackSelector(result *model.RuleFunctionResult, path string) bool {
	return result != nil &&
		result.PathFromRuleGiven &&
		resultPathHasSelectorSyntax(path)
}

func resultPathIsFallbackConcretePath(result *model.RuleFunctionResult) bool {
	return result != nil &&
		result.PathFromRuleGiven &&
		result.Path != "" &&
		result.Path != "unknown" &&
		!resultPathHasSelectorSyntax(result.Path)
}

func populateResultOrigins(
	results []model.RuleFunctionResult,
	resolvedRolodex *index.Rolodex,
	cacheRolodex *index.Rolodex,
	specFileName string,
) {
	if resolvedRolodex == nil || len(resolvedRolodex.GetIndexes()) == 0 {
		return
	}
	if cacheRolodex == nil {
		cacheRolodex = resolvedRolodex
	}
	nodeOwnerCache := buildNodeOwnerCache(cacheRolodex)
	rootPath := ""
	if rootIdx := resolvedRolodex.GetRootIndex(); rootIdx != nil {
		rootPath = rootIdx.GetSpecAbsolutePath()
	}
	for i := range results {
		if results[i].Origin != nil || results[i].StartNode == nil {
			continue
		}
		var origin *index.NodeOrigin
		if ownerIdx, ok := nodeOwnerCache[results[i].StartNode]; ok {
			absLoc := ownerIdx.GetSpecAbsolutePath()
			if rootPath != "" && absLoc == rootPath {
				absLoc = specFileName
			}
			origin = &index.NodeOrigin{
				Node:             results[i].StartNode,
				Line:             results[i].StartNode.Line,
				Column:           results[i].StartNode.Column,
				AbsoluteLocation: absLoc,
				Index:            ownerIdx,
			}
		} else {
			origin = resolvedRolodex.FindNodeOrigin(results[i].StartNode)
			if origin != nil && rootPath != "" && origin.AbsoluteLocation == rootPath {
				origin.AbsoluteLocation = specFileName
			}
		}
		if origin != nil {
			results[i].Origin = origin
		}
	}
}

func finalizeResultPaths(
	results []model.RuleFunctionResult,
	aliasRoot *yaml.Node,
	cacheRoot *yaml.Node,
	rootLocation string,
	rolodex *index.Rolodex,
	expandedAliases map[string][]string,
	cacheForMultipleResults bool,
) []model.RuleFunctionResult {
	var cache *resultPathCache
	pathIndexes := make(map[*index.SpecIndex]*vacuumUtils.NodePathIndex)

	// Result path finalization is order-sensitive: expand aliases first, repair
	// selector fallbacks, collapse duplicate alias results, then trim redundant
	// field aliases that only appear after collapse.
	if aliasRoot != nil && needsAliasedResultPathCompletion(results) {
		completeAliasedResultPathsFromGiven(results, aliasRoot, rolodex, expandedAliases)
	}
	completeAliasedResultPathsFromReferences(results, rolodex, pathIndexes)

	needsReconciliation := needsResultPathReconciliation(results)
	needsTerminalKeyUpgrade := needsTerminalKeySelectorPathUpgrade(results)
	needsCollapse := len(results) > 1
	if cacheRoot != nil && (needsReconciliation || needsTerminalKeyUpgrade || (cacheForMultipleResults && needsCollapse)) {
		cache = newResultPathCache(cacheRoot, rootLocation)
		if needsReconciliation {
			for i := range results {
				cache.reconcile(&results[i])
			}
		}
	}
	if needsSelectorTerminalPathUpgrade(results) {
		upgradeSelectorTerminalPaths(results)
	}
	if needsCollapse {
		results = collapseAliasedResults(results, cache)
	}
	upgradeTerminalKeySelectorPaths(results, cache)
	dropRedundantAdditionalPropertiesFieldAliasesFromResults(results, rolodex, pathIndexes)
	return results
}

func newResultPathCache(root *yaml.Node, rootLocation string) *resultPathCache {
	cache := &resultPathCache{
		rootLocation:       rootLocation,
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

	if selectorPath := resultSelectorPath(result); selectorPath != "" {
		if path, found := c.canonicalPathForSelectorResult(result); found {
			result.Path = appendSelectorTerminalSegment(path, selectorPath)
			return
		}
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

func needsSelectorTerminalPathUpgrade(results []model.RuleFunctionResult) bool {
	for i := range results {
		if selectorTerminalPathForResult(&results[i]) != "" {
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
		if c.originMatchesRoot(result.Origin) {
			if path, found := c.lookupPositionPath(result.Origin.Line, result.Origin.Column); found {
				return path, true
			}
			if path, found := c.lookupPositionPath(result.Origin.LineValue, result.Origin.ColumnValue); found {
				return path, true
			}
		}
	}

	if path, found := c.lookupNodePath(result.StartNode); found {
		return path, true
	}
	if result.Origin == nil || c.originMatchesRoot(result.Origin) {
		if path, found := c.lookupPositionPathForNode(result.StartNode); found {
			return path, true
		}
	}
	return "", false
}

func (c *resultPathCache) canonicalPathForSelectorResult(result *model.RuleFunctionResult) (string, bool) {
	if c == nil || result == nil {
		return "", false
	}

	// Selector matches point at the selected node; prefer StartNode before the
	// origin object so a path like "$..[?(@.in == 'header')].name" resolves to
	// the matched scalar rather than the containing parameter object.
	if path, found := c.lookupNodePath(result.StartNode); found {
		return path, true
	}
	if result.Origin != nil {
		if path, found := c.lookupNodePath(result.Origin.ValueNode); found {
			return path, true
		}
		if path, found := c.lookupNodePath(result.Origin.Node); found {
			return path, true
		}
		if c.originMatchesRoot(result.Origin) {
			if path, found := c.lookupPositionPathForNode(result.StartNode); found {
				return path, true
			}
			if path, found := c.lookupPositionPath(result.Origin.LineValue, result.Origin.ColumnValue); found {
				return path, true
			}
			if path, found := c.lookupPositionPath(result.Origin.Line, result.Origin.Column); found {
				return path, true
			}
		}
	}
	return "", false
}

func resultSelectorPath(result *model.RuleFunctionResult) string {
	if result == nil {
		return ""
	}
	if resultPathIsFallbackSelector(result, result.Path) {
		return result.Path
	}
	if !result.PathFromRuleGiven || result.Rule == nil {
		return ""
	}
	for _, givenPath := range resultGivenPaths(result.Rule) {
		if resultPathHasSelectorSyntax(givenPath) {
			return givenPath
		}
	}
	return ""
}

func appendSelectorTerminalSegment(canonicalPath, selectorPath string) string {
	terminal := selectorTerminalSegment(selectorPath)
	if terminal == "" || hasTerminalKeyPathSegment(canonicalPath, terminal) {
		return canonicalPath
	}
	return appendResultPathSegment(canonicalPath, terminal)
}

func selectorTerminalSegment(selectorPath string) string {
	selectorPath = strings.TrimSpace(selectorPath)
	if selectorPath == "" {
		return ""
	}
	if strings.HasSuffix(selectorPath, "]") {
		return bracketSelectorTerminalSegment(selectorPath)
	}
	lastDot := strings.LastIndex(selectorPath, ".")
	if lastDot < 0 || lastDot == len(selectorPath)-1 {
		return ""
	}
	segment := selectorPath[lastDot+1:]
	if !isSimpleResultPathKey(segment) {
		return ""
	}
	return segment
}

func bracketSelectorTerminalSegment(selectorPath string) string {
	lastOpen := strings.LastIndex(selectorPath, "[")
	if lastOpen < 0 || lastOpen >= len(selectorPath)-2 {
		return ""
	}
	segment := selectorPath[lastOpen+1 : len(selectorPath)-1]
	if len(segment) < 2 {
		return ""
	}
	quote := segment[0]
	if (quote != '\'' && quote != '"') || segment[len(segment)-1] != quote {
		return ""
	}
	key := segment[1 : len(segment)-1]
	if !isSimpleResultPathKey(key) {
		return ""
	}
	return key
}

func selectorTerminalPathForResult(result *model.RuleFunctionResult) string {
	if result == nil {
		return ""
	}
	if !resultPathIsFallbackConcretePath(result) {
		return ""
	}
	return selectorTerminalSegment(resultSelectorPath(result))
}

func upgradeSelectorTerminalPaths(results []model.RuleFunctionResult) {
	for i := range results {
		result := &results[i]
		terminal := selectorTerminalPathForResult(result)
		if terminal == "" {
			continue
		}
		result.Path = upgradeSelectorTerminalPath(result.Path, terminal)
		for j := range result.Paths {
			result.Paths[j] = upgradeSelectorTerminalPath(result.Paths[j], terminal)
		}
	}
}

func upgradeSelectorTerminalPath(path, terminal string) string {
	if path == "" || path == "unknown" || resultPathHasSelectorSyntax(path) || hasTerminalKeyPathSegment(path, terminal) {
		return path
	}
	return appendResultPathSegment(path, terminal)
}

func (c *resultPathCache) originMatchesRoot(origin *index.NodeOrigin) bool {
	if c == nil || origin == nil || c.rootLocation == "" {
		return true
	}
	if origin.AbsoluteLocation == "" && origin.AbsoluteLocationValue == "" {
		return true
	}
	if sameResultPathLocation(c.rootLocation, origin.AbsoluteLocation) {
		return true
	}
	return sameResultPathLocation(c.rootLocation, origin.AbsoluteLocationValue)
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

type resultReferenceAliasPathKey struct {
	sourceIndex *index.SpecIndex
	targetIndex *index.SpecIndex
	path        string
}

type resultReferenceAliasIndexPairKey struct {
	sourceIndex *index.SpecIndex
	targetIndex *index.SpecIndex
}

// resultReferenceAliasExpansion splits a resolved target path into the referenced
// root that should be expanded through $ref aliases and the nested suffix to
// append to each alias path.
type resultReferenceAliasExpansion struct {
	rootPath string
	suffix   string
}

type resultReferenceAliasExpansionSet struct {
	expansions []resultReferenceAliasExpansion
	truncated  bool
}

type resultReferenceAliasPathSet struct {
	paths     []string
	truncated bool
}

type resultReferenceAliasBudget struct {
	states           int
	paths            int
	maxStates        int
	maxPaths         int
	aliasesPerResult int
}

const (
	maxResultReferenceAliasStates      = 4096
	maxResultReferenceAliasPaths       = 4096
	maxResultReferenceAliasesPerResult = 256
)

func (b *resultReferenceAliasBudget) consumeState() bool {
	if b == nil {
		return true
	}
	limit := b.maxStates
	if limit <= 0 {
		limit = maxResultReferenceAliasStates
	}
	if b.states >= limit {
		return false
	}
	b.states++
	return true
}

func (b *resultReferenceAliasBudget) consumePath() bool {
	if b == nil {
		return true
	}
	limit := b.maxPaths
	if limit <= 0 {
		limit = maxResultReferenceAliasPaths
	}
	if b.paths >= limit {
		return false
	}
	b.paths++
	return true
}

func (b *resultReferenceAliasBudget) resultAliasLimit() int {
	if b != nil && b.aliasesPerResult > 0 {
		return b.aliasesPerResult
	}
	return maxResultReferenceAliasesPerResult
}

type resultPathNormalizeCache map[string]string

func (c resultPathNormalizeCache) normalize(path string) string {
	if c == nil {
		return normalizeSimpleBracketResultPath(path)
	}
	if normalized, ok := c[path]; ok {
		return normalized
	}
	normalized := normalizeSimpleBracketResultPath(path)
	c[path] = normalized
	return normalized
}

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

func completeAliasedResultPathsFromReferences(
	results []model.RuleFunctionResult,
	rolodex *index.Rolodex,
	pathIndexes map[*index.SpecIndex]*vacuumUtils.NodePathIndex,
) {
	completeAliasedResultPathsFromReferencesWithBudget(
		results, rolodex, pathIndexes, &resultReferenceAliasBudget{},
	)
}

func completeAliasedResultPathsFromReferencesWithBudget(
	results []model.RuleFunctionResult,
	rolodex *index.Rolodex,
	pathIndexes map[*index.SpecIndex]*vacuumUtils.NodePathIndex,
	budget *resultReferenceAliasBudget,
) {
	if len(results) == 0 || rolodex == nil || rolodex.GetRootIndex() == nil || rolodex.GetRootIndex().GetRootNode() == nil {
		return
	}

	if pathIndexes == nil {
		pathIndexes = make(map[*index.SpecIndex]*vacuumUtils.NodePathIndex)
	}
	aliasCache := make(map[resultReferenceAliasKey]resultReferenceAliasPathSet)
	aliasRootCache := make(map[resultReferenceAliasPathKey]string)
	aliasExpansionCache := make(map[resultReferenceAliasPathKey]resultReferenceAliasExpansionSet)
	aliasReferencePathCache := make(map[resultReferenceAliasIndexPairKey][]string)
	normalizeCache := make(resultPathNormalizeCache)
	if budget == nil {
		budget = &resultReferenceAliasBudget{}
	}
	resultAliasLimit := budget.resultAliasLimit()
	type aliasWork struct {
		resultIndex int
		targetIndex *index.SpecIndex
		targetPath  string
	}
	work := make([]aliasWork, 0, len(results))

	for i := range results {
		result := &results[i]
		if !shouldCompleteAliasedResultPathsFromReferences(result) {
			continue
		}

		targetIndex, targetPath := targetPathForResult(result, rolodex, pathIndexes)
		if targetIndex == nil || targetPath == "" {
			continue
		}
		work = append(work, aliasWork{resultIndex: i, targetIndex: targetIndex, targetPath: targetPath})
	}
	sort.SliceStable(work, func(i, j int) bool {
		left, right := work[i], work[j]
		leftLocation, rightLocation := left.targetIndex.GetSpecAbsolutePath(), right.targetIndex.GetSpecAbsolutePath()
		if leftLocation != rightLocation {
			return leftLocation < rightLocation
		}
		if left.targetPath != right.targetPath {
			return left.targetPath < right.targetPath
		}
		leftResult, rightResult := &results[left.resultIndex], &results[right.resultIndex]
		if leftResult.RuleId != rightResult.RuleId {
			return leftResult.RuleId < rightResult.RuleId
		}
		if leftResult.Path != rightResult.Path {
			return leftResult.Path < rightResult.Path
		}
		if leftResult.Message != rightResult.Message {
			return leftResult.Message < rightResult.Message
		}
		leftOrigin, leftLine, leftColumn := stableResultReferenceAliasPosition(leftResult)
		rightOrigin, rightLine, rightColumn := stableResultReferenceAliasPosition(rightResult)
		if leftOrigin != rightOrigin {
			return leftOrigin < rightOrigin
		}
		if leftLine != rightLine {
			return leftLine < rightLine
		}
		if leftColumn != rightColumn {
			return leftColumn < rightColumn
		}
		return left.resultIndex < right.resultIndex
	})

	for _, item := range work {
		result := &results[item.resultIndex]
		targetIndex, targetPath := item.targetIndex, item.targetPath

		expansionSet := referenceAliasExpansionsForTargetPath(
			rolodex.GetRootIndex(),
			targetIndex,
			targetPath,
			pathIndexes,
			aliasRootCache,
			aliasExpansionCache,
			aliasReferencePathCache,
			normalizeCache,
			budget,
		)
		result.PathsTruncated = result.PathsTruncated || expansionSet.truncated

		aliasPaths := make([]string, 0, resultAliasLimit)
		seenAliases := make(map[string]struct{})
		stop := false
		for _, expansion := range expansionSet.expansions {
			cacheKey := resultReferenceAliasKey{index: targetIndex, path: expansion.rootPath}
			rootAliasSet, ok := aliasCache[cacheKey]
			if !ok {
				rootAliasSet = expandResultReferenceAliasPaths(
					rolodex.GetRootIndex(), targetIndex, expansion.rootPath, pathIndexes, budget,
				)
				aliasCache[cacheKey] = rootAliasSet
			}
			result.PathsTruncated = result.PathsTruncated || rootAliasSet.truncated
			for _, rootAliasPath := range rootAliasSet.paths {
				aliasPath := appendResultPathSuffixToAlias(rootAliasPath, expansion.suffix)
				if aliasPath == "" {
					continue
				}
				if _, ok := seenAliases[aliasPath]; ok {
					continue
				}
				if len(aliasPaths) >= resultAliasLimit || !budget.consumePath() {
					result.PathsTruncated = true
					stop = true
					break
				}
				seenAliases[aliasPath] = struct{}{}
				aliasPaths = append(aliasPaths, aliasPath)
			}
			if stop {
				break
			}
		}
		sort.Strings(aliasPaths)
		componentAliasResult := resultUsesResolvedComponentAliases(result)
		if len(aliasPaths) == 0 && !componentAliasResult {
			enforceResultReferenceAliasLimit(result, resultAliasLimit)
			continue
		}

		candidatePaths := make([]string, 0, len(aliasPaths)+1)
		baseTargetCandidate := canonicalizeResultAliasPath(targetPath)
		targetCandidate := baseTargetCandidate
		if componentAliasResult {
			targetCandidate = resolvedComponentTargetPath(result, targetCandidate)
			if fieldSuffix, ok := trimAliasPathPrefix(targetCandidate, baseTargetCandidate); ok && fieldSuffix != "" {
				for i := range aliasPaths {
					aliasPaths[i] = appendResultPathSuffixToAlias(aliasPaths[i], fieldSuffix)
				}
			}
		}
		if strings.HasPrefix(targetCandidate, "$.components.") ||
			resultPathSuffix(result.Path, []string{targetCandidate}) != "" {
			candidatePaths = append(candidatePaths, targetCandidate)
		}
		candidatePaths = append(candidatePaths, aliasPaths...)
		if componentAliasResult {
			paths := buildResolvedComponentResultPaths(targetCandidate, candidatePaths)
			if len(paths) > 0 {
				result.Path = paths[0]
			}
			if len(paths) > 1 {
				result.Paths = paths
			} else {
				result.Paths = nil
			}
			enforceResultReferenceAliasLimit(result, resultAliasLimit)
			continue
		}
		mergeResultPathCandidates(result, applyResultPathSuffix(result, candidatePaths))
		enforceResultReferenceAliasLimit(result, resultAliasLimit)
	}
}

func enforceResultReferenceAliasLimit(result *model.RuleFunctionResult, aliasLimit int) {
	if result == nil || aliasLimit <= 0 || len(result.Paths) <= aliasLimit+1 {
		return
	}
	aliases := make([]string, 0, len(result.Paths))
	seen := map[string]struct{}{result.Path: {}}
	for _, path := range result.Paths {
		if path == "" || path == result.Path {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		aliases = append(aliases, path)
	}
	sort.Strings(aliases)
	truncated := len(aliases) > aliasLimit
	if truncated {
		aliases = aliases[:aliasLimit]
	}
	result.Paths = append([]string{result.Path}, aliases...)
	result.PathsTruncated = result.PathsTruncated || truncated
}

func stableResultReferenceAliasPosition(result *model.RuleFunctionResult) (string, int, int) {
	if result == nil {
		return "", 0, 0
	}
	if result.Origin != nil {
		location := result.Origin.AbsoluteLocation
		if location == "" {
			location = result.Origin.AbsoluteLocationValue
		}
		line, column := result.Origin.Line, result.Origin.Column
		if line <= 0 || column <= 0 {
			line, column = result.Origin.LineValue, result.Origin.ColumnValue
		}
		return location, line, column
	}
	if result.StartNode != nil {
		return "", result.StartNode.Line, result.StartNode.Column
	}
	return "", result.Range.Start.Line, result.Range.Start.Char
}

func buildResolvedComponentResultPaths(canonicalPath string, candidatePaths []string) []string {
	candidatePaths = uniqueSortedResultPaths(append(candidatePaths, canonicalPath))
	if len(candidatePaths) == 0 {
		return nil
	}
	paths := make([]string, 0, len(candidatePaths))
	paths = append(paths, canonicalPath)
	for _, path := range candidatePaths {
		if path != canonicalPath {
			paths = append(paths, path)
		}
	}
	return paths
}

func shouldCompleteAliasedResultPathsFromReferences(result *model.RuleFunctionResult) bool {
	if result == nil || result.Rule == nil {
		return false
	}
	usesRecursiveDescent := ruleUsesRecursiveDescent(result.Rule)
	componentAliasResult := resultUsesResolvedComponentAliases(result)
	if result.StartNode == nil && result.Origin == nil && !resultPathMayReferenceAlias(result.Path) {
		return false
	}
	if !usesRecursiveDescent && !componentAliasResult {
		return false
	}
	if len(result.Paths) <= 1 && !resultPathNeedsReconciliation(result) && !resultPathMayReferenceAlias(result.Path) && !componentAliasResult {
		return false
	}
	if _, ok := aliasedResultKey(result); ok {
		return true
	}
	return resultPathMayReferenceAlias(result.Path) || componentAliasResult
}

func resultUsesResolvedComponentAliases(result *model.RuleFunctionResult) bool {
	return result != nil &&
		result.Rule != nil &&
		!ruleUsesRecursiveDescent(result.Rule) &&
		result.Rule.Resolved &&
		ruleTargetsComponentPaths(result.Rule) &&
		resultPathMayReferenceComponentAlias(result)
}

func resolvedComponentTargetPath(result *model.RuleFunctionResult, targetPath string) string {
	if result == nil || result.Rule == nil || targetPath == "" {
		return targetPath
	}
	for _, field := range resultRuleActionFields(result.Rule) {
		if field == "" || hasTerminalKeyPathSegment(targetPath, field) {
			continue
		}
		if hasTerminalKeyPathSegment(result.Path, field) {
			return appendResultPathSegment(targetPath, field)
		}
		for _, path := range result.Paths {
			if hasTerminalKeyPathSegment(path, field) {
				return appendResultPathSegment(targetPath, field)
			}
		}
	}
	return targetPath
}

func resultRuleActionFields(rule *model.Rule) []string {
	if rule == nil || rule.Then == nil {
		return nil
	}
	var fields []string
	addAction := func(action model.RuleAction) {
		if action.Field != "" {
			fields = append(fields, action.Field)
		}
	}
	switch actions := rule.Then.(type) {
	case model.RuleAction:
		addAction(actions)
	case *model.RuleAction:
		if actions != nil {
			addAction(*actions)
		}
	case []model.RuleAction:
		for _, action := range actions {
			addAction(action)
		}
	case []*model.RuleAction:
		for _, action := range actions {
			if action != nil {
				addAction(*action)
			}
		}
	case map[string]interface{}:
		if field, ok := actions["field"].(string); ok && field != "" {
			fields = append(fields, field)
		}
	case []interface{}:
		for _, rawAction := range actions {
			if action, ok := rawAction.(map[string]interface{}); ok {
				if field, ok := action["field"].(string); ok && field != "" {
					fields = append(fields, field)
				}
			}
		}
	}
	return fields
}

// referenceAliasExpansionsForTargetPath returns the root/suffix pairs needed to
// rebuild aliased result paths for a resolved target path. A rule can report a
// node nested below a referenced schema, for example
// $.components.schemas.Address.properties.street; expanding the whole path would
// miss refs that point at $.components.schemas.Address, so we expand Address and
// append .properties.street to every alias.
func referenceAliasExpansionsForTargetPath(
	sourceIndex *index.SpecIndex,
	targetIndex *index.SpecIndex,
	targetPath string,
	pathIndexes map[*index.SpecIndex]*vacuumUtils.NodePathIndex,
	rootCache map[resultReferenceAliasPathKey]string,
	expansionCache map[resultReferenceAliasPathKey]resultReferenceAliasExpansionSet,
	referencePathCache map[resultReferenceAliasIndexPairKey][]string,
	normalizeCache resultPathNormalizeCache,
	budget *resultReferenceAliasBudget,
) resultReferenceAliasExpansionSet {
	if targetIndex == nil || targetPath == "" {
		return resultReferenceAliasExpansionSet{}
	}

	cacheKey := resultReferenceAliasPathKey{sourceIndex: sourceIndex, targetIndex: targetIndex, path: targetPath}
	if expansionCache != nil {
		if expansionSet, ok := expansionCache[cacheKey]; ok {
			return expansionSet
		}
	}

	targetPathIndex := resultPathIndexForSpec(targetIndex, pathIndexes)
	targetPathSet := equivalentResultReferenceTargetPaths(targetIndex, targetPath, targetPathIndex, budget)
	expansionSet := resultReferenceAliasExpansionSet{
		expansions: make([]resultReferenceAliasExpansion, 0, len(targetPathSet.paths)),
		truncated:  targetPathSet.truncated,
	}
	seen := make(map[string]struct{}, len(targetPathSet.paths))
	for _, candidateTargetPath := range targetPathSet.paths {
		rootPath := referenceAliasExpansionRootForTargetPath(sourceIndex, targetIndex, candidateTargetPath, rootCache, referencePathCache)
		suffix, ok := trimAliasPathPrefixCached(candidateTargetPath, rootPath, normalizeCache)
		if !ok {
			continue
		}
		key := rootPath + "\x00" + suffix
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		expansionSet.expansions = append(expansionSet.expansions, resultReferenceAliasExpansion{rootPath: rootPath, suffix: suffix})
	}

	sort.Slice(expansionSet.expansions, func(i, j int) bool {
		if expansionSet.expansions[i].rootPath == expansionSet.expansions[j].rootPath {
			return expansionSet.expansions[i].suffix < expansionSet.expansions[j].suffix
		}
		return expansionSet.expansions[i].rootPath < expansionSet.expansions[j].rootPath
	})
	if expansionCache != nil {
		expansionCache[cacheKey] = expansionSet
	}
	return expansionSet
}

// referenceAliasExpansionRootForTargetPath finds the deepest $ref target path
// that contains targetPath. That root is the stable cache key for alias
// expansion, while the remainder is kept as a suffix so nested results under
// the referenced schema still map back to every place the schema was used.
func referenceAliasExpansionRootForTargetPath(
	sourceIndex *index.SpecIndex,
	targetIndex *index.SpecIndex,
	targetPath string,
	cache map[resultReferenceAliasPathKey]string,
	referencePathCache map[resultReferenceAliasIndexPairKey][]string,
) string {
	if targetIndex == nil || targetPath == "" {
		return targetPath
	}
	targetPath = canonicalizeResultAliasPath(targetPath)
	cacheKey := resultReferenceAliasPathKey{sourceIndex: sourceIndex, targetIndex: targetIndex, path: targetPath}
	if cache != nil {
		if rootPath, ok := cache[cacheKey]; ok {
			return rootPath
		}
	}

	rootPath := targetPath
	rootPathLen := 0
	for _, refPath := range resultReferenceAliasRootCandidatePaths(sourceIndex, targetIndex, referencePathCache) {
		if refPath == "" || len(refPath) <= rootPathLen {
			continue
		}
		if resultPathHasPrefix(targetPath, refPath) {
			rootPath = refPath
			rootPathLen = len(refPath)
		}
	}

	if cache != nil {
		cache[cacheKey] = rootPath
	}
	return rootPath
}

// resultReferenceAliasRootCandidatePaths collects canonical $ref target paths
// for a source/target index pair once per reconciliation pass. These are the
// only paths that can act as expansion roots, and caching them avoids walking
// every sequenced reference for each candidate result path.
func resultReferenceAliasRootCandidatePaths(
	sourceIndex *index.SpecIndex,
	targetIndex *index.SpecIndex,
	cache map[resultReferenceAliasIndexPairKey][]string,
) []string {
	if targetIndex == nil {
		return nil
	}
	cacheKey := resultReferenceAliasIndexPairKey{sourceIndex: sourceIndex, targetIndex: targetIndex}
	if cache != nil {
		if paths, ok := cache[cacheKey]; ok {
			return paths
		}
	}

	targetReferences := targetIndex.GetAllSequencedReferences()
	estimatedCapacity := len(targetReferences)
	var sourceReferences []*index.Reference
	if sourceIndex != nil && sourceIndex != targetIndex {
		sourceReferences = sourceIndex.GetAllSequencedReferences()
		estimatedCapacity += len(sourceReferences)
	}
	paths := make([]string, 0, estimatedCapacity)
	collect := func(refs []*index.Reference) {
		for _, ref := range refs {
			if ref == nil || ref.Path == "" || !referenceTargetsIndex(ref, targetIndex) {
				continue
			}
			refPath := canonicalizeResultAliasPath(ref.Path)
			if refPath != "" {
				paths = append(paths, refPath)
			}
		}
	}
	collect(targetReferences)
	collect(sourceReferences)

	if cache != nil {
		cache[cacheKey] = paths
	}
	return paths
}

func ruleTargetsComponentPaths(rule *model.Rule) bool {
	for _, givenPath := range resultGivenPaths(rule) {
		givenPath = strings.TrimSpace(givenPath)
		if givenPath == "$.components" || strings.HasPrefix(givenPath, "$.components.") {
			return true
		}
		normalizedPath := normalizeSimpleBracketResultPath(givenPath)
		if normalizedPath == "$.components" || strings.HasPrefix(normalizedPath, "$.components.") {
			return true
		}
	}
	return false
}

func resultPathMayReferenceAlias(path string) bool {
	return strings.Contains(path, ".allOf[") ||
		strings.Contains(path, ".anyOf[") ||
		strings.Contains(path, ".oneOf[")
}

func resultPathMayReferenceComponentAlias(result *model.RuleFunctionResult) bool {
	if result == nil {
		return false
	}
	if resultPathIsComponent(result.Path) {
		return true
	}
	for _, path := range result.Paths {
		if resultPathIsComponent(path) {
			return true
		}
	}
	return false
}

func resultPathIsComponent(path string) bool {
	if path == "" {
		return false
	}
	return strings.HasPrefix(canonicalizeResultAliasPath(path), "$.components.")
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
		if targetPath := targetPathFromReferenceAliasResultPath(result.Path); targetPath != "" {
			return rolodex.GetRootIndex(), targetPath
		}
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
	if targetPath := targetPathFromReferenceAliasResultPath(result.Path); targetPath != "" {
		return rolodex.GetRootIndex(), targetPath
	}
	return nil, ""
}

func targetPathFromReferenceAliasResultPath(path string) string {
	for _, marker := range []string{".allOf[", ".anyOf[", ".oneOf["} {
		idx := strings.Index(path, marker)
		if idx > 0 {
			return normalizeSimpleBracketResultPath(path[:idx])
		}
	}
	return ""
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
	budget *resultReferenceAliasBudget,
) resultReferenceAliasPathSet {
	if sourceIndex == nil || targetIndex == nil || targetPath == "" {
		return resultReferenceAliasPathSet{}
	}

	sourcePathIndex := resultPathIndexForSpec(sourceIndex, pathIndexes)
	targetPathIndex := resultPathIndexForSpec(targetIndex, pathIndexes)
	if sourcePathIndex == nil || targetPathIndex == nil {
		return resultReferenceAliasPathSet{}
	}

	targetPathSet := equivalentResultReferenceTargetPaths(targetIndex, targetPath, targetPathIndex, budget)
	pathSet := resultReferenceAliasPathSet{truncated: targetPathSet.truncated}
	seenPaths := make(map[string]struct{})
	sourceReferences := sourceIndex.GetAllSequencedReferences()
	targetReferences := targetIndex.GetAllSequencedReferences()
	normalizeCache := make(resultPathNormalizeCache, len(targetPathSet.paths)+len(sourceReferences)+len(targetReferences))
	sourceEdges := sortedResultReferenceAliasEdges(sourceReferences, targetIndex, sourcePathIndex)
	targetEdges := sortedResultReferenceAliasEdges(targetReferences, targetIndex, targetPathIndex)
	addPath := func(path string, consumeBudget bool) bool {
		path = canonicalizeResultAliasPath(path)
		if path == "" {
			return true
		}
		if _, ok := seenPaths[path]; ok {
			return true
		}
		if consumeBudget && !budget.consumePath() {
			pathSet.truncated = true
			return false
		}
		seenPaths[path] = struct{}{}
		pathSet.paths = append(pathSet.paths, path)
		return true
	}
	for _, candidateTargetPath := range targetPathSet.paths {
		if strings.HasPrefix(candidateTargetPath, "$.components.") {
			if !addPath(candidateTargetPath, true) {
				break
			}
		}
		for _, edge := range sourceEdges {
			expanded := expandResultReferenceAliasPath(
				edge.targetPath,
				edge.sourcePath,
				candidateTargetPath,
				targetEdges,
				normalizeCache,
				budget,
			)
			pathSet.truncated = pathSet.truncated || expanded.truncated
			for _, path := range expanded.paths {
				if !addPath(path, false) {
					break
				}
			}
			if expanded.truncated {
				break
			}
		}
		if pathSet.truncated {
			break
		}
	}
	sort.Strings(pathSet.paths)
	return pathSet
}

type resultReferenceAliasEdge struct {
	sourcePath string
	targetPath string
}

type resultReferenceAliasVisit struct {
	target string
	parent *resultReferenceAliasVisit
}

func (v *resultReferenceAliasVisit) contains(target string) bool {
	for current := v; current != nil; current = current.parent {
		if current.target == target {
			return true
		}
	}
	return false
}

func sortedResultReferenceAliasEdges(
	references []*index.Reference,
	targetIndex *index.SpecIndex,
	pathIndex *vacuumUtils.NodePathIndex,
) []resultReferenceAliasEdge {
	edges := make([]resultReferenceAliasEdge, 0, len(references))
	for _, ref := range references {
		if ref == nil || ref.Node == nil || ref.Path == "" || !referenceTargetsIndex(ref, targetIndex) {
			continue
		}
		sourcePath, ok := pathIndex.Lookup(ref.Node)
		if !ok || sourcePath == "" {
			continue
		}
		edges = append(edges, resultReferenceAliasEdge{
			sourcePath: canonicalizeResultAliasPath(sourcePath),
			targetPath: canonicalizeResultAliasPath(ref.Path),
		})
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].sourcePath == edges[j].sourcePath {
			return edges[i].targetPath < edges[j].targetPath
		}
		return edges[i].sourcePath < edges[j].sourcePath
	})
	unique := edges[:0]
	for _, edge := range edges {
		if len(unique) > 0 &&
			unique[len(unique)-1].sourcePath == edge.sourcePath &&
			unique[len(unique)-1].targetPath == edge.targetPath {
			continue
		}
		unique = append(unique, edge)
	}
	return unique
}

func equivalentResultReferenceTargetPaths(
	targetIndex *index.SpecIndex,
	targetPath string,
	targetPathIndex *vacuumUtils.NodePathIndex,
	budget *resultReferenceAliasBudget,
) resultReferenceAliasPathSet {
	if targetIndex == nil || targetPath == "" || targetPathIndex == nil {
		return resultReferenceAliasPathSet{}
	}

	pathSet := resultReferenceAliasPathSet{}
	seenPaths := make(map[string]struct{})
	type queuedPath struct {
		path    string
		visited *resultReferenceAliasVisit
	}

	var queue []queuedPath
	add := func(path string, visited *resultReferenceAliasVisit, consumeBudget bool) bool {
		if path == "" {
			return true
		}
		path = canonicalizeResultAliasPath(path)
		if _, ok := seenPaths[path]; ok {
			return true
		}
		if consumeBudget && (!budget.consumeState() || !budget.consumePath()) {
			pathSet.truncated = true
			return false
		}
		seenPaths[path] = struct{}{}
		pathSet.paths = append(pathSet.paths, path)
		queue = append(queue, queuedPath{path: path, visited: visited})
		return true
	}

	targetPath = canonicalizeResultAliasPath(targetPath)
	add(targetPath, &resultReferenceAliasVisit{target: targetPath}, false)
	targetReferences := targetIndex.GetAllSequencedReferences()
	normalizeCache := make(resultPathNormalizeCache, len(targetReferences)+1)
	edges := sortedResultReferenceAliasEdges(targetReferences, targetIndex, targetPathIndex)
	for head := 0; head < len(queue) && !pathSet.truncated; head++ {
		current := queue[head]

		for _, edge := range edges {
			suffix, ok := trimAliasPathPrefixCached(current.path, edge.sourcePath, normalizeCache)
			if !ok {
				continue
			}
			semanticTarget := edge.targetPath
			if current.visited.contains(semanticTarget) {
				continue
			}
			visited := &resultReferenceAliasVisit{target: semanticTarget, parent: current.visited}
			if !add(edge.targetPath+suffix, visited, true) {
				break
			}
		}
	}

	sort.Strings(pathSet.paths)
	return pathSet
}

func expandResultReferenceAliasPath(
	currentTargetPath string,
	currentSourcePath string,
	targetPath string,
	targetEdges []resultReferenceAliasEdge,
	normalizeCache resultPathNormalizeCache,
	budget *resultReferenceAliasBudget,
) resultReferenceAliasPathSet {
	if currentTargetPath == "" || currentSourcePath == "" || targetPath == "" {
		return resultReferenceAliasPathSet{}
	}

	type queuedAlias struct {
		targetPath string
		sourcePath string
		visited    *resultReferenceAliasVisit
	}
	currentTargetPath = canonicalizeResultAliasPath(currentTargetPath)
	targetPath = canonicalizeResultAliasPath(targetPath)
	visited := &resultReferenceAliasVisit{target: currentTargetPath}
	if resultPathHasPrefix(canonicalizeResultAliasPath(currentSourcePath), targetPath) {
		visited = &resultReferenceAliasVisit{target: targetPath, parent: visited}
	}
	queue := []queuedAlias{{
		targetPath: currentTargetPath,
		sourcePath: canonicalizeResultAliasPath(currentSourcePath),
		visited:    visited,
	}}
	pathSet := resultReferenceAliasPathSet{}
	seenPaths := make(map[string]struct{})
	addPath := func(path string) bool {
		path = canonicalizeResultAliasPath(path)
		if path == "" {
			return true
		}
		if _, ok := seenPaths[path]; ok {
			return true
		}
		if !budget.consumePath() {
			pathSet.truncated = true
			return false
		}
		seenPaths[path] = struct{}{}
		pathSet.paths = append(pathSet.paths, path)
		return true
	}

	for head := 0; head < len(queue) && !pathSet.truncated; head++ {
		current := queue[head]
		if !budget.consumeState() {
			pathSet.truncated = true
			break
		}
		if suffix, ok := trimAliasPathPrefixCached(targetPath, current.targetPath, normalizeCache); ok {
			if !addPath(current.sourcePath + suffix) {
				break
			}
		}
		for _, edge := range targetEdges {
			nestedSuffix, ok := trimAliasPathPrefixCached(edge.sourcePath, current.targetPath, normalizeCache)
			if !ok {
				continue
			}
			if current.visited.contains(edge.targetPath) {
				continue
			}
			visited := &resultReferenceAliasVisit{target: edge.targetPath, parent: current.visited}
			queue = append(queue, queuedAlias{
				targetPath: edge.targetPath,
				sourcePath: canonicalizeResultAliasPath(current.sourcePath + nestedSuffix),
				visited:    visited,
			})
		}
	}

	sort.Strings(pathSet.paths)
	return pathSet
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

func appendResultPathSuffixToAliases(paths []string, suffix string) []string {
	if len(paths) == 0 || suffix == "" {
		return paths
	}
	suffixed := make([]string, 0, len(paths))
	for _, path := range paths {
		suffixed = append(suffixed, appendResultPathSuffixToAlias(path, suffix))
	}
	return suffixed
}

func appendResultPathSuffixToAlias(path, suffix string) string {
	if path == "" || suffix == "" {
		return path
	}
	return canonicalizeResultAliasPath(path + suffix)
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

func trimAliasPathPrefixCached(path, prefix string, normalizeCache resultPathNormalizeCache) (string, bool) {
	if resultPathHasPrefix(path, prefix) {
		return strings.TrimPrefix(path, prefix), true
	}

	normalizedPath := normalizeCache.normalize(path)
	normalizedPrefix := normalizeCache.normalize(prefix)
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
	for _, marker := range []string{".components.schemas.", ".properties.", ".patternProperties."} {
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
	candidatePaths = uniqueSortedResultPaths(candidatePaths)
	if len(candidatePaths) == 0 {
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
	if resultPathCandidatesAreAuthoritative(result, candidatePaths) {
		mergedResult.Path = ""
		mergedResult.Paths = append(mergedResult.Paths, candidatePaths...)
	} else {
		mergedResult.Paths = make([]string, 0, len(result.Paths)+len(candidatePaths))
		mergedResult.Paths = append(mergedResult.Paths, result.Paths...)
		mergedResult.Paths = append(mergedResult.Paths, candidatePaths...)
	}
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

func resultPathCandidatesAreAuthoritative(result *model.RuleFunctionResult, candidatePaths []string) bool {
	if result == nil || len(candidatePaths) <= 1 {
		return false
	}
	if strings.HasPrefix(result.Path, "$.components.") {
		return false
	}
	return resultPathCandidatesOverlap(result, candidatePaths)
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
	if cache != nil && !resultUsesResolvedComponentAliases(primary) {
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
	primary.PathsTruncated = primary.PathsTruncated || duplicate.PathsTruncated
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

func dropRedundantAdditionalPropertiesFieldAliasesFromResults(
	results []model.RuleFunctionResult,
	rolodex *index.Rolodex,
	pathIndexes map[*index.SpecIndex]*vacuumUtils.NodePathIndex,
) {
	if len(results) == 0 || rolodex == nil || rolodex.GetRootIndex() == nil {
		return
	}
	if !resultsContainDirectAdditionalPropertiesFieldAlias(results) {
		return
	}

	// A schema referenced both directly and through additionalProperties can
	// produce parent-field and additionalProperties-field paths for the same
	// violation. Keep the direct field path unless additionalProperties is the
	// actual reference source.
	refSourcePaths := resultReferenceSourcePaths(rolodex.GetRootIndex(), pathIndexes)
	if len(refSourcePaths) == 0 {
		return
	}

	for i := range results {
		if len(results[i].Paths) <= 1 {
			continue
		}
		filteredPaths := dropRedundantAdditionalPropertiesFieldAliases(results[i].Paths, refSourcePaths)
		if len(filteredPaths) == len(results[i].Paths) {
			continue
		}
		results[i].Paths = filteredPaths
		if !resultPathListContains(filteredPaths, results[i].Path) && len(filteredPaths) > 0 {
			results[i].Path = filteredPaths[0]
		}
		if len(results[i].Paths) <= 1 {
			results[i].Paths = nil
		}
	}
}

func resultsContainDirectAdditionalPropertiesFieldAlias(results []model.RuleFunctionResult) bool {
	for i := range results {
		if len(results[i].Paths) <= 1 {
			continue
		}
		pathSet := make(map[string]struct{}, len(results[i].Paths))
		for _, path := range results[i].Paths {
			pathSet[path] = struct{}{}
		}
		for _, path := range results[i].Paths {
			parentFieldPath, _, ok := directAdditionalPropertiesParentFieldPath(path)
			if !ok {
				continue
			}
			if _, found := pathSet[parentFieldPath]; found {
				return true
			}
		}
	}
	return false
}

func resultReferenceSourcePaths(sourceIndex *index.SpecIndex, pathIndexes map[*index.SpecIndex]*vacuumUtils.NodePathIndex) map[string]struct{} {
	if sourceIndex == nil || sourceIndex.GetRootNode() == nil {
		return nil
	}
	if pathIndexes == nil {
		pathIndexes = make(map[*index.SpecIndex]*vacuumUtils.NodePathIndex)
	}

	pathIndex := resultPathIndexForSpec(sourceIndex, pathIndexes)
	if pathIndex == nil {
		return nil
	}

	refSourcePaths := make(map[string]struct{})
	for _, ref := range sourceIndex.GetAllSequencedReferences() {
		if ref == nil || ref.Node == nil {
			continue
		}
		sourcePath, ok := pathIndex.Lookup(ref.Node)
		if !ok || sourcePath == "" {
			continue
		}
		refSourcePaths[canonicalizeResultAliasPath(sourcePath)] = struct{}{}
	}
	return refSourcePaths
}

func dropRedundantAdditionalPropertiesFieldAliases(paths []string, refSourcePaths map[string]struct{}) []string {
	if len(paths) <= 1 {
		return paths
	}

	pathSet := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		pathSet[path] = struct{}{}
	}

	filtered := make([]string, 0, len(paths))
	for _, path := range paths {
		if parentFieldPath, additionalPropertiesPath, ok := directAdditionalPropertiesParentFieldPath(path); ok {
			if _, found := pathSet[parentFieldPath]; found {
				if _, refFound := refSourcePaths[additionalPropertiesPath]; !refFound {
					continue
				}
			}
		}
		filtered = append(filtered, path)
	}
	return filtered
}

func directAdditionalPropertiesParentFieldPath(path string) (string, string, bool) {
	const marker = ".additionalProperties."
	idx := strings.LastIndex(path, marker)
	if idx < 0 {
		return "", "", false
	}
	suffix := path[idx+len(marker):]
	if suffix == "" || strings.ContainsAny(suffix, ".[") {
		return "", "", false
	}
	return path[:idx] + "." + suffix, path[:idx] + ".additionalProperties", true
}

func resultPathListContains(paths []string, path string) bool {
	for _, candidate := range paths {
		if candidate == path {
			return true
		}
	}
	return false
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
