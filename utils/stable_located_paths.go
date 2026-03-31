// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package utils

import "sort"

// buildStablePrimaryAndPaths keeps the canonical path as the primary identity for a result
// and returns a stable list of alternate locations for rendering/debugging.
func buildStablePrimaryAndPaths(canonicalPath string, locatedPaths []string) (primaryPath string, allPaths []string) {
	seen := make(map[string]struct{}, len(locatedPaths)+1)

	if canonicalPath != "" {
		seen[canonicalPath] = struct{}{}
	}

	extraPaths := make([]string, 0, len(locatedPaths))
	for _, path := range locatedPaths {
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		extraPaths = append(extraPaths, path)
	}

	sort.Strings(extraPaths)

	if canonicalPath == "" {
		if len(extraPaths) == 0 {
			return "", nil
		}
		return extraPaths[0], extraPaths
	}

	allPaths = make([]string, 0, 1+len(extraPaths))
	allPaths = append(allPaths, canonicalPath)
	allPaths = append(allPaths, extraPaths...)
	return canonicalPath, allPaths
}
