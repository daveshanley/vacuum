package transform

import (
	"fmt"
	"strings"

	"go.yaml.in/yaml/v4"
)

var standardOperationKeys = map[string]bool{
	"get": true, "put": true, "post": true, "delete": true,
	"options": true, "head": true, "patch": true, "trace": true,
}

// FilterOperationsByTags keeps only operations matching the inclusive tag
// allow-list. An empty allow-list is a no-op.
func FilterOperationsByTags(root *yaml.Node, specVersion string, options TagFilterOptions) (FilterStats, error) {
	var stats FilterStats
	root, err := requireMappingRoot(root)
	if err != nil {
		return stats, err
	}
	if len(options.IncludeTags) == 0 {
		return stats, nil
	}
	strategy := options.MatchStrategy
	if strategy == "" {
		strategy = MatchAny
	}
	if strategy != MatchAny && strategy != MatchAll {
		return stats, fmt.Errorf("invalid tag match strategy %q: expected any or all", strategy)
	}
	included := make(map[string]struct{}, len(options.IncludeTags))
	for _, tag := range options.IncludeTags {
		if strings.TrimSpace(tag) == "" {
			return stats, fmt.Errorf("included tags must not be empty or whitespace")
		}
		included[tag] = struct{}{}
	}
	beforeIDs, beforeRefs := snapshotOperationTargets(root, specVersion)
	ctx := filterContext{
		root:        root,
		version:     specVersion,
		included:    included,
		strategy:    strategy,
		processed:   make(map[*yaml.Node]bool),
		processing:  make(map[*yaml.Node]bool),
		callbacks:   make(map[*yaml.Node]bool),
		callbacking: make(map[*yaml.Node]bool),
		deferred:    make(map[*yaml.Node]struct{}),
		stats:       &stats,
	}
	ctx.filterRootMap("paths", false)
	if !strings.HasPrefix(specVersion, "2.") {
		if strings.HasPrefix(specVersion, "3.1") || strings.HasPrefix(specVersion, "3.2") {
			ctx.filterRootMap("webhooks", true)
		}
		ctx.filterReusable()
	}
	ctx.finalizeDeferredCallbacks()
	afterIDs, afterRefs := snapshotOperationTargets(root, specVersion)
	stats.Warnings = danglingLinkWarnings(root, beforeIDs, afterIDs, beforeRefs, afterRefs)
	return stats, nil
}

type filterContext struct {
	root        *yaml.Node
	version     string
	included    map[string]struct{}
	strategy    MatchStrategy
	processed   map[*yaml.Node]bool
	processing  map[*yaml.Node]bool
	callbacks   map[*yaml.Node]bool
	callbacking map[*yaml.Node]bool
	deferred    map[*yaml.Node]struct{}
	stats       *FilterStats
}

func (c *filterContext) filterRootMap(key string, webhook bool) {
	container := mapValue(c.root, key)
	if container == nil || container.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(container.Content); {
		if strings.HasPrefix(container.Content[i].Value, "x-") {
			i += 2
			continue
		}
		value := container.Content[i+1]
		if c.filterPathItem(value) {
			i += 2
			continue
		}
		removeMapPair(container, i)
		if webhook {
			c.stats.WebhooksRemoved++
		} else {
			c.stats.PathItemsRemoved++
		}
	}
}

// HasReachableOperations reports whether at least one operation is reachable
// from the document's Paths or Webhooks Objects. Unreferenced reusable Path
// Items and Callbacks are not publication roots.
func HasReachableOperations(root *yaml.Node, specVersion string) bool {
	root = documentRoot(root)
	if root == nil || root.Kind != yaml.MappingNode {
		return false
	}
	seen := make(map[*yaml.Node]struct{})
	var hasOperation func(*yaml.Node) bool
	hasOperation = func(node *yaml.Node) bool {
		if node == nil || node.Kind != yaml.MappingNode {
			return false
		}
		if _, ok := seen[node]; ok {
			return false
		}
		seen[node] = struct{}{}
		if ref := mapValue(node, "$ref"); ref != nil && ref.Kind == yaml.ScalarNode {
			if hasOperation(resolveLocalNode(root, ref.Value)) {
				return true
			}
		}
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i].Value
			if standardOperationKeys[key] || (key == "query" && strings.HasPrefix(specVersion, "3.2")) {
				return true
			}
			if key == "additionalOperations" && strings.HasPrefix(specVersion, "3.2") {
				operations := node.Content[i+1]
				if operations.Kind == yaml.MappingNode && len(operations.Content) > 0 {
					return true
				}
			}
		}
		return false
	}
	for _, key := range []string{"paths", "webhooks"} {
		if key == "webhooks" && !strings.HasPrefix(specVersion, "3.1") && !strings.HasPrefix(specVersion, "3.2") {
			continue
		}
		container := mapValue(root, key)
		if container == nil || container.Kind != yaml.MappingNode {
			continue
		}
		for i := 0; i+1 < len(container.Content); i += 2 {
			if strings.HasPrefix(container.Content[i].Value, "x-") {
				continue
			}
			if hasOperation(container.Content[i+1]) {
				return true
			}
		}
	}
	return false
}

func (c *filterContext) filterReusable() {
	components := mapValue(c.root, "components")
	if components == nil || components.Kind != yaml.MappingNode {
		return
	}
	if strings.HasPrefix(c.version, "3.1") || strings.HasPrefix(c.version, "3.2") {
		if pathItems := mapValue(components, "pathItems"); pathItems != nil && pathItems.Kind == yaml.MappingNode {
			for i := 1; i < len(pathItems.Content); i += 2 {
				c.filterPathItem(pathItems.Content[i])
			}
		}
	}
	if callbacks := mapValue(components, "callbacks"); callbacks != nil && callbacks.Kind == yaml.MappingNode {
		for i := 1; i < len(callbacks.Content); i += 2 {
			c.filterCallback(callbacks.Content[i])
		}
	}
}

func (c *filterContext) filterPathItem(node *yaml.Node) bool {
	hasOperation, _ := c.filterPathItemState(node)
	return hasOperation
}

// filterPathItemState reports whether a retained operation is reachable and
// whether that negative result is definitive rather than caused by a back-edge.
func (c *filterContext) filterPathItemState(node *yaml.Node) (bool, bool) {
	if node == nil || node.Kind != yaml.MappingNode {
		return false, true
	}
	if result, ok := c.processed[node]; ok {
		return result, true
	}
	if c.processing[node] {
		return false, false
	}
	c.processing[node] = true
	hasOperation := false
	complete := true
	for i := 0; i+1 < len(node.Content); {
		key, value := node.Content[i].Value, node.Content[i+1]
		if c.isOperationKey(key) {
			c.stats.OperationsSeen++
			if c.matches(value) {
				c.stats.OperationsKept++
				c.filterOperationCallbacks(value)
				hasOperation = true
				i += 2
			} else {
				c.stats.OperationsRemoved++
				removeMapPair(node, i)
			}
			continue
		}
		if key == "additionalOperations" && strings.HasPrefix(c.version, "3.2") {
			if c.filterAdditionalOperations(value) {
				hasOperation = true
				i += 2
			} else {
				removeMapPair(node, i)
			}
			continue
		}
		if key == "$ref" && value.Kind == yaml.ScalarNode && strings.HasPrefix(value.Value, "#") {
			if target := resolveLocalNode(c.root, value.Value); target != nil {
				hasReferencedOperation, referenceComplete := c.filterPathItemState(target)
				hasOperation = hasOperation || hasReferencedOperation
				complete = complete && referenceComplete
			}
		}
		i += 2
	}
	delete(c.processing, node)
	if hasOperation || complete {
		c.processed[node] = hasOperation
		return hasOperation, true
	}
	return false, false
}

func (c *filterContext) isOperationKey(key string) bool {
	return standardOperationKeys[key] || (key == "query" && strings.HasPrefix(c.version, "3.2"))
}

func (c *filterContext) matches(operation *yaml.Node) bool {
	tags := mapValue(operation, "tags")
	if tags == nil || tags.Kind != yaml.SequenceNode {
		return false
	}
	operationTags := make(map[string]struct{}, len(tags.Content))
	for _, tag := range tags.Content {
		if tag.Kind != yaml.ScalarNode || tag.Tag != "!!str" {
			return false
		}
		operationTags[tag.Value] = struct{}{}
	}
	if c.strategy == MatchAll {
		for tag := range c.included {
			if _, ok := operationTags[tag]; !ok {
				return false
			}
		}
		return true
	}
	for tag := range c.included {
		if _, ok := operationTags[tag]; ok {
			return true
		}
	}
	return false
}

func (c *filterContext) filterAdditionalOperations(node *yaml.Node) bool {
	if node == nil || node.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i+1 < len(node.Content); {
		c.stats.OperationsSeen++
		if c.matches(node.Content[i+1]) {
			c.stats.OperationsKept++
			c.filterOperationCallbacks(node.Content[i+1])
			i += 2
		} else {
			c.stats.OperationsRemoved++
			removeMapPair(node, i)
		}
	}
	return len(node.Content) > 0
}

func (c *filterContext) filterOperationCallbacks(operation *yaml.Node) {
	c.filterOperationCallbacksNow(operation, true)
}

func (c *filterContext) filterOperationCallbacksNow(operation *yaml.Node, deferUnresolved bool) {
	callbacks := mapValue(operation, "callbacks")
	if callbacks == nil || callbacks.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(callbacks.Content); {
		hasItem, complete := c.filterCallbackState(callbacks.Content[i+1])
		if hasItem {
			i += 2
		} else if complete || !deferUnresolved {
			removeMapPair(callbacks, i)
		} else {
			c.deferred[operation] = struct{}{}
			i += 2
		}
	}
	if len(callbacks.Content) == 0 {
		for i := 0; i+1 < len(operation.Content); i += 2 {
			if operation.Content[i].Value == "callbacks" {
				removeMapPair(operation, i)
				break
			}
		}
	}
}

func (c *filterContext) finalizeDeferredCallbacks() {
	for operation := range c.deferred {
		c.filterOperationCallbacksNow(operation, false)
	}
}

func (c *filterContext) filterCallback(node *yaml.Node) bool {
	hasItem, _ := c.filterCallbackState(node)
	return hasItem
}

func (c *filterContext) filterCallbackState(node *yaml.Node) (bool, bool) {
	if node == nil || node.Kind != yaml.MappingNode {
		return false, true
	}
	if result, ok := c.callbacks[node]; ok {
		return result, true
	}
	if c.callbacking[node] {
		return false, false
	}
	c.callbacking[node] = true
	hasItem := false
	complete := true
	if ref := mapValue(node, "$ref"); ref != nil && ref.Kind == yaml.ScalarNode {
		if target := resolveLocalNode(c.root, ref.Value); target != nil {
			hasReferencedItem, referenceComplete := c.filterCallbackState(target)
			hasItem = hasItem || hasReferencedItem
			complete = complete && referenceComplete
		}
	}
	for i := 0; i+1 < len(node.Content); {
		key := node.Content[i].Value
		if key == "$ref" || strings.HasPrefix(key, "x-") {
			i += 2
			continue
		}
		itemHasOperation, itemComplete := c.filterPathItemState(node.Content[i+1])
		if itemHasOperation {
			hasItem = true
			i += 2
		} else if itemComplete {
			removeMapPair(node, i)
			c.stats.CallbackItemsRemoved++
		} else {
			complete = false
			i += 2
		}
	}
	delete(c.callbacking, node)
	if hasItem || complete {
		c.callbacks[node] = hasItem
		return hasItem, true
	}
	return false, false
}

func resolveLocalNode(root *yaml.Node, ref string) *yaml.Node {
	tokens, err := localPointerTokens(ref)
	if err != nil || len(tokens) == 0 {
		return nil
	}
	node := root
	for _, token := range tokens {
		node = mapValue(node, token)
		if node == nil {
			return nil
		}
	}
	return node
}

func snapshotOperationTargets(root *yaml.Node, version string) (map[string]struct{}, map[string]struct{}) {
	ids := make(map[string]struct{})
	refs := make(map[string]struct{})
	seen := make(map[*yaml.Node]struct{})
	seenCallbacks := make(map[*yaml.Node]struct{})
	var pathItem func(*yaml.Node, []string)
	var callback func(*yaml.Node, []string)
	var operation func(*yaml.Node, []string)
	operation = func(node *yaml.Node, path []string) {
		if id := mapValue(node, "operationId"); id != nil && id.Kind == yaml.ScalarNode {
			ids[id.Value] = struct{}{}
		}
		callbacks := mapValue(node, "callbacks")
		if callbacks != nil && callbacks.Kind == yaml.MappingNode {
			for i := 0; i+1 < len(callbacks.Content); i += 2 {
				callback(callbacks.Content[i+1], appendPath(appendPath(path, "callbacks"), callbacks.Content[i].Value))
			}
		}
	}
	callback = func(node *yaml.Node, path []string) {
		if node == nil || node.Kind != yaml.MappingNode {
			return
		}
		if _, ok := seenCallbacks[node]; ok {
			return
		}
		seenCallbacks[node] = struct{}{}
		if ref := mapValue(node, "$ref"); ref != nil {
			if target := resolveLocalNode(root, ref.Value); target != nil {
				callback(target, localReferencePath(ref.Value))
			}
		}
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i].Value
			if key != "$ref" && !strings.HasPrefix(key, "x-") {
				pathItem(node.Content[i+1], appendPath(path, key))
			}
		}
	}
	pathItem = func(node *yaml.Node, path []string) {
		if node == nil || node.Kind != yaml.MappingNode {
			return
		}
		if _, ok := seen[node]; ok {
			return
		}
		seen[node] = struct{}{}
		if ref := mapValue(node, "$ref"); ref != nil {
			if target := resolveLocalNode(root, ref.Value); target != nil {
				pathItem(target, localReferencePath(ref.Value))
			}
		}
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i].Value
			if standardOperationKeys[key] || (key == "query" && strings.HasPrefix(version, "3.2")) {
				opPath := appendPath(path, key)
				refs[pointerKey(opPath)] = struct{}{}
				operation(node.Content[i+1], opPath)
			}
			if key == "additionalOperations" && strings.HasPrefix(version, "3.2") && node.Content[i+1].Kind == yaml.MappingNode {
				additional := node.Content[i+1]
				for j := 0; j+1 < len(additional.Content); j += 2 {
					opPath := appendPath(appendPath(path, key), additional.Content[j].Value)
					refs[pointerKey(opPath)] = struct{}{}
					operation(additional.Content[j+1], opPath)
				}
			}
		}
	}
	for _, top := range []string{"paths", "webhooks"} {
		container := mapValue(root, top)
		if container != nil && container.Kind == yaml.MappingNode {
			for i := 0; i+1 < len(container.Content); i += 2 {
				pathItem(container.Content[i+1], []string{top, container.Content[i].Value})
			}
		}
	}
	if components := mapValue(root, "components"); components != nil {
		if items := mapValue(components, "pathItems"); items != nil && items.Kind == yaml.MappingNode {
			for i := 0; i+1 < len(items.Content); i += 2 {
				pathItem(items.Content[i+1], []string{"components", "pathItems", items.Content[i].Value})
			}
		}
		if callbacks := mapValue(components, "callbacks"); callbacks != nil && callbacks.Kind == yaml.MappingNode {
			for i := 0; i+1 < len(callbacks.Content); i += 2 {
				callback(callbacks.Content[i+1], []string{"components", "callbacks", callbacks.Content[i].Value})
			}
		}
	}
	return ids, refs
}

func danglingLinkWarnings(root *yaml.Node, beforeIDs, afterIDs, beforeRefs, afterRefs map[string]struct{}) []Warning {
	var warnings []Warning
	var walk func(*yaml.Node, []string)
	walk = func(node *yaml.Node, path []string) {
		if node == nil {
			return
		}
		if node.Kind == yaml.MappingNode {
			link := isLinkObjectPath(path)
			if id := mapValue(node, "operationId"); link && id != nil && id.Kind == yaml.ScalarNode {
				if _, existed := beforeIDs[id.Value]; existed {
					if _, remains := afterIDs[id.Value]; !remains {
						warnings = append(warnings, Warning{Path: jsonPath(appendPath(path, "operationId")), Message: fmt.Sprintf("link targets filtered operationId %q", id.Value), owner: componentOwnerFromPath(path)})
					}
				}
			}
			if ref := mapValue(node, "operationRef"); link && ref != nil && ref.Kind == yaml.ScalarNode && strings.HasPrefix(ref.Value, "#/") {
				key := pointerKey(localReferencePath(ref.Value))
				if _, existed := beforeRefs[key]; existed {
					if _, remains := afterRefs[key]; !remains {
						warnings = append(warnings, Warning{Path: jsonPath(appendPath(path, "operationRef")), Message: fmt.Sprintf("link targets filtered operation %q", ref.Value), owner: componentOwnerFromPath(path)})
					}
				}
			}
			for i := 0; i+1 < len(node.Content); i += 2 {
				walk(node.Content[i+1], appendPath(path, node.Content[i].Value))
			}
		} else if node.Kind == yaml.SequenceNode {
			for i, child := range node.Content {
				walk(child, appendPath(path, fmt.Sprintf("%d", i)))
			}
		}
	}
	walk(root, nil)
	return warnings
}

func componentOwnerFromPath(path []string) *ComponentID {
	if len(path) < 3 || path[0] != "components" {
		return nil
	}
	id := ComponentID{Section: path[1], Name: path[2]}
	return &id
}

func localReferencePath(ref string) []string {
	tokens, err := localPointerTokens(ref)
	if err != nil {
		return nil
	}
	return tokens
}

func pointerKey(path []string) string {
	return strings.Join(path, "\x00")
}
