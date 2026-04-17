package motor

import (
	"sort"
	"strconv"
	"strings"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
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
	primaryPath := canonicalPath
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
	if primaryPath != "" {
		return primaryPath
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
