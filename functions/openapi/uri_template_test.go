// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package openapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTemplateSegment(t *testing.T) {
	tests := []struct {
		input      string
		isVariable bool
		operator   string
		name       string
		explode    bool
		prefix     int
	}{
		// Simple expansion
		{"{id}", true, "", "id", false, 0},
		// RFC 6570 operators
		{"{;id}", true, ";", "id", false, 0},
		{"{+path}", true, "+", "path", false, 0},
		{"{#frag}", true, "#", "frag", false, 0},
		{"{.ext}", true, ".", "ext", false, 0},
		{"{/seg}", true, "/", "seg", false, 0},
		{"{?q}", true, "?", "q", false, 0},
		{"{&more}", true, "&", "more", false, 0},
		// Explode modifier
		{"{;id*}", true, ";", "id", true, 0},
		// Prefix modifier
		{"{id:3}", true, "", "id", false, 3},
		// Operator + explode
		{"{+path*}", true, "+", "path", true, 0},
		// Operator + prefix
		{"{;id:3}", true, ";", "id", false, 3},
		// Composite variables — kept intact
		{"{x,y}", true, "", "x,y", false, 0},
		{"{;x,y}", true, ";", "x,y", false, 0},
		// Literals
		{"literal", false, "", "", false, 0},
		{"", false, "", "", false, 0},
		// Too short / malformed
		{"{}", false, "", "", false, 0},
		{"{", false, "", "", false, 0},
		{"{a", false, "", "", false, 0},
		{"a}", false, "", "", false, 0},
		// Operator with empty name
		{"{;}", true, ";", "", false, 0},
		// Just explode modifier, empty name
		{"{*}", true, "", "", true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseTemplateSegment(tt.input)
			assert.Equal(t, tt.input, result.Raw, "Raw")
			assert.Equal(t, tt.isVariable, result.IsVariable, "IsVariable")
			assert.Equal(t, tt.operator, result.Operator, "Operator")
			assert.Equal(t, tt.name, result.Name, "Name")
			assert.Equal(t, tt.explode, result.Explode, "Explode")
			assert.Equal(t, tt.prefix, result.Prefix, "Prefix")
		})
	}
}

func TestNormalizeTemplateParam(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"{id}", "%"},
		{"{;id}", "%op_semi"},
		{"{+path}", "%op_plus"},
		{"{#frag}", "%op_hash"},
		{"{.ext}", "%op_dot"},
		{"{/seg}", "%op_slash"},
		{"{?q}", "%op_query"},
		{"{&more}", "%op_amp"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tv := ParseTemplateSegment(tt.input)
			assert.Equal(t, tt.expected, normalizeTemplateParam(tv))
		})
	}
}

func TestIsTemplateSegment(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"{id}", true},
		{"{;id}", true},
		{"{+path}", true},
		{"{id}.json", true},
		{"{;id}.json", true},
		{"literal", false},
		{"no-braces", false},
		{"{}", false}, // too short, not a variable
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, isTemplateSegment(tt.input))
		})
	}
}
