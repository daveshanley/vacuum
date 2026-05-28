// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package upgrade

import (
	"strconv"
	"strings"
)

// NormalizeVersion removes the leading tag prefix used by Vacuum releases.
func NormalizeVersion(version string) string {
	version = strings.TrimSpace(version)
	version = strings.TrimPrefix(version, "v")
	return version
}

// CompareVersions compares simple stable semver values.
// It returns ok=false for unknown, source, or prerelease-looking versions.
func CompareVersions(a, b string) (int, bool) {
	av, ok := parseStableVersion(a)
	if !ok {
		return 0, false
	}
	bv, ok := parseStableVersion(b)
	if !ok {
		return 0, false
	}
	for i := range av {
		if av[i] > bv[i] {
			return 1, true
		}
		if av[i] < bv[i] {
			return -1, true
		}
	}
	return 0, true
}

// IsNewer reports whether latest is newer than current.
func IsNewer(latest, current string) bool {
	cmp, ok := CompareVersions(latest, current)
	return ok && cmp > 0
}

func IsComparableVersion(version string) bool {
	_, ok := parseStableVersion(version)
	return ok
}

func parseStableVersion(version string) ([3]uint64, bool) {
	version = NormalizeVersion(version)
	if version == "" || version == "unknown" || version == "(devel)" {
		return [3]uint64{}, false
	}
	if strings.ContainsAny(version, "-+") {
		return [3]uint64{}, false
	}

	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return [3]uint64{}, false
	}
	var parsed [3]uint64
	for i, part := range parts {
		if part == "" {
			return [3]uint64{}, false
		}
		v, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return [3]uint64{}, false
		}
		parsed[i] = v
	}
	return parsed, true
}
