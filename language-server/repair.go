package languageserver

import (
	"strings"

	"github.com/daveshanley/vacuum/language-server/protocol"
)

type repairWorkspaceEdit struct {
	DocumentChanges []repairDocumentEdit `json:"documentChanges"`
}
type repairDocumentEdit struct {
	TextDocument protocol.VersionedTextDocumentIdentifier `json:"textDocument"`
	Edits        []repairTextEdit                         `json:"edits"`
}
type repairTextEdit struct {
	Range   protocol.Range `json:"range"`
	NewText string         `json:"newText"`
}

// repairCodeAction returns a verified buffer edit. The client checks the version
// before applying it; the server never writes files for a code action.
func (s *ServerState) repairCodeAction(uri protocol.DocumentUri) *protocol.CodeAction {
	if s.lintRequest.MainSpecPath != "" {
		return nil
	}
	doc, ok := s.documentStore.Get(uri)
	if !ok {
		return nil
	}
	doc.mu.RLock()
	content, version := doc.Content, doc.Version
	doc.mu.RUnlock()
	result, config, err := s.lintDocumentMode(content, uri, fileURIToPath(uri), nil, s.executionOptions, true)
	if err != nil {
		return nil
	}
	defer result.ReleaseOwnedResources()
	if len(result.Errors) > 0 || len(result.FixedResults) == 0 || len(result.ModifiedSpec) == 0 {
		return nil
	}
	current, ok := s.documentStore.Get(uri)
	if !ok || current != doc {
		return nil
	}
	doc.mu.RLock()
	stale := doc.Version != version || doc.Content != content
	doc.mu.RUnlock()
	s.configMu.RLock()
	stale = stale || config.generation != s.configGeneration
	s.configMu.RUnlock()
	if stale {
		return nil
	}
	end := protocol.Position{Line: protocol.UInteger(strings.Count(content, "\n"))}.EndOfLineIn(content)
	kind := protocol.CodeActionKindQuickFix
	return &protocol.CodeAction{
		Title: "Apply verified built-in repairs", Kind: &kind,
		Edit: repairWorkspaceEdit{DocumentChanges: []repairDocumentEdit{{
			TextDocument: protocol.VersionedTextDocumentIdentifier{TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri}, Version: version},
			Edits:        []repairTextEdit{{Range: protocol.Range{End: end}, NewText: string(result.ModifiedSpec)}},
		}}},
	}
}
