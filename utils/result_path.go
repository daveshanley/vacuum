// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package utils

import (
	"strconv"
	"strings"

	"github.com/pb33f/jsonpath/pkg/jsonpath"
)

// AppendResultPathSegment appends a mapping key to a vacuum result path.
// It uses bracket notation for non-simple keys and escapes quoted keys.
func AppendResultPathSegment(basePath, key string) string {
	if IsSimpleResultPathKey(key) {
		return basePath + "." + key
	}
	if strings.ContainsAny(key, "'\\\n\r\t") {
		return basePath + "[" + strconv.Quote(key) + "]"
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

// CanonicalResultPath normalizes a singular JSONPath for comparisons without changing literal keys.
func CanonicalResultPath(path string) string {
	parsed, err := jsonpath.NewPath(path)
	if err != nil || !parsed.IsSingular() {
		return path
	}
	segments, err := parsed.GetSegmentInfo()
	if err != nil {
		return path
	}
	var normalized strings.Builder
	normalized.WriteByte('$')
	for _, segment := range segments {
		normalized.WriteByte('[')
		if segment.Kind == jsonpath.SegmentKindArrayIndex {
			normalized.WriteString(strconv.FormatInt(segment.Index, 10))
		} else {
			normalized.WriteString(strconv.Quote(segment.Key))
		}
		normalized.WriteByte(']')
	}
	return normalized.String()
}

// CanonicalSchemaPath uses Doctor bracket notation for schema names and property names.
func CanonicalSchemaPath(path string) string {
	if !strings.Contains(path, ".components.schemas.") && !strings.Contains(path, ".properties.") && !strings.Contains(path, ".patternProperties.") {
		return path
	}
	parsed, err := jsonpath.NewPath(path)
	if err != nil || !parsed.IsSingular() {
		return path
	}
	segments, err := parsed.GetSegmentInfo()
	if err != nil {
		return path
	}
	normalized := "$"
	for i, segment := range segments {
		if segment.Kind == jsonpath.SegmentKindArrayIndex {
			normalized = AppendResultPathIndex(normalized, int(segment.Index))
			continue
		}
		schemaName := i == 2 && segments[0].Key == "components" && segments[1].Key == "schemas"
		propertyName := i > 0 && (segments[i-1].Key == "properties" || segments[i-1].Key == "patternProperties")
		if (schemaName || propertyName) && IsSimpleResultPathKey(segment.Key) {
			normalized += "['" + segment.Key + "']"
		} else {
			normalized = AppendResultPathSegment(normalized, segment.Key)
		}
	}
	return normalized
}
