package motor

import (
	"strconv"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"go.yaml.in/yaml/v4"
)

type resultPathPosition struct {
	line   int
	column int
}

type resultPathCache struct {
	nodePaths     map[*yaml.Node]string
	positionPaths map[resultPathPosition]string
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
		nodePaths:     make(map[*yaml.Node]string),
		positionPaths: make(map[resultPathPosition]string),
	}
	cache.indexNode(root, "$")
	return cache
}

func (c *resultPathCache) reconcile(result *model.RuleFunctionResult) {
	if c == nil || result == nil || !resultPathNeedsReconciliation(result) {
		return
	}

	if result.Origin != nil {
		if path, found := c.lookupNodePath(result.Origin.Node); found {
			result.Path = path
			return
		}
		if path, found := c.lookupNodePath(result.Origin.ValueNode); found {
			result.Path = path
			return
		}
		if path, found := c.lookupPositionPath(result.Origin.Line, result.Origin.Column); found {
			result.Path = path
			return
		}
		if path, found := c.lookupPositionPath(result.Origin.LineValue, result.Origin.ColumnValue); found {
			result.Path = path
			return
		}
	}

	if path, found := c.lookupNodePath(result.StartNode); found {
		result.Path = path
		return
	}
	if path, found := c.lookupPositionPathForNode(result.StartNode); found {
		result.Path = path
	}
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
			c.storeNodePath(keyNode, path)
			c.storeNodePath(valueNode, path)
			c.indexNode(valueNode, appendResultPathSegment(path, keyNode.Value))
		}
	case yaml.SequenceNode:
		for i, child := range node.Content {
			childPath := appendResultPathIndex(path, i)
			c.storeNodePath(child, childPath)
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

func appendResultPathSegment(basePath, key string) string {
	if isSimpleResultPathKey(key) {
		return basePath + "." + key
	}
	return basePath + "['" + key + "']"
}

func appendResultPathIndex(basePath string, index int) string {
	return basePath + "[" + strconv.Itoa(index) + "]"
}

func isSimpleResultPathKey(key string) bool {
	if key == "" {
		return false
	}

	first := key[0]
	if !((first >= 'A' && first <= 'Z') || (first >= 'a' && first <= 'z') || first == '_') {
		return false
	}

	for i := 1; i < len(key); i++ {
		ch := key[i]
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '_' {
			continue
		}
		return false
	}
	return true
}
