package languageserver

import (
	"context"
	"strings"

	"github.com/daveshanley/vacuum/language-server/protocol"
	"github.com/daveshanley/vacuum/motor"
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

// repairActionCache retains at most one verified edit (or no-fix result) per
// open document. No parser graph is retained after verification.
type repairActionCache struct {
	content    string
	version    protocol.Integer
	revision   uint64
	generation uint64
	action     *protocol.CodeAction
}

func (s *ServerState) codeActions(ctx context.Context, params *protocol.CodeActionParams) []protocol.CodeAction {
	var actions []protocol.CodeAction
	if !quickFixRequested(params.Context.Only) {
		return actions
	}
	vacuumDiagnostic := false
	for _, diagnostic := range params.Context.Diagnostics {
		if diagnostic.Source != nil && *diagnostic.Source == serverName && diagnostic.Code != nil {
			if code, ok := diagnostic.Code.Value.(string); ok && code != "" {
				vacuumDiagnostic = true
			}
		}
		if diagnostic.CodeDescription != nil && diagnostic.CodeDescription.HRef != "" {
			kind := protocol.CodeActionKindQuickFix
			actions = append(actions, protocol.CodeAction{Title: "View documentation", Kind: &kind, Command: &protocol.Command{Title: "Open documentation", Command: "vacuum.openUrl", Arguments: []interface{}{diagnostic.CodeDescription.HRef}}})
		}
	}
	if vacuumDiagnostic {
		if action := s.repairCodeAction(ctx, params.TextDocument.URI); action != nil {
			actions = append(actions, *action)
		}
	}
	return actions
}

func quickFixRequested(only []protocol.CodeActionKind) bool {
	if len(only) == 0 {
		return true
	}
	for _, kind := range only {
		if kind == "" || kind == protocol.CodeActionKindQuickFix {
			return true
		}
	}
	return false
}

// repairCodeAction returns a verified buffer edit. The client checks the version
// before applying it; the server never writes files for a code action. Repeated
// requests for the same buffer and configuration reuse the verified result.
func (s *ServerState) repairCodeAction(ctx context.Context, uri protocol.DocumentUri) *protocol.CodeAction {
	if s.lintRequest.MainSpecPath != "" {
		return nil
	}
	doc, ok := s.documentStore.Get(uri)
	if !ok {
		return nil
	}
	select {
	case doc.repairGate <- struct{}{}:
		defer func() { <-doc.repairGate }()
	case <-ctx.Done():
		return nil
	}
	if ctx.Err() != nil {
		return nil
	}
	doc.mu.RLock()
	content, version, revision := doc.Content, doc.Version, doc.revision
	doc.mu.RUnlock()
	s.configMu.RLock()
	generation := s.configGeneration
	s.configMu.RUnlock()
	if cache := doc.repairCache; cache != nil && cache.content == content && cache.version == version && cache.revision == revision && cache.generation == generation {
		if current, ok := s.documentStore.Get(uri); ok && current == doc {
			return copyRepairAction(cache.action)
		}
		return nil
	}
	doc.repairCache = nil
	options := motor.ExecutionOptions{}
	if s.executionOptions != nil {
		options = *s.executionOptions
	}
	options.Context = ctx
	result, config, err := s.lintDocumentMode(content, uri, fileURIToPath(uri), nil, &options, true)
	if result != nil {
		defer result.ReleaseOwnedResources()
	}
	if ctx.Err() != nil {
		return nil
	}
	current, ok := s.documentStore.Get(uri)
	if !ok || current != doc {
		return nil
	}
	doc.mu.RLock()
	stale := doc.Version != version || doc.Content != content || doc.revision != revision
	doc.mu.RUnlock()
	s.configMu.RLock()
	stale = stale || generation != s.configGeneration
	if config != nil {
		stale = stale || config.generation != s.configGeneration
	}
	s.configMu.RUnlock()
	if stale {
		return nil
	}
	cache := &repairActionCache{content: content, version: version, generation: generation, revision: revision}
	doc.repairCache = cache
	if err != nil || result == nil || len(result.Errors) > 0 || len(result.FixedResults) == 0 || len(result.ModifiedSpec) == 0 {
		return nil
	}
	end := protocol.Position{Line: protocol.UInteger(strings.Count(content, "\n"))}.EndOfLineIn(content)
	kind := protocol.CodeActionKindQuickFix
	cache.action = &protocol.CodeAction{Title: "Apply verified built-in repairs", Kind: &kind, Edit: repairWorkspaceEdit{DocumentChanges: []repairDocumentEdit{{TextDocument: protocol.VersionedTextDocumentIdentifier{TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri}, Version: version}, Edits: []repairTextEdit{{Range: protocol.Range{End: end}, NewText: string(result.ModifiedSpec)}}}}}}
	return copyRepairAction(cache.action)
}

// Copy the small edit envelope so request consumers cannot mutate cached slices.
func copyRepairAction(action *protocol.CodeAction) *protocol.CodeAction {
	if action == nil {
		return nil
	}
	out := *action
	kind := *action.Kind
	out.Kind = &kind
	edit := action.Edit.(repairWorkspaceEdit)
	edit.DocumentChanges = append([]repairDocumentEdit(nil), edit.DocumentChanges...)
	for i := range edit.DocumentChanges {
		edit.DocumentChanges[i].Edits = append([]repairTextEdit(nil), edit.DocumentChanges[i].Edits...)
	}
	out.Edit = edit
	return &out
}
