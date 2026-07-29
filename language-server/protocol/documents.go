// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package protocol

import (
	"encoding/json"
	"fmt"
)

const (
	// MethodTextDocumentDidOpen reports an opened document.
	MethodTextDocumentDidOpen = Method("textDocument/didOpen")
	// MethodTextDocumentDidChange reports document content changes.
	MethodTextDocumentDidChange = Method("textDocument/didChange")
	// MethodTextDocumentDidClose reports a closed document.
	MethodTextDocumentDidClose = Method("textDocument/didClose")
	// MethodTextDocumentCompletion requests completions.
	MethodTextDocumentCompletion = Method("textDocument/completion")
	// MethodTextDocumentCodeAction requests code actions.
	MethodTextDocumentCodeAction = Method("textDocument/codeAction")
)

// TextDocumentSyncKind identifies how document content is synchronized.
type TextDocumentSyncKind Integer

const (
	// TextDocumentSyncKindNone disables document synchronization.
	TextDocumentSyncKindNone = TextDocumentSyncKind(0)
	// TextDocumentSyncKindFull sends the complete document for each change.
	TextDocumentSyncKindFull = TextDocumentSyncKind(1)
	// TextDocumentSyncKindIncremental sends ranged document changes.
	TextDocumentSyncKindIncremental = TextDocumentSyncKind(2)
)

// TextDocumentIdentifier identifies a document by URI.
type TextDocumentIdentifier struct {
	URI DocumentUri `json:"uri"`
}

// VersionedTextDocumentIdentifier identifies a versioned document.
type VersionedTextDocumentIdentifier struct {
	TextDocumentIdentifier
	Version Integer `json:"version"`
}

// TextDocumentItem contains an opened document and its content.
type TextDocumentItem struct {
	URI        DocumentUri `json:"uri"`
	LanguageID string      `json:"languageId"`
	Version    Integer     `json:"version"`
	Text       string      `json:"text"`
}

// DidOpenTextDocumentParams contains an opened document.
type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// TextDocumentContentChangeEvent is an incremental ranged content change.
type TextDocumentContentChangeEvent struct {
	Range       *Range    `json:"range"`
	RangeLength *UInteger `json:"rangeLength,omitempty"`
	Text        string    `json:"text"`
}

// TextDocumentContentChangeEventWhole replaces the complete document.
type TextDocumentContentChangeEventWhole struct {
	Text string `json:"text"`
}

// DidChangeTextDocumentParams contains ordered document changes.
type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier `json:"textDocument"`
	ContentChanges []any                           `json:"contentChanges"`
}

func (p *DidChangeTextDocumentParams) UnmarshalJSON(data []byte) error {
	if p == nil {
		return fmt.Errorf("didChange params are nil")
	}
	var wire struct {
		TextDocument   VersionedTextDocumentIdentifier `json:"textDocument"`
		ContentChanges []json.RawMessage               `json:"contentChanges"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	p.TextDocument = wire.TextDocument
	p.ContentChanges = make([]any, 0, len(wire.ContentChanges))
	for _, raw := range wire.ContentChanges {
		var change TextDocumentContentChangeEvent
		if err := json.Unmarshal(raw, &change); err != nil {
			return err
		}
		if change.Range == nil {
			p.ContentChanges = append(p.ContentChanges, TextDocumentContentChangeEventWhole{Text: change.Text})
			continue
		}
		p.ContentChanges = append(p.ContentChanges, change)
	}
	return nil
}

// DidCloseTextDocumentParams identifies a closed document.
type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// TextDocumentPositionParams identifies a position within a document.
type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// CompletionParams contains a completion request position.
type CompletionParams struct {
	TextDocumentPositionParams
}
