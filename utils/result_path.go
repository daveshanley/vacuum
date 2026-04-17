// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package utils

import "strconv"

// AppendResultPathSegment appends a mapping key to a vacuum result path.
// It intentionally mirrors the historical path formatting used across vacuum
// and doctor, including bracket notation for non-simple keys.
func AppendResultPathSegment(basePath, key string) string {
	if IsSimpleResultPathKey(key) {
		return basePath + "." + key
	}
	return basePath + "['" + key + "']"
}

// AppendResultPathIndex appends a sequence index to a vacuum result path.
func AppendResultPathIndex(basePath string, index int) string {
	return basePath + "[" + strconv.Itoa(index) + "]"
}

// IsSimpleResultPathKey reports whether a key can be represented using dot
// notation instead of bracket notation.
func IsSimpleResultPathKey(key string) bool {
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
