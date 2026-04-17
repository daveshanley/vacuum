// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package utils

import (
	"time"

	"github.com/daveshanley/vacuum/model"
	openapiUtils "github.com/pb33f/libopenapi/utils"
	"go.yaml.in/yaml/v4"
)

// IgnoreMatcherOptions controls how ignore expressions are resolved.
type IgnoreMatcherOptions struct {
	// RootNode is the unresolved document root used to resolve JSONPath ignore expressions.
	RootNode *yaml.Node
	// SpecBytes is an optional fallback used to rebuild a YAML tree when RootNode is not available.
	SpecBytes []byte
	// LookupTimeout controls how long JSONPath expression lookup may run.
	LookupTimeout time.Duration
}

// IgnoreMatcher resolves ignore rules once and then performs fast per-result checks.
// Exact literal matching is always preserved for backward compatibility.
type IgnoreMatcher struct {
	literalByRule  map[string]map[string]struct{}
	resolvedByRule map[string]map[string]struct{}
}

// NewIgnoreMatcher builds a matcher from ignored items and an optional document root.
func NewIgnoreMatcher(ignored model.IgnoredItems, options IgnoreMatcherOptions) *IgnoreMatcher {
	matcher := &IgnoreMatcher{
		literalByRule:  make(map[string]map[string]struct{}),
		resolvedByRule: make(map[string]map[string]struct{}),
	}
	if len(ignored) == 0 {
		return matcher
	}

	root := options.RootNode
	if root == nil && len(options.SpecBytes) > 0 {
		var parsed yaml.Node
		if err := yaml.Unmarshal(options.SpecBytes, &parsed); err == nil {
			root = &parsed
		}
	}

	var pathIndex *NodePathIndex
	var expressionCache map[string]map[string]struct{}
	if root != nil {
		pathIndex = BuildNodePathIndex(root)
		expressionCache = make(map[string]map[string]struct{})
	}

	for ruleID, ignorePaths := range ignored {
		if len(ignorePaths) == 0 {
			continue
		}

		literalSet := make(map[string]struct{}, len(ignorePaths))
		var resolvedSet map[string]struct{}

		for _, ignorePath := range ignorePaths {
			if ignorePath == "" {
				continue
			}
			literalSet[ignorePath] = struct{}{}

			if root == nil || pathIndex == nil {
				continue
			}

			exactMatches := resolveIgnoreExpressionPaths(ignorePath, root, pathIndex, options.LookupTimeout, expressionCache)
			if len(exactMatches) == 0 {
				continue
			}
			if resolvedSet == nil {
				resolvedSet = make(map[string]struct{}, len(exactMatches))
			}
			for path := range exactMatches {
				resolvedSet[path] = struct{}{}
			}
		}

		if len(literalSet) > 0 {
			matcher.literalByRule[ruleID] = literalSet
		}
		if len(resolvedSet) > 0 {
			matcher.resolvedByRule[ruleID] = resolvedSet
		}
	}

	return matcher
}

// Matches reports whether a result should be ignored.
func (m *IgnoreMatcher) Matches(result *model.RuleFunctionResult) bool {
	if m == nil || result == nil {
		return false
	}

	ruleID := result.RuleId
	if ruleID == "" && result.Rule != nil {
		ruleID = result.Rule.Id
	}
	if ruleID == "" {
		return false
	}

	if matchesAnyPath(m.literalByRule[ruleID], result.Path, result.Paths) {
		return true
	}
	return matchesAnyPath(m.resolvedByRule[ruleID], result.Path, result.Paths)
}

func resolveIgnoreExpressionPaths(
	expression string,
	root *yaml.Node,
	pathIndex *NodePathIndex,
	timeout time.Duration,
	cache map[string]map[string]struct{},
) map[string]struct{} {
	if cached, ok := cache[expression]; ok {
		return cached
	}

	nodes, err := openapiUtils.FindNodesWithoutDeserializingWithOptions(root, expression, openapiUtils.JSONPathLookupOptions{
		Timeout: timeout,
	})
	if err != nil || len(nodes) == 0 {
		cache[expression] = nil
		return nil
	}

	matches := make(map[string]struct{}, len(nodes))
	for _, node := range nodes {
		if path, ok := pathIndex.Lookup(node); ok && path != "" {
			matches[path] = struct{}{}
		}
	}

	cache[expression] = matches
	return matches
}

func matchesAnyPath(allowed map[string]struct{}, primary string, alternates []string) bool {
	if len(allowed) == 0 {
		return false
	}
	if _, ok := allowed[primary]; ok {
		return true
	}
	for _, path := range alternates {
		if _, ok := allowed[path]; ok {
			return true
		}
	}
	return false
}
