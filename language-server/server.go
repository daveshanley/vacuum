// Copyright 2024 Princess Beef Heavy Industries / Dave Shanley
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
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/plugin"
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
	server          *glspserv.Server
	documentStore   *DocumentStore
	lintRequest     *utils.LintFileRequest
	rulesetSelector RulesetSelector

	// Configuration layers (in order of increasing priority)
	baseConfig    *LSPConfig // From command-line flags (immutable after init)
	fileConfig    *LSPConfig // From vacuum.conf.yaml (updated by file watcher)
	initConfig    *LSPConfig // From InitializationOptions (set once at init)
	runtimeConfig *LSPConfig // From didChangeConfiguration (updated at runtime)

	// Effective configuration (computed from all layers)
	effectiveConfig *LSPConfig

	// Synchronization for config updates
	configMu sync.RWMutex

	// Logger for config-related messages
	logger *slog.Logger

	// Cached resource paths to detect changes and avoid reloading
	loadedRulesetPath   string
	loadedFunctionsPath string
	loadedHardMode      bool

	// Notify function for triggering re-lints (used by file watcher)
	notifyFunc glsp.NotifyFunc
	notifyMu   sync.RWMutex
}

func NewServer(version string, lintRequest *utils.LintFileRequest) *ServerState {
	handler := protocol.Handler{}
	server := glspserv.NewServer(&handler, serverName, true)

	// Initialize logger
	logger := lintRequest.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	}

	// Create base config from initial lintRequest values (command-line flags)
	baseConfig := &LSPConfig{
		Base:      lintRequest.BaseFlag,
		Remote:    boolPtr(lintRequest.Remote),
		SkipCheck: boolPtr(lintRequest.SkipCheckFlag),
		Timeout:   intPtr(lintRequest.TimeoutFlag),
	}
	if lintRequest.IgnoreArrayCircleRef {
		baseConfig.IgnoreArrayCircleRef = boolPtr(true)
	}
	if lintRequest.IgnorePolymorphCircleRef {
		baseConfig.IgnorePolymorphCircleRef = boolPtr(true)
	}
	if lintRequest.ExtensionRefs {
		baseConfig.ExtensionRefs = boolPtr(true)
	}

	state := &ServerState{
		server:        server,
		lintRequest:   lintRequest,
		documentStore: newDocumentStore(),
		logger:        logger,
		baseConfig:    baseConfig,
	}
	handler.Initialize = func(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
		if params.Trace != nil {
			protocol.SetTraceValue(*params.Trace)
		}

		// Extract and apply InitializationOptions if provided
		if params.InitializationOptions != nil {
			initConfig, err := ParseLSPConfig(params.InitializationOptions)
			if err != nil {
				state.logger.Warn("failed to parse InitializationOptions", "error", err)
			} else if initConfig != nil {
				state.initConfig = initConfig
				if err := state.applyEffectiveConfig(); err != nil {
					state.logger.Warn("failed to apply InitializationOptions", "error", err)
				}
			}
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
		// Store notify function for file watcher re-linting
		state.setNotifyFunc(context.Notify)

		doc := state.documentStore.Add(params.TextDocument.URI, params.TextDocument.Text)
		state.runDiagnostic(doc, context.Notify)
		return nil
	}
	handler.TextDocumentDidChange = func(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
		doc, ok := state.documentStore.Get(params.TextDocument.URI)
		if !ok {
			return nil
		}

		// Hold write lock while modifying Content
		doc.mu.Lock()
		for _, change := range params.ContentChanges {
			switch c := change.(type) {
			case protocol.TextDocumentContentChangeEvent:
				startIndex, endIndex := c.Range.IndexesIn(doc.Content)
				doc.Content = doc.Content[:startIndex] + c.Text + doc.Content[endIndex:]
			case protocol.TextDocumentContentChangeEventWhole:
				doc.Content = c.Text
			}
		}
		doc.mu.Unlock()

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

	handler.WorkspaceDidChangeConfiguration = func(context *glsp.Context, params *protocol.DidChangeConfigurationParams) error {
		config, err := ParseLSPConfig(params.Settings)
		if err != nil {
			state.logger.Warn("failed to parse configuration settings", "error", err)
			return nil // Don't fail - log and continue with existing config
		}

		if config != nil {
			state.runtimeConfig = config

			if err := state.applyEffectiveConfig(); err != nil {
				state.logger.Warn("failed to apply configuration", "error", err)
				return nil
			}

			// Trigger re-linting of all open documents
			state.relintAllDocuments(context.Notify)
		}

		return nil
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
// Check the value of info.contact.name in the spec and return the relevant rules for that team.
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
	// Copy document data while holding read lock to avoid data race
	doc.mu.RLock()
	content := doc.Content
	uri := doc.URI
	doc.mu.RUnlock()

	specFileName := strings.TrimPrefix(uri, "file://")

	// Copy config data while holding config lock to avoid data race
	s.configMu.RLock()
	baseForDoc := s.lintRequest.BaseFlag
	if baseForDoc == "" {
		baseForDoc = filepath.Dir(specFileName)
	}

	deepGraph := len(s.lintRequest.IgnoredResults) > 0
	ignoredResults := s.lintRequest.IgnoredResults

	// Build the rule execution config with copied values
	ruleExec := &motor.RuleSetExecution{
		RuleSet:                         s.lintRequest.SelectedRS,
		Spec:                            []byte(content),
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
	}
	s.configMu.RUnlock()

	// Apply ruleset selector if configured (uses copied content)
	if s.rulesetSelector != nil {
		docCtx := &DocumentContext{
			Content:  []byte(content),
			Filename: specFileName,
			URI:      uri,
		}
		ruleExec.RuleSet = s.rulesetSelector(docCtx)
	}

	go func() {
		result := motor.ApplyRulesToRuleSet(ruleExec)

		filteredResults := utils.FilterIgnoredResults(result.Results, ignoredResults)
		result.Results = filteredResults
		diagnostics := ConvertResultsIntoDiagnostics(result)

		notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
			URI:         uri,
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

// applyEffectiveConfig computes the effective configuration from all layers
// and updates the lintRequest accordingly. Thread-safe.
func (s *ServerState) applyEffectiveConfig() error {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	// Start with defaults
	effective := &LSPConfig{
		Remote:  boolPtr(true), // Default: remote lookups enabled
		Timeout: intPtr(5),     // Default: 5 second timeout
	}

	// Apply layers in order of increasing priority
	MergeConfig(effective, s.baseConfig)    // Command-line flags
	MergeConfig(effective, s.fileConfig)    // vacuum.conf.yaml
	MergeConfig(effective, s.initConfig)    // InitializationOptions
	MergeConfig(effective, s.runtimeConfig) // didChangeConfiguration

	s.effectiveConfig = effective

	// Apply to lintRequest
	return s.updateLintRequestFromEffectiveConfig()
}

// updateLintRequestFromEffectiveConfig applies the effective config to the lintRequest.
// Must be called with configMu held.
func (s *ServerState) updateLintRequestFromEffectiveConfig() error {
	cfg := s.effectiveConfig

	s.lintRequest.BaseFlag = cfg.Base

	if cfg.Remote != nil {
		s.lintRequest.Remote = *cfg.Remote
	}
	if cfg.SkipCheck != nil {
		s.lintRequest.SkipCheckFlag = *cfg.SkipCheck
	}
	if cfg.Timeout != nil {
		s.lintRequest.TimeoutFlag = *cfg.Timeout
	}
	if cfg.LookupTimeout != nil {
		s.lintRequest.LookupTimeoutFlag = *cfg.LookupTimeout
	}
	if cfg.IgnoreArrayCircleRef != nil {
		s.lintRequest.IgnoreArrayCircleRef = *cfg.IgnoreArrayCircleRef
	}
	if cfg.IgnorePolymorphCircleRef != nil {
		s.lintRequest.IgnorePolymorphCircleRef = *cfg.IgnorePolymorphCircleRef
	}
	if cfg.ExtensionRefs != nil {
		s.lintRequest.ExtensionRefs = *cfg.ExtensionRefs
	}

	// Handle HTTP client config
	s.lintRequest.HTTPClientConfig = utils.HTTPClientConfig{
		CertFile: cfg.CertFile,
		KeyFile:  cfg.KeyFile,
		CAFile:   cfg.CAFile,
	}
	if cfg.Insecure != nil {
		s.lintRequest.HTTPClientConfig.Insecure = *cfg.Insecure
	}

	// Handle ruleset loading (only if changed)
	if err := s.loadRulesetIfChanged(cfg); err != nil {
		s.logger.Warn("failed to load ruleset", "error", err)
	}

	// Handle functions loading (only if changed)
	if err := s.loadFunctionsIfChanged(cfg); err != nil {
		s.logger.Warn("failed to load functions", "error", err)
	}

	return nil
}

// loadRulesetIfChanged loads the ruleset only if the path or hard mode has changed.
func (s *ServerState) loadRulesetIfChanged(cfg *LSPConfig) error {
	effectiveRuleset := cfg.Ruleset
	hardMode := cfg.HardMode != nil && *cfg.HardMode

	// Check if anything changed
	if effectiveRuleset == s.loadedRulesetPath && hardMode == s.loadedHardMode {
		return nil // No change
	}

	// Build default rulesets
	defaultRuleSets := rulesets.BuildDefaultRuleSetsWithLogger(s.logger)
	var selectedRS *rulesets.RuleSet

	if hardMode {
		selectedRS = defaultRuleSets.GenerateOpenAPIDefaultRuleSet()
		owaspRules := rulesets.GetAllOWASPRules()
		for k, v := range owaspRules {
			selectedRS.Rules[k] = v
		}
	} else if effectiveRuleset != "" {
		rsBytes, err := os.ReadFile(effectiveRuleset)
		if err != nil {
			return fmt.Errorf("failed to read ruleset %s: %w", effectiveRuleset, err)
		}
		userRS, err := rulesets.CreateRuleSetFromData(rsBytes)
		if err != nil {
			return fmt.Errorf("failed to parse ruleset: %w", err)
		}
		selectedRS = defaultRuleSets.GenerateRuleSetFromSuppliedRuleSet(userRS)
	} else {
		selectedRS = defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()
	}

	s.lintRequest.DefaultRuleSets = defaultRuleSets
	s.lintRequest.SelectedRS = selectedRS
	s.loadedRulesetPath = effectiveRuleset
	s.loadedHardMode = hardMode

	return nil
}

// loadFunctionsIfChanged loads custom functions only if the path has changed.
func (s *ServerState) loadFunctionsIfChanged(cfg *LSPConfig) error {
	effectiveFunctions := cfg.Functions

	if effectiveFunctions == s.loadedFunctionsPath {
		return nil // No change
	}

	if effectiveFunctions != "" {
		pm, err := plugin.LoadFunctions(effectiveFunctions, true)
		if err != nil {
			return fmt.Errorf("failed to load functions from %s: %w", effectiveFunctions, err)
		}
		s.lintRequest.Functions = pm.GetCustomFunctions()
	} else {
		s.lintRequest.Functions = nil
	}

	s.loadedFunctionsPath = effectiveFunctions
	return nil
}

// relintAllDocuments triggers re-linting of all open documents.
func (s *ServerState) relintAllDocuments(notify glsp.NotifyFunc) {
	s.documentStore.mu.RLock()
	docs := make([]*Document, 0, len(s.documentStore.documents))
	for _, doc := range s.documentStore.documents {
		docs = append(docs, doc)
	}
	s.documentStore.mu.RUnlock()

	for _, doc := range docs {
		s.runDiagnostic(doc, notify)
	}
}

// setNotifyFunc stores the notify function for use by the file watcher.
func (s *ServerState) setNotifyFunc(notify glsp.NotifyFunc) {
	s.notifyMu.Lock()
	s.notifyFunc = notify
	s.notifyMu.Unlock()
}

// getNotifyFunc retrieves the stored notify function.
func (s *ServerState) getNotifyFunc() glsp.NotifyFunc {
	s.notifyMu.RLock()
	defer s.notifyMu.RUnlock()
	return s.notifyFunc
}
