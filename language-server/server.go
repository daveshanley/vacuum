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
	"path/filepath"
	"strings"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	"github.com/spf13/viper"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	glspserv "github.com/tliron/glsp/server"
)

var serverName = "vacuum"

// DocumentContext contains details about the file being processed by the LSP.
// This allows you to add logic to the RulesetSelector based on the file name or
// content of the document being processed.
type DocumentContext struct {
	Content  []byte
	Filename string
	URI      string
}

// RulesetSelector is used in NewServerWithRulesetSelector to allow you to dynamically
// return what rules should be used for the language server diagnostics based on
// the actual content of the OpenAPI spec being procesed.
type RulesetSelector func(ctx *DocumentContext) *rulesets.RuleSet

type ServerState struct {
	server        *glspserv.Server
	documentStore *DocumentStore
	lintRequest   *utils.LintFileRequest
	rulesetSelector RulesetSelector
}

func NewServer(version string, lintRequest *utils.LintFileRequest) *ServerState {
	handler := protocol.Handler{}
	server := glspserv.NewServer(&handler, serverName, true)

	state := &ServerState{
		server:        server,
		lintRequest:   lintRequest,
		documentStore: newDocumentStore(),
	}
	handler.Initialize = func(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
		if params.Trace != nil {
			protocol.SetTraceValue(*params.Trace)
		}

		serverCapabilities := handler.CreateServerCapabilities()
		serverCapabilities.TextDocumentSync = protocol.TextDocumentSyncKindIncremental
		serverCapabilities.CompletionProvider = &protocol.CompletionOptions{}
		serverCapabilities.CodeActionProvider = &protocol.CodeActionOptions{
			CodeActionKinds: []protocol.CodeActionKind{protocol.CodeActionKindQuickFix},
		}
		serverCapabilities.ExecuteCommandProvider = &protocol.ExecuteCommandOptions{
			Commands: []string{"vacuum.openUrl"},
		}

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
		state.runDiagnostic(doc, context.Notify)
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
		state.runDiagnostic(doc, context.Notify)
		return nil
	}

	handler.TextDocumentDidClose = func(context *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
		state.documentStore.Remove(params.TextDocument.URI)
		return nil
	}

	handler.TextDocumentCompletion = func(context *glsp.Context, params *protocol.CompletionParams) (any, error) {
		return nil, nil
	}

	handler.TextDocumentCodeAction = func(context *glsp.Context, params *protocol.CodeActionParams) (any, error) {
		var actions []protocol.CodeAction
		
		for _, diagnostic := range params.Context.Diagnostics {
			if diagnostic.CodeDescription != nil && diagnostic.CodeDescription.HRef != "" {
				quickFixKind := protocol.CodeActionKindQuickFix
				actions = append(actions, protocol.CodeAction{
					Title: "View documentation",
					Kind:  &quickFixKind,
					Command: &protocol.Command{
						Title:     "Open documentation",
						Command:   "vacuum.openUrl",
						Arguments: []interface{}{diagnostic.CodeDescription.HRef},
					},
				})
			}
		}
		
		return actions, nil
	}

	handler.WorkspaceExecuteCommand = func(context *glsp.Context, params *protocol.ExecuteCommandParams) (any, error) {
		if params.Command == "vacuum.openUrl" && len(params.Arguments) > 0 {
			if url, ok := params.Arguments[0].(string); ok {
				utils.OpenURL(url)
			}
		}
		return nil, nil
	}
	return state
}

// NewServerWithRulesetSelector creates a new instance of the language server with
// a custom RulsetSelector function to allow you to dynamically select the ruleset
// used based on the content of the spec being processed.
// 
// This allows you to determine specifically what rules should be applied per spec, e.g.:
// 
// Have different teams which require different rules?
// Check the value of info.contact.name in the spec and return the releant rules for that team.
// 
// Want to enable OWASP rules for only a specific server?
// Check the value of servers[0].url and return the rules including the OWASP ruleset
// for your specific super secure sever url.
func NewServerWithRulesetSelector(version string, lintRequest *utils.LintFileRequest, selector RulesetSelector) *ServerState {
	state := NewServer(version, lintRequest)
	state.rulesetSelector = selector
	return state
}

func (s *ServerState) Run() error {
	s.initializeConfig()

	viper.OnConfigChange(s.onConfigChange)
	viper.WatchConfig()
	return s.server.RunStdio()
}

func (s *ServerState) runDiagnostic(doc *Document, notify glsp.NotifyFunc) {

	go func() {
		specFileName := strings.TrimPrefix(doc.URI, "file://")

		baseForDoc := s.lintRequest.BaseFlag
		if baseForDoc == "" {
			baseForDoc = filepath.Dir(specFileName)
		}

		deepGraph := false
		if len(s.lintRequest.IgnoredResults) > 0 {
			deepGraph = true
		}

		selectedRuleSet := s.lintRequest.SelectedRS
		if s.rulesetSelector != nil {
			docCtx := &DocumentContext{
				Content:  []byte(doc.Content),
				Filename: specFileName,
				URI:      doc.URI,
			}
			selectedRuleSet = s.rulesetSelector(docCtx)
		}

		result := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
			RuleSet:                         selectedRuleSet,
			Spec:                            []byte(doc.Content),
			SpecFileName:                    specFileName,
			Timeout:                         time.Duration(s.lintRequest.TimeoutFlag) * time.Second,
			NodeLookupTimeout:               time.Duration(s.lintRequest.LookupTimeoutFlag) * time.Millisecond,
			CustomFunctions:                 s.lintRequest.Functions,
			IgnoreCircularArrayRef:          s.lintRequest.IgnoreArrayCircleRef,
			IgnoreCircularPolymorphicRef:    s.lintRequest.IgnorePolymorphCircleRef,
			AllowLookup:                     s.lintRequest.Remote,
			Base:                            baseForDoc,
			SkipDocumentCheck:               s.lintRequest.SkipCheckFlag,
			Logger:                          s.lintRequest.Logger,
			BuildDeepGraph:                  deepGraph,
			ExtractReferencesFromExtensions: s.lintRequest.ExtensionRefs,
			HTTPClientConfig:                s.lintRequest.HTTPClientConfig,
		})

		filteredResults := utils.FilterIgnoredResults(result.Results, s.lintRequest.IgnoredResults)
		result.Results = filteredResults
		diagnostics := ConvertResultsIntoDiagnostics(result)
		go notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
			URI:         doc.URI,
			Diagnostics: diagnostics,
		})
	}()
}

func ConvertResultsIntoDiagnostics(result *motor.RuleSetExecutionResult) []protocol.Diagnostic {
	diagnostics := []protocol.Diagnostic{}

	for _, vacuumResult := range result.Results {
		diagnostics = append(diagnostics, ConvertResultIntoDiagnostic(&vacuumResult))

	}
	return diagnostics
}

func ConvertResultIntoDiagnostic(vacuumResult *model.RuleFunctionResult) protocol.Diagnostic {
	severity := GetDiagnosticSeverityFromRule(vacuumResult.Rule)

	diagnosticErrorHref := fmt.Sprintf("%s/rules/unknown", model.WebsiteUrl)
	if vacuumResult.Rule.DocumentationURL != "" {
		diagnosticErrorHref = vacuumResult.Rule.DocumentationURL
	} else if vacuumResult.Rule.RuleCategory != nil {
		diagnosticErrorHref = fmt.Sprintf("%s/rules/%s/%s", model.WebsiteUrl,
			strings.ToLower(vacuumResult.Rule.RuleCategory.Id),
			strings.ReplaceAll(strings.ToLower(vacuumResult.Rule.Id), "$", ""))
	}
	startLine := 1
	startChar := 1
	endLine := 1
	endChar := 1

	if vacuumResult.StartNode != nil && vacuumResult.StartNode.Line > 0 {
		startLine = vacuumResult.StartNode.Line - 1
		startChar = vacuumResult.StartNode.Column - 1
	}
	if vacuumResult.EndNode != nil && vacuumResult.EndNode.Line > 0 {
		endLine = vacuumResult.EndNode.Line - 1
		endChar = vacuumResult.EndNode.Column - 1
	}

	// Build comprehensive message with rule details
	message := vacuumResult.Message
	if vacuumResult.Rule.Description != "" {
		message += "\n\nDescription: " + vacuumResult.Rule.Description
	}
	if vacuumResult.Rule.HowToFix != "" {
		message += "\n\nHow to fix: " + vacuumResult.Rule.HowToFix + "\n\nRule ID: " + vacuumResult.Rule.Id + "\n"
	}

	return protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{Line: protocol.UInteger(startLine),
				Character: protocol.UInteger(startChar)},
			End: protocol.Position{Line: protocol.UInteger(endLine),
				Character: protocol.UInteger(endChar)},
		},
		Severity:        &severity,
		Source:          &serverName,
		Code:            &protocol.IntegerOrString{Value: vacuumResult.Rule.Id},
		CodeDescription: &protocol.CodeDescription{HRef: diagnosticErrorHref},
		Message:         message,
	}
}

func GetDiagnosticSeverityFromRule(rule *model.Rule) protocol.DiagnosticSeverity {
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
