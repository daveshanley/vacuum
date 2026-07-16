// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package protocol

// DiagnosticSeverity is the client-facing severity of a diagnostic.
type DiagnosticSeverity Integer

const (
	// DiagnosticSeverityError identifies an error.
	DiagnosticSeverityError = DiagnosticSeverity(1)
	// DiagnosticSeverityWarning identifies a warning.
	DiagnosticSeverityWarning = DiagnosticSeverity(2)
	// DiagnosticSeverityInformation identifies informational feedback.
	DiagnosticSeverityInformation = DiagnosticSeverity(3)
	// DiagnosticSeverityHint identifies a hint.
	DiagnosticSeverityHint = DiagnosticSeverity(4)
)

// DiagnosticTag identifies additional diagnostic semantics.
type DiagnosticTag Integer

const (
	// DiagnosticTagUnnecessary marks unnecessary code.
	DiagnosticTagUnnecessary = DiagnosticTag(1)
	// DiagnosticTagDeprecated marks deprecated code.
	DiagnosticTagDeprecated = DiagnosticTag(2)
)

// Location identifies a range within a document.
type Location struct {
	URI   DocumentUri `json:"uri"`
	Range Range       `json:"range"`
}

// DiagnosticRelatedInformation adds a related location and message.
type DiagnosticRelatedInformation struct {
	Location Location `json:"location"`
	Message  string   `json:"message"`
}

// CodeDescription links a diagnostic to further documentation.
type CodeDescription struct {
	HRef URI `json:"href"`
}

// Diagnostic is a Vacuum result encoded for a language client.
type Diagnostic struct {
	Range              Range                          `json:"range"`
	Severity           *DiagnosticSeverity            `json:"severity,omitempty"`
	Code               *IntegerOrString               `json:"code,omitempty"`
	CodeDescription    *CodeDescription               `json:"codeDescription,omitempty"`
	Source             *string                        `json:"source,omitempty"`
	Message            string                         `json:"message"`
	Tags               []DiagnosticTag                `json:"tags,omitempty"`
	RelatedInformation []DiagnosticRelatedInformation `json:"relatedInformation,omitempty"`
	Data               any                            `json:"data,omitempty"`
}

// ServerTextDocumentPublishDiagnostics is the diagnostics notification method.
const ServerTextDocumentPublishDiagnostics = Method("textDocument/publishDiagnostics")

// PublishDiagnosticsParams publishes the current diagnostics for a document.
type PublishDiagnosticsParams struct {
	URI         DocumentUri  `json:"uri"`
	Version     *UInteger    `json:"version,omitempty"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}
