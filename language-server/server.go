// Copyright 2024 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT
// https://pb33f.io
// https://quobix.com/vacuum
//
// This code was originally written by KDanisme (https://github.com/KDanisme) and was submitted as a PR
// to the vacuum project. It then was modified by Dave Shanley to fit the needs of the vacuum project.
// The original code can be found here:
// https://github.com/KDanisme/vacuum/tree/language-server
//
// I (Dave Shanley) do not know what happened to KDasnime, or why the PR was
// closed, but I am grateful for the contribution.
//
// This feature is why I built vacuum. This is the reason for its existence.

package languageserver

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/utils"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	glspserv "github.com/tliron/glsp/server"
	"strings"
	"time"
)

var serverName = "vacuum"

type ServerState struct {
	server        *glspserv.Server
	documentStore *DocumentStore
	lintRequest   *utils.LintFileRequest
}

func NewServer(version string, lintRequest *utils.LintFileRequest) *ServerState {
	handler := protocol.Handler{}
	server := glspserv.NewServer(&handler, serverName, true)

	state := &ServerState{
		server:        server,
		lintRequest:   lintRequest,
		documentStore: newDocumentStore(),
	}
	handler.Initialize = func(context *glsp.Context, params *protocol.InitializeParams) (interface{}, error) {
		if params.Trace != nil {
			protocol.SetTraceValue(*params.Trace)
		}

		serverCapabilities := handler.CreateServerCapabilities()
		serverCapabilities.TextDocumentSync = protocol.TextDocumentSyncKindIncremental
		serverCapabilities.CompletionProvider = &protocol.CompletionOptions{}

		return protocol.InitializeResult{
			Capabilities: serverCapabilities,
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
		doc := state.documentStore.Add(params.TextDocument.URI, params.TextDocument.Text)
		state.runDiagnostic(doc, context.Notify, false)
		return nil
	}
	handler.TextDocumentDidChange = func(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
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
		state.documentStore.Remove(params.TextDocument.URI)

		return nil
	}

	handler.TextDocumentCompletion = func(context *glsp.Context, params *protocol.CompletionParams) (any, error) {
		return nil, nil
	}
	return state
}

func (s *ServerState) Run() error {
	return s.server.RunStdio()
}

func (s *ServerState) runDiagnostic(doc *Document, notify glsp.NotifyFunc, delay bool) {

	go func() {
		var diagnostics []protocol.Diagnostic

		result := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
			RuleSet:                      s.lintRequest.SelectedRS,
			Timeout:                      time.Duration(s.lintRequest.TimeoutFlag) * time.Second,
			CustomFunctions:              s.lintRequest.Functions,
			IgnoreCircularArrayRef:       s.lintRequest.IgnoreArrayCircleRef,
			IgnoreCircularPolymorphicRef: s.lintRequest.IgnorePolymorphCircleRef,
			AllowLookup:                  s.lintRequest.Remote,
			Base:                         s.lintRequest.BaseFlag,
			Spec:                         []byte(doc.Content),
			SkipDocumentCheck:            s.lintRequest.SkipCheckFlag,
			Logger:                       s.lintRequest.Logger,
		})

		for _, vacuumResult := range result.Results {
			severity := getDiagnosticSeverityFromRule(vacuumResult.Rule)
			diagnosticErrorHref := fmt.Sprintf("%s/rules/%s/%s", model.WebsiteUrl,
				vacuumResult.Rule.RuleCategory.Id, strings.ReplaceAll(vacuumResult.Rule.Id, "$", ""))
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
	return protocol.DiagnosticSeverityError
}
