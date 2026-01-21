// Copyright 2024-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package utils

import (
	"errors"
	"strconv"
	"strings"

	"github.com/pb33f/libopenapi/utils"
	"go.yaml.in/yaml/v4"
)

// SegmentType indicates what kind of path segment this is
type SegmentType int

const (
	SegmentKey        SegmentType = iota // Simple key: "name"
	SegmentArrayIndex                    // Numeric index: [0]
	SegmentMapKey                        // String key: ['200']
)

// PathSegment represents a single segment in a field path
type PathSegment struct {
	Type  SegmentType
	Key   string // Used for SegmentKey and SegmentMapKey
	Index int    // Used only for SegmentArrayIndex
}

// FieldPathResult contains the results of a field path lookup
type FieldPathResult struct {
	KeyNode   *yaml.Node // For error reporting (line/column)
	ValueNode *yaml.Node // For validation
	Found     bool
}

// FieldPathOptions controls the behavior of FindFieldPath
type FieldPathOptions struct {
	// RecursiveFirstSegment uses FindKeyNode (recursive) for the first segment
	// instead of FindKeyNodeTop (non-recursive).
	//
	// Background: libopenapi provides two lookup functions:
	//   - FindKeyNodeTop: searches only the immediate children (non-recursive)
	//   - FindKeyNode: searches immediate children AND recursively into nested maps
	//
	// Historical Usage in vacuum functions:
	//   - FindKeyNodeTop (non-recursive): truthy, schema, success_response, pattern
	//   - FindKeyNode (recursive): falsy, defined, undefined, length
	//
	// The recursive variant allows finding keys that may be nested within the
	// first level of the node tree. This is useful when the JSONPath 'given'
	// lands on a parent node and the field might be in a child structure.
	//
	// Set to true for backward compatibility with functions that previously
	// used FindKeyNode. New code should generally use false (non-recursive)
	// for more predictable behavior.
	RecursiveFirstSegment bool
}

// ParseFieldPath parses a field path string into segments.
// Returns an error for invalid syntax (fail fast, better UX).
//
// Supported syntax:
//   - Simple key: "name"
//   - Dot notation: "a.b.c"
//   - Escaped dot: "some\.key" -> key is "some.key"
//   - Escaped backslash: "a\\b" -> key is "a\b"
//   - Numeric index: "items[0]"
//   - String index: "responses['200']" or "responses[\"200\"]"
//   - Combined: "items[0].type", "paths['/pet'].get"
//
// Errors are returned for:
//   - Empty segments (e.g., "a..b")
//   - Path starting with index (e.g., "[0].name")
//   - Unclosed brackets
//   - Mismatched quotes in brackets
//   - Non-numeric content in brackets without quotes (e.g., "items[abc]")
//   - Quotes appearing in bracket string keys (no escape sequences inside brackets)
func ParseFieldPath(fieldPath string) ([]PathSegment, error) {
	if fieldPath == "" {
		return nil, nil
	}

	// Performance fast path: if no special characters, return single key segment
	if strings.IndexAny(fieldPath, ".[]\\") == -1 {
		return []PathSegment{{Type: SegmentKey, Key: fieldPath}}, nil
	}

	// Check if path starts with an index (invalid)
	if fieldPath[0] == '[' {
		return nil, errors.New("field path cannot start with an index; use 'key[0]' instead of '[0]'")
	}

	// Pre-allocate with reasonable capacity - most paths have < 4 segments
	segments := make([]PathSegment, 0, 4)
	var currentKey strings.Builder
	currentKey.Grow(len(fieldPath) / 2) // reasonable estimate for key size
	i := 0
	n := len(fieldPath)

	for i < n {
		ch := fieldPath[i]

		switch ch {
		case '\\':
			// Escape sequence
			if i+1 < n {
				nextCh := fieldPath[i+1]
				if nextCh == '.' || nextCh == '\\' {
					// \. -> literal dot, \\ -> literal backslash
					currentKey.WriteByte(nextCh)
					i += 2
				} else {
					// \x -> literal \x (backslash preserved)
					currentKey.WriteByte('\\')
					currentKey.WriteByte(nextCh)
					i += 2
				}
			} else {
				// Trailing backslash -> include literally (lenient)
				currentKey.WriteByte('\\')
				i++
			}

		case '.':
			// Dot separator - end current key segment
			key := currentKey.String()
			if key == "" {
				return nil, errors.New("empty segment in field path (consecutive dots)")
			}
			segments = append(segments, PathSegment{Type: SegmentKey, Key: key})
			currentKey.Reset()
			i++

		case '[':
			// Start of bracket notation
			// First, if we have accumulated a key, add it as a segment
			if currentKey.Len() > 0 {
				segments = append(segments, PathSegment{Type: SegmentKey, Key: currentKey.String()})
				currentKey.Reset()
			} else if len(segments) == 0 {
				// No key before bracket and no previous segments
				return nil, errors.New("field path cannot start with an index; use 'key[0]' instead of '[0]'")
			}

			// Parse bracket content
			i++ // skip '['
			if i >= n {
				return nil, errors.New("unclosed bracket in field path")
			}

			bracketCh := fieldPath[i]
			if bracketCh == '\'' || bracketCh == '"' {
				// String index: ['key'] or ["key"]
				quote := bracketCh
				i++ // skip opening quote
				startIdx := i

				// Find closing quote
				for i < n && fieldPath[i] != quote {
					// Check for quotes inside the string (not allowed)
					if fieldPath[i] == '\'' || fieldPath[i] == '"' {
						if fieldPath[i] != quote {
							return nil, errors.New("quotes cannot appear inside bracket string keys")
						}
					}
					i++
				}
				if i >= n {
					return nil, errors.New("unclosed quote in bracket notation")
				}

				key := fieldPath[startIdx:i]
				i++ // skip closing quote

				if i >= n || fieldPath[i] != ']' {
					return nil, errors.New("expected ']' after quoted string in bracket notation")
				}
				i++ // skip ']'

				segments = append(segments, PathSegment{Type: SegmentMapKey, Key: key})

				// After bracket, if there's a dot, skip it (it's just a separator)
				if i < n && fieldPath[i] == '.' {
					i++
				}

			} else {
				// Numeric index: [0]
				startIdx := i

				// Find closing bracket
				for i < n && fieldPath[i] != ']' {
					i++
				}
				if i >= n {
					return nil, errors.New("unclosed bracket in field path")
				}

				content := fieldPath[startIdx:i]
				i++ // skip ']'

				// Parse as integer
				index, err := strconv.Atoi(content)
				if err != nil {
					if content == "" {
						return nil, errors.New("empty index in bracket notation")
					}
					// Check for negative index
					if len(content) > 0 && content[0] == '-' {
						return nil, errors.New("negative indices are not supported")
					}
					return nil, errors.New("non-numeric content in brackets without quotes; use ['key'] for string keys")
				}
				if index < 0 {
					return nil, errors.New("negative indices are not supported")
				}

				segments = append(segments, PathSegment{Type: SegmentArrayIndex, Index: index})
			}

			// After bracket, if there's a dot, skip it (it's just a separator)
			if i < n && fieldPath[i] == '.' {
				i++
			}

		default:
			// Regular character - add to current key
			currentKey.WriteByte(ch)
			i++
		}
	}

	// Add final key segment if any
	if currentKey.Len() > 0 {
		segments = append(segments, PathSegment{Type: SegmentKey, Key: currentKey.String()})
	}

	return segments, nil
}

// FindFieldPath navigates a YAML node tree using a field path.
// It supports dot-notation, escape sequences, array indices, and string indices.
//
// The function returns:
//   - KeyNode: The YAML node representing the final key (for line/column info)
//   - ValueNode: The YAML node representing the value at the path
//   - Found: Whether the path was successfully resolved
//
// Use opts.RecursiveFirstSegment=true for backward compatibility with functions
// that currently use FindKeyNode (recursive) for the first lookup.
func FindFieldPath(fieldPath string, nodes []*yaml.Node, opts FieldPathOptions) FieldPathResult {
	if fieldPath == "" {
		return FieldPathResult{Found: false}
	}

	// Performance fast path: if no special characters, use original lookup directly
	if strings.IndexAny(fieldPath, ".[]\\") == -1 {
		var keyNode, valueNode *yaml.Node
		if opts.RecursiveFirstSegment {
			keyNode, valueNode = utils.FindKeyNode(fieldPath, nodes)
		} else {
			keyNode, valueNode = utils.FindKeyNodeTop(fieldPath, nodes)
		}
		return FieldPathResult{KeyNode: keyNode, ValueNode: valueNode, Found: keyNode != nil}
	}

	// Parse the field path
	segments, err := ParseFieldPath(fieldPath)
	if err != nil || len(segments) == 0 {
		return FieldPathResult{Found: false}
	}

	currentNodes := nodes
	var keyNode, valueNode *yaml.Node

	for i, segment := range segments {
		switch segment.Type {
		case SegmentKey, SegmentMapKey:
			// Look up key in current nodes
			if i == 0 && opts.RecursiveFirstSegment {
				keyNode, valueNode = utils.FindKeyNode(segment.Key, currentNodes)
			} else {
				keyNode, valueNode = utils.FindKeyNodeTop(segment.Key, currentNodes)
			}

			if keyNode == nil {
				return FieldPathResult{Found: false}
			}

			// If not the last segment, continue traversing
			if i < len(segments)-1 {
				nextSeg := segments[i+1]
				if nextSeg.Type == SegmentArrayIndex {
					// Next segment is array index - valueNode should be array
					if valueNode == nil || !utils.IsNodeArray(valueNode) {
						return FieldPathResult{Found: false}
					}
				} else {
					// Next segment is key - valueNode should be map
					if valueNode == nil || !utils.IsNodeMap(valueNode) {
						return FieldPathResult{Found: false}
					}
				}
				currentNodes = valueNode.Content
			}

		case SegmentArrayIndex:
			// Access array element by index
			// currentNodes should be the Content of an array node
			// But we need to check if we're at the array level
			if len(currentNodes) <= segment.Index {
				return FieldPathResult{Found: false}
			}

			// For arrays, Content is a flat list of elements (not key-value pairs)
			valueNode = currentNodes[segment.Index]
			keyNode = valueNode // For arrays, use the element itself as the key node

			// If not the last segment, continue traversing
			if i < len(segments)-1 {
				nextSeg := segments[i+1]
				if nextSeg.Type == SegmentArrayIndex {
					// Next segment is another array index
					if valueNode == nil || !utils.IsNodeArray(valueNode) {
						return FieldPathResult{Found: false}
					}
				} else {
					// Next segment is key - valueNode should be map
					if valueNode == nil || !utils.IsNodeMap(valueNode) {
						return FieldPathResult{Found: false}
					}
				}
				currentNodes = valueNode.Content
			}
		}
	}

	return FieldPathResult{KeyNode: keyNode, ValueNode: valueNode, Found: true}
}
