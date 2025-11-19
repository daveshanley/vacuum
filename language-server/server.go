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
	"github.com/daveshanley/vacuum/utils"
	"github.com/spf13/viper"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	glspserv "github.com/tliron/glsp/server"
	"go.yaml.in/yaml/v4"
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

	handler.TextDocumentDidSave = func(context *glsp.Context, params *protocol.DidSaveTextDocumentParams) error {
		if doc, ok := state.documentStore.Get(params.TextDocument.URI); ok {
			state.runDiagnostic(doc, context.Notify)
		}
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

			// I don't know why, but accessing the rule ID directly from the code field doesn't
			// seem to work, but reading it from the message does reliably.
			ruleId := ""
			if strings.Contains(diagnostic.Message, "Rule ID: ") {
				parts := strings.Split(diagnostic.Message, "Rule ID: ")
				if len(parts) > 1 {
					ruleId = strings.TrimSpace(strings.Split(parts[1], "\n")[0])
				}
			}

			if ruleId != "" && state.hasAutoFixForRule(ruleId) {
				doc, ok := state.documentStore.Get(params.TextDocument.URI)
				if !ok {
					continue
				}

				fixedText, fixedRange := state.applyAutoFix(doc, ruleId, diagnostic.Range)
				if fixedText == "" {
					continue
				}

				quickFixKind := protocol.CodeActionKindQuickFix
				actions = append(actions, protocol.CodeAction{
					Title: fmt.Sprintf("Auto fix %s", ruleId),
					Kind:  &quickFixKind,
					Edit: &protocol.WorkspaceEdit{
						Changes: map[string][]protocol.TextEdit{
							params.TextDocument.URI: {
								{
									Range:   fixedRange,
									NewText: fixedText,
								},
							},
						},
					},
					Diagnostics: []protocol.Diagnostic{diagnostic},
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

func (s *ServerState) applyAutoFix(
	doc *Document,
	ruleId string,
	diagnosticRange protocol.Range,
) (string, protocol.Range) {
	rule, exists := s.lintRequest.SelectedRS.Rules[ruleId]
	if !exists || rule.AutoFixFunction == "" {
		return "", diagnosticRange
	}

	autoFixFunc, exists := s.lintRequest.AutoFixFunctions[rule.AutoFixFunction]
	if !exists {
		return "", diagnosticRange
	}

	var rootNode yaml.Node
	if err := yaml.Unmarshal([]byte(doc.Content), &rootNode); err != nil {
		return "", diagnosticRange
	}

	// Find the node at the diagnostic location
	targetLine := int(diagnosticRange.Start.Line) + 1
	targetNode := s.findNodeAtLine(&rootNode, targetLine)
	if targetNode == nil {
		return "", diagnosticRange
	}

	// Apply the autofix function
	fixedNode, err := autoFixFunc(targetNode, &rootNode, nil)
	if err != nil || fixedNode == nil {
		return "", diagnosticRange
	}

	// The autofix function modifies the node in place, so use the modified targetNode
	modifiedNode := targetNode

	// Calculate the actual range of the node that was modified
	// For mapping nodes, we need to include the key and colon
	nodeRange := protocol.Range{
		Start: protocol.Position{
			Line:      protocol.UInteger(targetNode.Line - 1),
			Character: protocol.UInteger(targetNode.Column - 1),
		},
		End: protocol.Position{
			Line:      protocol.UInteger(targetNode.Line - 1),
			Character: protocol.UInteger(targetNode.Column - 1 + len(targetNode.Value)),
		},
	}

	// If this is a key in a mapping, we need to replace "key: value" not just "value"
	if targetNode.Kind == yaml.ScalarNode && fixedNode.Kind == yaml.ScalarNode {
		// Find if this node is a key by checking if it's at an even index in parent's content
		parent := s.findParentNode(&rootNode, targetNode)
		if parent != nil && parent.Kind == yaml.MappingNode {
			for i, child := range parent.Content {
				if child == targetNode && i%2 == 0 {
					// This is a key node, return "newkey: value" format
					valueNode := parent.Content[i+1]

					return modifiedNode.Value + ":" + valueNode.Value, nodeRange
				}
			}
		}
	}

	// For complex structures, marshal the entire node
	fixedBytes, err := yaml.Marshal(fixedNode)
	if err != nil {
		return "", diagnosticRange
	}

	return strings.TrimSpace(string(fixedBytes)), nodeRange
}

func (s *ServerState) hasAutoFixForRule(ruleId string) bool {
	if s.lintRequest.SelectedRS == nil {
		return false
	}

	rule, exists := s.lintRequest.SelectedRS.Rules[ruleId]
	if !exists {
		return false
	}

	return rule.AutoFixFunction != "" && s.lintRequest.AutoFixFunctions != nil &&
		s.lintRequest.AutoFixFunctions[rule.AutoFixFunction] != nil
}

func (s *ServerState) findParentNode(root *yaml.Node, target *yaml.Node) *yaml.Node {
	for _, child := range root.Content {
		if child == target {
			return root
		}
		if parent := s.findParentNode(child, target); parent != nil {
			return parent
		}
	}
	return nil
}

func (s *ServerState) findNodeAtLine(node *yaml.Node, targetLine int) *yaml.Node {
	if node.Line == targetLine {
		return node
	}

	for _, child := range node.Content {
		if found := s.findNodeAtLine(child, targetLine); found != nil {
			return found
		}
	}

	return nil
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

		result := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
			RuleSet:      s.lintRequest.SelectedRS,
			Spec:         []byte(doc.Content),
			SpecFileName: specFileName,
			Timeout:      time.Duration(s.lintRequest.TimeoutFlag) * time.Second,
			NodeLookupTimeout: time.Duration(
				s.lintRequest.LookupTimeoutFlag,
			) * time.Millisecond,
			CustomFunctions:                 s.lintRequest.Functions,
			AutoFixFunctions:                s.lintRequest.AutoFixFunctions,
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
			Start: protocol.Position{
				Line:      protocol.UInteger(startLine),
				Character: protocol.UInteger(startChar),
			},
			End: protocol.Position{
				Line:      protocol.UInteger(endLine),
				Character: protocol.UInteger(endChar),
			},
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
