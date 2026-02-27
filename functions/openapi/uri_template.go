// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package openapi

import (
	"strconv"
	"strings"
)

// TemplateSegment represents a parsed RFC 6570 URI template expression.
type TemplateSegment struct {
	Raw        string // original input, e.g. "{;id*}"
	Operator   string // RFC 6570 operator: +, #, ., /, ;, ?, & or ""
	Name       string // variable name with operator/modifiers stripped
	Explode    bool   // has * modifier
	Prefix     int    // :N modifier value, 0 if absent
	IsVariable bool   // true if wrapped in {}
}

// ParseTemplateSegment parses a single URI template segment (e.g. "{;id*}", "{+path}", "literal").
// It handles all RFC 6570 operators and modifiers using pure string manipulation.
func ParseTemplateSegment(segment string) TemplateSegment {
	ts := TemplateSegment{Raw: segment}

	if len(segment) < 3 || segment[0] != '{' || segment[len(segment)-1] != '}' {
		return ts
	}

	ts.IsVariable = true
	inner := segment[1 : len(segment)-1]

	// Check first char for RFC 6570 operator
	if len(inner) > 0 {
		switch inner[0] {
		case '+', '#', '.', '/', ';', '?', '&':
			ts.Operator = string(inner[0])
			inner = inner[1:]
		}
	}

	// Check trailing * for explode modifier
	if len(inner) > 0 && inner[len(inner)-1] == '*' {
		ts.Explode = true
		inner = inner[:len(inner)-1]
	}

	// Check for :N prefix modifier
	if colonIdx := strings.LastIndexByte(inner, ':'); colonIdx >= 0 {
		suffix := inner[colonIdx+1:]
		if n, err := strconv.Atoi(suffix); err == nil && n > 0 {
			ts.Prefix = n
			inner = inner[:colonIdx]
		}
	}

	ts.Name = inner
	return ts
}

// operatorTokens maps RFC 6570 operators to safe normalization tokens.
// Using raw operator chars like ";" or "/" would corrupt path structure when splitting on "/".
var operatorTokens = map[string]string{
	"+": "%op_plus",
	"#": "%op_hash",
	".": "%op_dot",
	"/": "%op_slash",
	";": "%op_semi",
	"?": "%op_query",
	"&": "%op_amp",
}

// normalizeTemplateParam returns a normalization token for a parsed template variable.
// Variables with different RFC 6570 operators produce different tokens, preventing false duplicates.
func normalizeTemplateParam(tv TemplateSegment) string {
	if tv.Operator != "" {
		if tok, ok := operatorTokens[tv.Operator]; ok {
			return tok
		}
	}
	return "%"
}

// isTemplateSegment checks if a path segment contains an RFC 6570 template expression.
// It handles both full-segment variables like {;id} and embedded expressions like {id}.json.
func isTemplateSegment(seg string) bool {
	braceOpen := strings.IndexByte(seg, '{')
	if braceOpen < 0 {
		return false
	}
	braceClose := strings.IndexByte(seg[braceOpen+1:], '}')
	if braceClose < 0 {
		return false
	}
	tv := ParseTemplateSegment(seg[braceOpen : braceOpen+1+braceClose+1])
	return tv.IsVariable
}
