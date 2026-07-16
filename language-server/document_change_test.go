// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package languageserver

import (
	"testing"

	"github.com/daveshanley/vacuum/language-server/protocol"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestApplyDocumentChangesRejectsInvalidRangesWithoutPoisoningDocument(t *testing.T) {
	document := &Document{Content: "hello\nworld"}
	tests := []struct {
		name       string
		rangeValue protocol.Range
	}{
		{
			name: "reversed",
			rangeValue: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 2},
				End:   protocol.Position{Line: 0, Character: 1},
			},
		},
		{
			name: "invalid line",
			rangeValue: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 9, Character: 0},
			},
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			err := applyDocumentChanges(document, []any{
				protocol.TextDocumentContentChangeEvent{
					Range: &testCase.rangeValue,
					Text:  "broken",
				},
			})
			require.Error(t, err)
			assert.Equal(t, "hello\nworld", document.Content)
		})
	}

	validRange := protocol.Range{
		Start: protocol.Position{Line: 1, Character: 0},
		End:   protocol.Position{Line: 1, Character: 5},
	}
	require.NoError(t, applyDocumentChanges(document, []any{
		protocol.TextDocumentContentChangeEvent{Range: &validRange, Text: "fixed"},
	}))
	assert.Equal(t, "hello\nfixed", document.Content)
}

func TestApplyDocumentChangesIsTransactional(t *testing.T) {
	document := &Document{Content: "hello\nworld"}
	first := protocol.Range{
		Start: protocol.Position{Line: 0, Character: 0},
		End:   protocol.Position{Line: 0, Character: 5},
	}
	invalid := protocol.Range{
		Start: protocol.Position{Line: 7, Character: 0},
		End:   protocol.Position{Line: 7, Character: 1},
	}
	err := applyDocumentChanges(document, []any{
		protocol.TextDocumentContentChangeEvent{Range: &first, Text: "changed"},
		protocol.TextDocumentContentChangeEvent{Range: &invalid, Text: "broken"},
	})
	require.Error(t, err)
	assert.Equal(t, "hello\nworld", document.Content)
}
