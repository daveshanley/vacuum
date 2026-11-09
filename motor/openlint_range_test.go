// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package motor

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/model/reports"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

func TestOpenLintUTF16PositionsAcrossLineEndings(t *testing.T) {
	index := newOpenLintRangeIndex([]byte("a😀b\r\n𐐀c\rd😀\n"), nil)
	expected := []reports.RangeItem{
		{Line: 0, Char: 0}, {Line: 0, Char: 1}, {Line: 0, Char: 3},
		{Line: 0, Char: 4}, {Line: 0, Char: 5},
		{Line: 1, Char: 0}, {Line: 1, Char: 2}, {Line: 1, Char: 3},
		{Line: 2, Char: 0}, {Line: 2, Char: 1}, {Line: 2, Char: 3},
		{Line: 3, Char: 0},
	}
	for offset, position := range expected {
		require.Equal(t, position, index.position(offset), "offset %d", offset)
	}
	require.Equal(t, expected[0], index.position(-1))
	require.Equal(t, expected[len(expected)-1], index.position(100))
	require.Equal(t, reports.RangeItem{}, newOpenLintRangeIndex(nil, nil).position(0))
}

func TestOpenLintYAMLSourceRanges(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		start, end   reports.RangeItem
	}{
		{"literal", "x: |-\n  hello\n  world\ny: next\n", reports.RangeItem{Line: 0, Char: 3}, reports.RangeItem{Line: 3, Char: 0}},
		{"folded", "x: >-\n  hello\n  world\ny: next\n", reports.RangeItem{Line: 0, Char: 3}, reports.RangeItem{Line: 3, Char: 0}},
		{"explicit-indent", "x: |2- # 9 is a comment\n  hello\n  world\ny: next\n", reports.RangeItem{Line: 0, Char: 3}, reports.RangeItem{Line: 3, Char: 0}},
		{"block-blank-lines", "x: |+\n  hello\n\n  world\n\ny: next\n", reports.RangeItem{Line: 0, Char: 3}, reports.RangeItem{Line: 5, Char: 0}},
		{"block-no-final-newline", "x: |-\n  hello\n  world", reports.RangeItem{Line: 0, Char: 3}, reports.RangeItem{Line: 2, Char: 7}},
		{"plain-multiline", "x: hello\n  world\ny: next\n", reports.RangeItem{Line: 0, Char: 3}, reports.RangeItem{Line: 1, Char: 7}},
		{"plain-comment", "x: hello # comment\ny: next\n", reports.RangeItem{Line: 0, Char: 3}, reports.RangeItem{Line: 0, Char: 8}},
		{"plain-punctuation", "x: https://example.com/a,b#part\ny: next\n", reports.RangeItem{Line: 0, Char: 3}, reports.RangeItem{Line: 0, Char: 31}},
		{"tag", "x: !!str hello\n", reports.RangeItem{Line: 0, Char: 9}, reports.RangeItem{Line: 0, Char: 14}},
		{"anchor", "x: &label hello\n", reports.RangeItem{Line: 0, Char: 10}, reports.RangeItem{Line: 0, Char: 15}},
		{"tag-and-anchor-quoted", "x: !!str &label \"hello \\\"world\\\"\"\n", reports.RangeItem{Line: 0, Char: 16}, reports.RangeItem{Line: 0, Char: 33}},
		{"single-quoted", "x: !!str 'it''s fine'\n", reports.RangeItem{Line: 0, Char: 9}, reports.RangeItem{Line: 0, Char: 21}},
		{"flow-trailing-comma", "x: [one, two,]\n", reports.RangeItem{Line: 0, Char: 3}, reports.RangeItem{Line: 0, Char: 14}},
		{"flow-comment", "x: [one, two # comment\n]\n", reports.RangeItem{Line: 0, Char: 3}, reports.RangeItem{Line: 1, Char: 1}},
		{"flow-map-comment", "x: {a: one, # comment\n}\n", reports.RangeItem{Line: 0, Char: 3}, reports.RangeItem{Line: 1, Char: 1}},
		{"flow-nested-comment", "x: [{a: one, # inner\n}, # outer\n]\n", reports.RangeItem{Line: 0, Char: 3}, reports.RangeItem{Line: 2, Char: 1}},
		{"anchored-flow", "x: &label [one, two,]\n", reports.RangeItem{Line: 0, Char: 10}, reports.RangeItem{Line: 0, Char: 21}},
		{"crlf", "x: hello\r\n  world\r\ny: next\r\n", reports.RangeItem{Line: 0, Char: 3}, reports.RangeItem{Line: 1, Char: 7}},
		{"cr", "x: hello\r  world\ry: next\r", reports.RangeItem{Line: 0, Char: 3}, reports.RangeItem{Line: 1, Char: 7}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var root yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(tc.source), &root))
			index := newOpenLintRangeIndex([]byte(tc.source), &root)
			node := root.Content[0].Content[1]
			finding := model.RuleFunctionResult{StartNode: node}
			index.setResultRange(&finding)
			require.Equal(t, tc.start, finding.Range.Start)
			require.Equal(t, tc.end, finding.Range.End)
			// Resolved documents have different node pointers at the same source position.
			copied := *node
			finding.StartNode = &copied
			index.setResultRange(&finding)
			require.Equal(t, tc.end, finding.Range.End)
		})
	}
}

func TestOpenLintScalarRangesExcludeFollowingSequenceIndicators(t *testing.T) {
	source := []byte("x:\n  - one\n  - two\n")
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal(source, &root))
	index := newOpenLintRangeIndex(source, &root)
	node := root.Content[0].Content[1].Content[0]
	result := model.RuleFunctionResult{StartNode: node}
	index.setResultRange(&result)
	require.Equal(t, reports.RangeItem{Line: 1, Char: 7}, result.Range.End)
}

func TestOpenLintScalarRangeStopsAtDocumentEnd(t *testing.T) {
	source := []byte("hello\n...\n")
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal(source, &root))
	index := newOpenLintRangeIndex(source, &root)
	result := model.RuleFunctionResult{StartNode: root.Content[0]}
	index.setResultRange(&result)
	require.Equal(t, reports.RangeItem{Line: 0, Char: 5}, result.Range.End)
}

func TestOpenLintEmptyTaggedScalarDoesNotConsumeNextNode(t *testing.T) {
	source := []byte("x: !!str\ny: hello\n")
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal(source, &root))
	index := newOpenLintRangeIndex(source, &root)
	result := model.RuleFunctionResult{StartNode: root.Content[0].Content[1]}
	index.setResultRange(&result)
	require.Equal(t, reports.RangeItem{Line: 0, Char: 8}, result.Range.Start)
	require.Equal(t, result.Range.Start, result.Range.End)
}

func TestOpenLintBlockScalarIndentUsesContainingMapping(t *testing.T) {
	source := []byte("- x: |2-\n    hello\n  # comment\n  y: next\n")
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal(source, &root))
	index := newOpenLintRangeIndex(source, &root)
	result := model.RuleFunctionResult{StartNode: root.Content[0].Content[0].Content[1]}
	index.setResultRange(&result)
	require.Equal(t, reports.RangeItem{Line: 2, Char: 0}, result.Range.End)
}
