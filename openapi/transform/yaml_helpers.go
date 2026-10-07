// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package transform

import (
	"fmt"
	"strings"

	"github.com/pb33f/go-yaml"
)

var namedObjectMapKeys = map[string]struct{}{
	"callbacks":           {},
	"content":             {},
	"definitions":         {},
	"dependentSchemas":    {},
	"encoding":            {},
	"examples":            {},
	"headers":             {},
	"links":               {},
	"mediaTypes":          {},
	"parameters":          {},
	"pathItems":           {},
	"paths":               {},
	"patternProperties":   {},
	"properties":          {},
	"requestBodies":       {},
	"responses":           {},
	"schemas":             {},
	"securityDefinitions": {},
	"securitySchemes":     {},
	"webhooks":            {},
	"$defs":               {},
}

func documentRoot(node *yaml.Node) *yaml.Node {
	if node != nil && node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return node.Content[0]
	}
	return node
}

func mapValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func removeMapPair(node *yaml.Node, index int) {
	copy(node.Content[index:], node.Content[index+2:])
	clear(node.Content[len(node.Content)-2:])
	node.Content = node.Content[:len(node.Content)-2]
}

func jsonPath(parts []string) string {
	var b strings.Builder
	b.WriteByte('$')
	for _, part := range parts {
		if isPathIdentifier(part) {
			b.WriteByte('.')
			b.WriteString(part)
			continue
		}
		b.WriteString("['")
		for _, char := range part {
			if char == '\\' || char == '\'' {
				b.WriteByte('\\')
			}
			b.WriteRune(char)
		}
		b.WriteString("']")
	}
	return b.String()
}

func isPathIdentifier(part string) bool {
	if part == "" {
		return false
	}
	for i, char := range part {
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char == '_' || char == '$' || i > 0 && char >= '0' && char <= '9' {
			continue
		}
		return false
	}
	return true
}

func requireMappingRoot(root *yaml.Node) (*yaml.Node, error) {
	root = documentRoot(root)
	if root == nil || root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("OpenAPI document root must be a mapping")
	}
	return root, nil
}

// isWithinArbitraryExample reports whether path is inside example data rather
// than an OpenAPI object. Example values may contain keys such as $ref that are
// ordinary payload data and must not participate in document reference checks.
func isWithinArbitraryExample(path []string) bool {
	for i, part := range path {
		switch part {
		case "example", "default", "const", "enum":
			if !isNamedObjectMapEntry(path, i) {
				return true
			}
		case "value":
			if i >= 2 && path[i-2] == "examples" {
				return true
			}
		}
	}
	return false
}

// isArbitraryData distinguishes schema examples arrays from the named Example
// Object maps used by media types and components. A numeric key is still a name.
func isArbitraryData(node *yaml.Node, path []string) bool {
	if isWithinArbitraryExample(path) {
		return true
	}
	last := len(path) - 1
	return node != nil && node.Kind == yaml.SequenceNode && last >= 0 &&
		path[last] == "examples" && !isNamedObjectMapEntry(path, last)
}

// isWithinExtension reports whether path is inside an OpenAPI specification
// extension. Extension values are arbitrary, but local component references in
// them are still followed conservatively by the pruning graph.
func isWithinExtension(path []string) bool {
	for i, part := range path {
		if strings.HasPrefix(part, "x-") && !isNamedObjectMapEntry(path, i) {
			return true
		}
	}
	return false
}

func isNamedObjectMapEntry(path []string, index int) bool {
	if index == 0 {
		return false
	}
	_, ok := namedObjectMapKeys[path[index-1]]
	return ok
}

// Expansion bounds limit malicious alias graphs without penalizing ordinary
// documents, which take the no-copy path when no expansion is needed.
const (
	maxExpandedYAMLNodes  = 1_000_000
	maxYAMLExpansionDepth = 1_000
)

// expandYAMLReferences expands merge keys on a private node tree. Filtering also
// materializes aliases, giving each use site independent content. Pruning keeps
// ordinary aliases connected to their cloned anchor owners for reachability.
// Unlike decoding through Go maps, this preserves order, comments and styles.
func expandYAMLReferences(root *yaml.Node, aliases bool) (*yaml.Node, error) {
	inspection := yamlReferenceInspection{
		expandAliases: aliases,
		active:        make(map[*yaml.Node]bool),
		heights:       make(map[*yaml.Node]int),
	}
	needed, err := inspection.inspect(root, 0)
	if err != nil {
		return nil, fmt.Errorf("cannot expand YAML aliases and merge keys: %w", err)
	}
	// Retained aliases still need a bounded traversal when reference validation
	// follows them at each use site, even when no physical expansion is needed.
	if inspection.hasAliases && expandedYAMLSize(root, make(map[*yaml.Node]int)) > maxExpandedYAMLNodes {
		return nil, fmt.Errorf("cannot expand YAML aliases and merge keys: YAML expansion exceeds maximum node count %d", maxExpandedYAMLNodes)
	}
	if !needed {
		return root, nil
	}
	expander := yamlExpander{clones: make(map[*yaml.Node]*yaml.Node)}
	expanded, err := expander.clone(root, aliases, 0)
	if err != nil {
		return nil, fmt.Errorf("cannot expand YAML aliases and merge keys: %w", err)
	}
	return expanded, nil
}

// yamlReferenceInspection checks each shared subtree once while tracking its
// height. Active nodes detect cycles; cached heights still enforce depth limits
// when a completed subtree is reached through a longer alias chain.
type yamlReferenceInspection struct {
	expandAliases bool
	hasAliases    bool
	active        map[*yaml.Node]bool
	heights       map[*yaml.Node]int
}

func (v *yamlReferenceInspection) inspect(node *yaml.Node, depth int) (bool, error) {
	if node == nil {
		return false, fmt.Errorf("nil YAML node")
	}
	if v.active[node] {
		return false, fmt.Errorf("recursive YAML alias or node graph")
	}
	if depth+v.heights[node] > maxYAMLExpansionDepth {
		return false, fmt.Errorf("YAML expansion exceeds maximum depth %d", maxYAMLExpansionDepth)
	}
	if v.heights[node] != 0 {
		return false, nil
	}
	v.active[node] = true
	defer delete(v.active, node)
	height := 1
	needed := node.Tag == "!!merge" || v.expandAliases && node.Kind == yaml.AliasNode
	if node.Kind == yaml.AliasNode {
		v.hasAliases = true
		more, err := v.inspect(node.Alias, depth+1)
		if err != nil {
			return false, err
		}
		needed = needed || more
		height = max(height, v.heights[node.Alias]+1)
	}
	if node.Kind == yaml.MappingNode && len(node.Content)%2 != 0 {
		return false, fmt.Errorf("YAML mapping has an unmatched key")
	}
	for _, child := range node.Content {
		more, err := v.inspect(child, depth+1)
		if err != nil {
			return false, err
		}
		needed = needed || more
		height = max(height, v.heights[child]+1)
	}
	v.heights[node] = height
	return needed, nil
}

// expandedYAMLSize rejects exponentially expanding aliases before allocating
// their copies. Counts saturate at the limit, so hostile graphs cannot overflow.
func expandedYAMLSize(node *yaml.Node, sizes map[*yaml.Node]int) int {
	if size := sizes[node]; size != 0 {
		return size
	}
	size := 1
	if node.Kind == yaml.AliasNode {
		size += expandedYAMLSize(node.Alias, sizes)
	}
	for _, child := range node.Content {
		size += expandedYAMLSize(child, sizes)
		if size > maxExpandedYAMLNodes {
			size = maxExpandedYAMLNodes + 1
			break
		}
	}
	sizes[node] = size
	return size
}

// yamlExpander preserves alias identity only when aliases are retained. Expanded
// aliases and merges must instead have independent nodes at every use site.
type yamlExpander struct {
	clones map[*yaml.Node]*yaml.Node
	nodes  int
}

func (e *yamlExpander) clone(node *yaml.Node, aliases bool, depth int) (*yaml.Node, error) {
	if depth > maxYAMLExpansionDepth {
		return nil, fmt.Errorf("YAML expansion exceeds maximum depth %d", maxYAMLExpansionDepth)
	}
	if !aliases {
		if clone := e.clones[node]; clone != nil {
			return clone, nil
		}
	}
	e.nodes++
	if e.nodes > maxExpandedYAMLNodes {
		return nil, fmt.Errorf("YAML expansion exceeds maximum node count %d", maxExpandedYAMLNodes)
	}
	if aliases && node.Kind == yaml.AliasNode {
		clone, err := e.clone(node.Alias, true, depth+1)
		if err != nil {
			return nil, err
		}
		clone.HeadComment = joinYAMLComments(clone.HeadComment, node.HeadComment)
		clone.LineComment = joinYAMLComments(clone.LineComment, node.LineComment)
		clone.FootComment = joinYAMLComments(clone.FootComment, node.FootComment)
		return clone, nil
	}
	clone := *node
	clone.Content = nil
	clone.Alias = nil
	if aliases {
		// Copies of an anchored source must not emit duplicate anchor names.
		clone.Anchor = ""
	} else {
		e.clones[node] = &clone
	}
	if node.Kind == yaml.AliasNode {
		target, err := e.clone(node.Alias, false, depth+1)
		if err != nil {
			return nil, err
		}
		clone.Alias = target
	}
	if node.Kind == yaml.MappingNode {
		if err := e.cloneMapping(node, &clone, aliases, depth+1); err != nil {
			return nil, err
		}
		return &clone, nil
	}
	for _, child := range node.Content {
		copy, err := e.clone(child, aliases, depth+1)
		if err != nil {
			return nil, err
		}
		clone.Content = append(clone.Content, copy)
	}
	return &clone, nil
}

// cloneMapping inserts inherited keys at the merge location. Local keys always
// win; within a merge sequence the first source wins, as required by YAML.
func (e *yamlExpander) cloneMapping(source, target *yaml.Node, aliases bool, depth int) error {
	seen := make(map[string]struct{}, len(source.Content)/2)
	for i := 0; i < len(source.Content); i += 2 {
		key := source.Content[i]
		if key.Kind != yaml.ScalarNode {
			return fmt.Errorf("OpenAPI mappings require scalar keys")
		}
		if key.Tag != "!!merge" {
			if _, exists := seen[key.Value]; exists {
				return fmt.Errorf("duplicate YAML mapping key %q", key.Value)
			}
			seen[key.Value] = struct{}{}
		}
	}
	for i := 0; i < len(source.Content); i += 2 {
		key, value := source.Content[i], source.Content[i+1]
		if key.Tag == "!!merge" {
			merged, err := e.clone(value, true, depth)
			if err != nil {
				return err
			}
			start := len(target.Content)
			if err := appendYAMLMerge(target, merged, seen); err != nil {
				return err
			}
			// The merge directive disappears, but its comments remain attached
			// to the inherited entries, or to the map when all were overridden.
			comments := joinYAMLComments(key.HeadComment, key.LineComment, key.FootComment, value.HeadComment, value.LineComment, value.FootComment)
			if len(target.Content) > start {
				target.Content[start].HeadComment = joinYAMLComments(comments, target.Content[start].HeadComment)
			} else {
				target.FootComment = joinYAMLComments(target.FootComment, comments)
			}
			continue
		}
		for _, child := range []*yaml.Node{key, value} {
			copy, err := e.clone(child, aliases, depth)
			if err != nil {
				return err
			}
			target.Content = append(target.Content, copy)
		}
	}
	return nil
}

func appendYAMLMerge(target, source *yaml.Node, seen map[string]struct{}) error {
	sources := []*yaml.Node{source}
	if source.Kind == yaml.SequenceNode {
		sources = source.Content
	}
	for _, mapping := range sources {
		if mapping.Kind != yaml.MappingNode {
			return fmt.Errorf("YAML merge source must be a mapping or sequence of mappings")
		}
		for i := 0; i < len(mapping.Content); i += 2 {
			key := mapping.Content[i]
			if _, exists := seen[key.Value]; !exists {
				seen[key.Value] = struct{}{}
				target.Content = append(target.Content, key, mapping.Content[i+1])
			}
		}
	}
	return nil
}

func joinYAMLComments(comments ...string) string {
	var retained []string
	for _, comment := range comments {
		if comment != "" {
			retained = append(retained, comment)
		}
	}
	return strings.Join(retained, "\n")
}
