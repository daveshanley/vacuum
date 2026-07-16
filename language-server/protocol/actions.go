// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package protocol

// Command describes a command that a language client can execute.
type Command struct {
	Title     string `json:"title"`
	Command   string `json:"command"`
	Arguments []any  `json:"arguments,omitempty"`
}

// CodeActionKind identifies a class of code action.
type CodeActionKind = string

const (
	// CodeActionKindEmpty represents an unspecified code-action kind.
	CodeActionKindEmpty = CodeActionKind("")
	// CodeActionKindQuickFix represents a quick-fix code action.
	CodeActionKindQuickFix = CodeActionKind("quickfix")
)

// CodeActionContext contains diagnostics and requested action kinds.
type CodeActionContext struct {
	Diagnostics []Diagnostic     `json:"diagnostics"`
	Only        []CodeActionKind `json:"only,omitempty"`
}

// CodeActionParams identifies the document range requesting actions.
type CodeActionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
	Context      CodeActionContext      `json:"context"`
}

// CodeAction describes an edit or command offered for a document range.
type CodeAction struct {
	Title       string              `json:"title"`
	Kind        *CodeActionKind     `json:"kind,omitempty"`
	Diagnostics []Diagnostic        `json:"diagnostics,omitempty"`
	IsPreferred *bool               `json:"isPreferred,omitempty"`
	Disabled    *CodeActionDisabled `json:"disabled,omitempty"`
	Edit        any                 `json:"edit,omitempty"`
	Command     *Command            `json:"command,omitempty"`
	Data        any                 `json:"data,omitempty"`
}

// CodeActionDisabled explains why a code action cannot be applied.
type CodeActionDisabled struct {
	Reason string `json:"reason"`
}
