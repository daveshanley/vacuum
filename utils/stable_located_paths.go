// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package utils

import (
	"sort"
	"strings"
)

// buildStablePrimaryAndPaths keeps a real component definition as the primary
// identity, but treats concrete aliases as sortable candidates.
func buildStablePrimaryAndPaths(canonicalPath string, locatedPaths []string) (primaryPath string, allPaths []string) {
	seen := make(map[string]struct{}, len(locatedPaths)+1)

	candidates := make([]string, 0, len(locatedPaths)+1)
	if canonicalPath != "" {
		seen[canonicalPath] = struct{}{}
		candidates = append(candidates, canonicalPath)
	}

	for _, path := range locatedPaths {
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		candidates = append(candidates, path)
	}

	if len(candidates) == 0 {
		return "", nil
	}

	sort.Strings(candidates)
	if isComponentDefinitionPath(canonicalPath) {
		allPaths = make([]string, 0, len(candidates))
		allPaths = append(allPaths, canonicalPath)
		for _, path := range candidates {
			if path != canonicalPath {
				allPaths = append(allPaths, path)
			}
		}
		return canonicalPath, allPaths
	}

	return candidates[0], candidates
}

func isComponentDefinitionPath(path string) bool {
	return strings.HasPrefix(path, "$.components.") ||
		strings.HasPrefix(path, "$.definitions.") ||
		strings.HasPrefix(path, "$.parameters.") ||
		strings.HasPrefix(path, "$.responses.") ||
		strings.HasPrefix(path, "$.securityDefinitions.")
}
