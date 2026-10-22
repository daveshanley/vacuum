// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package utils

import (
	"github.com/daveshanley/vacuum/model"
	"strconv"
	"strings"

	"github.com/pb33f/jsonpath/pkg/jsonpath"
)

// AppendResultPathSegment appends a mapping key using the shared result-path syntax.
func AppendResultPathSegment(basePath, key string) string {
	return model.AppendResultPathSegment(basePath, key)
}

// AppendResultPathIndex appends a sequence index using the shared result-path syntax.
func AppendResultPathIndex(basePath string, index int) string {
	return model.AppendResultPathIndex(basePath, index)
}

// IsSimpleResultPathKey reports whether dot notation can represent a key.
func IsSimpleResultPathKey(key string) bool { return model.IsSimpleResultPathKey(key) }

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
