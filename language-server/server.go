// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package languageserver

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	glspserv "github.com/tliron/glsp/server"
	"log/slog"
	"time"
)

var serverName = "vacuum"

type ServerState struct {
	server        *glspserv.Server
	documentStore *DocumentStore
}

func NewServer(version string, logger *slog.Logger) *ServerState {
	handler := protocol.Handler{}
	server := glspserv.NewServer(&handler, serverName, false)
	state := &ServerState{
		server:        server,
		documentStore: newDocumentStore(),
	}
	handler.Initialize = func(context *glsp.Context, params *protocol.InitializeParams) (interface{}, error) {
		logger.Info("Initializing vacuum language server")
		if params.Trace != nil {
			protocol.SetTraceValue(*params.Trace)
		}
		return protocol.InitializeResult{
			Capabilities: handler.CreateServerCapabilities(),
			ServerInfo: &protocol.InitializeResultServerInfo{
				Name:    serverName,
				Version: &version,
			},
		}, nil
	}
	handler.SetTrace = func(context *glsp.Context, params *protocol.SetTraceParams) error {
		protocol.SetTraceValue(params.Value)
		return nil
	}
	handler.TextDocumentDidOpen = func(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
		logger.Info("document opened")
		doc := state.documentStore.Add(params.TextDocument.URI, params.TextDocument.Text)
		state.runDiagnostic(doc, context.Notify, false)
		return nil
	}
	handler.TextDocumentDidChange = func(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
		logger.Info("document changed")
		doc, ok := state.documentStore.Get(params.TextDocument.URI)
		if !ok {
			return nil
		}
		for _, change := range params.ContentChanges {
			switch c := change.(type) {
			case protocol.TextDocumentContentChangeEvent:
				startIndex, endIndex := c.Range.IndexesIn(doc.Content)
				doc.Content = doc.Content[:startIndex] + c.Text + doc.Content[endIndex:]
			case protocol.TextDocumentContentChangeEventWhole:
				doc.Content = c.Text
			}
		}
		state.runDiagnostic(doc, context.Notify, true)
		return nil
	}

	handler.TextDocumentDidClose = func(context *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
		logger.Info("document closed")
		state.documentStore.Remove(params.TextDocument.URI)

		return nil
	}
	return state
}

func (s *ServerState) Run() error {
	return s.server.RunStdio()
}

func (s *ServerState) runDiagnostic(doc *Document, notify glsp.NotifyFunc, delay bool) {
	if doc.RunningDiagnostic {
		return
	}

	doc.RunningDiagnostic = true
	go func() {
		if delay {
			time.Sleep(1 * time.Second)
		}
		doc.RunningDiagnostic = false

		var diagnostics []protocol.Diagnostic
		defaultRuleSets := rulesets.BuildDefaultRuleSets()

		selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()
		result := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
			RuleSet:           selectedRS,
			Spec:              []byte(doc.Content),
			CustomFunctions:   nil,
			Base:              "",
			SkipDocumentCheck: false,
		})
		for _, vacuumResult := range result.Results {
			severity := getDiagnosticSeverityFromRule(vacuumResult.Rule)
			diagnosticErrorHref := fmt.Sprintf("%s/rules/%s/%s,", model.WebsiteUrl,
				vacuumResult.Rule.RuleCategory.Id, vacuumResult.Rule.Id)
			diagnostics = append(diagnostics, protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: protocol.UInteger(vacuumResult.StartNode.Line - 1),
						Character: protocol.UInteger(vacuumResult.StartNode.Column - 1)},
					End: protocol.Position{Line: protocol.UInteger(vacuumResult.EndNode.Line - 1),
						Character: protocol.UInteger(vacuumResult.EndNode.Column + len(vacuumResult.EndNode.Value) - 1)},
				},
				Severity:        &severity,
				Source:          &serverName,
				Code:            &protocol.IntegerOrString{Value: vacuumResult.Rule.Id},
				CodeDescription: &protocol.CodeDescription{HRef: diagnosticErrorHref},
				Message:         vacuumResult.Message,
			})
		}
		if len(diagnostics) > 0 {
			go notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
				URI:         doc.URI,
				Diagnostics: diagnostics,
			})
		}
	}()
}

func getDiagnosticSeverityFromRule(rule *model.Rule) protocol.DiagnosticSeverity {
	switch rule.Severity {
	case model.SeverityError:
		return protocol.DiagnosticSeverityError
	case model.SeverityWarn:
		return protocol.DiagnosticSeverityWarning
	case model.SeverityInfo:
		return protocol.DiagnosticSeverityInformation
	}
	return -1
}
