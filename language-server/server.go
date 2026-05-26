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
	server           *glspserv.Server
	documentStore    *DocumentStore
	lintRequest      *utils.LintFileRequest
	executionOptions *motor.ExecutionOptions
	rulesetSelector  RulesetSelector

	// Configuration layers (in order of increasing priority)
	baseConfig    *LSPConfig // From command-line flags (immutable after init)
	fileConfig    *LSPConfig // From vacuum.conf.yaml (updated by file watcher)
	initConfig    *LSPConfig // From InitializationOptions (set once at init)
	runtimeConfig *LSPConfig // From didChangeConfiguration (updated at runtime)

	// Effective configuration (computed from all layers)
	effectiveConfig *LSPConfig

	// Synchronization for config updates
	configMu         sync.RWMutex
	configGeneration uint64

	// Logger for config-related messages
	logger *slog.Logger

	// Cached resource paths to detect changes and avoid reloading
	loadedRulesetPath    string
	loadedFunctionsPath  string
	loadedIgnoreFilePath string
	loadedHardMode       bool

	workspaceConfigurationSupported                    bool
	didChangeConfigurationDynamicRegistrationSupported bool
	workspaceFolders                                   []protocol.WorkspaceFolder
	workspaceMu                                        sync.RWMutex
	configDirectory                                    string
	documentRuntimeConfigs                             map[protocol.DocumentUri]*documentRuntimeConfig
	documentRuntimeConfigMu                            sync.RWMutex

	// Notify function for triggering re-lints (used by file watcher)
	notifyFunc glsp.NotifyFunc
	notifyMu   sync.RWMutex
	callFunc   glsp.CallFunc
	callMu     sync.RWMutex
}

func NewServer(version string, lintRequest *utils.LintFileRequest) *ServerState {
	return NewServerWithExecutionOptions(version, lintRequest, nil)
}

func NewServerWithExecutionOptions(version string, lintRequest *utils.LintFileRequest, executionOptions *motor.ExecutionOptions) *ServerState {
	handler := protocol.Handler{}
	server := glspserv.NewServer(&handler, serverName, true)

	// Initialize logger
	logger := lintRequest.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	}

	// Create base config from initial lintRequest values (command-line flags)
	baseConfig := &LSPConfig{
		Base:          lintRequest.BaseFlag,
		Remote:        boolPtr(lintRequest.Remote),
		SkipCheck:     boolPtr(lintRequest.SkipCheckFlag),
		Timeout:       intPtr(lintRequest.TimeoutFlag),
		LookupTimeout: intPtr(lintRequest.LookupTimeoutFlag),
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
		server:                 server,
		lintRequest:            lintRequest,
		executionOptions:       executionOptions,
		documentStore:          newDocumentStore(),
		logger:                 logger,
		baseConfig:             baseConfig,
		documentRuntimeConfigs: map[protocol.DocumentUri]*documentRuntimeConfig{},
	}
	handler.Initialize = func(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
		if params.Trace != nil {
			protocol.SetTraceValue(*params.Trace)
		}
		state.setClientCapabilities(params.Capabilities)
		state.setWorkspaceFolders(params.RootURI, params.WorkspaceFolders)

		// Extract and apply InitializationOptions if provided
		if params.InitializationOptions != nil {
			initConfig, err := ParseLSPConfig(params.InitializationOptions)
			if err != nil {
				state.logger.Warn("failed to parse InitializationOptions", "error", err)
			} else if initConfig != nil {
				if err := state.setInitConfig(initConfig); err != nil {
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
		applyWorkspaceFolderCapabilities(&serverCapabilities)

		return protocol.InitializeResult{
			Capabilities: serverCapabilities,
			ServerInfo: &protocol.InitializeResultServerInfo{
				Name:    serverName,
				Version: &version,
			},
		}, nil
	}
	handler.Initialized = func(context *glsp.Context, params *protocol.InitializedParams) error {
		state.setCallFunc(context.Call)
		state.registerConfigurationChangeNotifications(context.Call)
		return nil
	}
	handler.SetTrace = func(context *glsp.Context, params *protocol.SetTraceParams) error {
		protocol.SetTraceValue(params.Value)
		return nil
	}
	handler.TextDocumentDidOpen = func(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
		// Store notify function for file watcher re-linting
		state.setNotifyFunc(context.Notify)
		state.setCallFunc(context.Call)

		doc := state.documentStore.Add(params.TextDocument.URI, params.TextDocument.Text)
		state.runDiagnostic(doc, context.Notify)
		return nil
	}
	handler.TextDocumentDidChange = func(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
		state.setCallFunc(context.Call)
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
		state.clearDocumentRuntimeConfig(params.TextDocument.URI)
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
		state.setCallFunc(context.Call)
		if !state.workspaceConfigurationSupported {
			config, err := ParseLSPConfig(params.Settings)
			if err != nil {
				state.logger.Warn("failed to parse configuration settings", "error", err)
				return nil // Don't fail - log and continue with existing config
			}

			if config != nil {
				if err := state.setRuntimeConfig(config); err != nil {
					state.logger.Warn("failed to apply configuration", "error", err)
					return nil
				}
			}
		} else {
			state.bumpConfigGeneration()
		}

		// Trigger re-linting of all open documents
		state.clearDocumentRuntimeConfigCache()
		state.relintAllDocuments(context.Notify)
		return nil
	}
	handler.WorkspaceDidChangeWorkspaceFolders = func(context *glsp.Context, params *protocol.DidChangeWorkspaceFoldersParams) error {
		state.updateWorkspaceFolders(params.Event.Added, params.Event.Removed)
		state.clearDocumentRuntimeConfigCache()
		state.relintAllDocuments(context.Notify)
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

	runtimeConfig, err := s.runtimeConfigForDocument(uri)
	if err != nil {
		s.logger.Warn("failed to build document configuration", "uri", uri, "error", err)
		runtimeConfig = s.defaultRuntimeConfig(uri)
	}

	baseForDoc := runtimeConfig.config.Base
	if baseForDoc == "" {
		baseForDoc = filepath.Dir(specFileName)
	}

	deepGraph := len(runtimeConfig.ignoredResults) > 0
	ignoredResults := runtimeConfig.ignoredResults

	// Build the rule execution config with copied values
	ruleExec := &motor.RuleSetExecution{
		RuleSet:                         runtimeConfig.selectedRS,
		Spec:                            []byte(content),
		SpecFileName:                    specFileName,
		Timeout:                         time.Duration(runtimeConfig.timeoutSeconds()) * time.Second,
		NodeLookupTimeout:               time.Duration(runtimeConfig.lookupTimeoutMilliseconds()) * time.Millisecond,
		CustomFunctions:                 runtimeConfig.functions,
		IgnoreCircularArrayRef:          runtimeConfig.ignoreArrayCircleRef,
		IgnoreCircularPolymorphicRef:    runtimeConfig.ignorePolymorphCircleRef,
		AllowLookup:                     runtimeConfig.remote,
		Base:                            baseForDoc,
		SkipDocumentCheck:               runtimeConfig.skipCheck,
		Logger:                          runtimeConfig.logger,
		BuildDeepGraph:                  deepGraph,
		ExtractReferencesFromExtensions: runtimeConfig.extensionRefs,
		HTTPClientConfig:                runtimeConfig.httpClientConfig,
	}

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
		result := motor.ApplyRulesToRuleSetWithOptions(ruleExec, s.executionOptions)
		defer result.ReleaseOwnedResources()

		ignoreOptions := utils.IgnoreMatcherOptions{
			SpecBytes: []byte(content),
		}
		if result != nil && result.RuleSetExecution != nil {
			ignoreOptions.RootNode = result.RuleSetExecution.CanonicalDocument
		}
		filteredResults := utils.FilterIgnoredResultsWithOptions(result.Results, ignoredResults, ignoreOptions)
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

func applyWorkspaceFolderCapabilities(capabilities *protocol.ServerCapabilities) {
	if capabilities.Workspace == nil {
		capabilities.Workspace = &protocol.ServerCapabilitiesWorkspace{}
	}
	capabilities.Workspace.WorkspaceFolders = &protocol.WorkspaceFoldersServerCapabilities{
		Supported:           boolPtr(true),
		ChangeNotifications: &protocol.BoolOrString{Value: true},
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
	s.configGeneration++

	return s.applyEffectiveConfigLocked()
}

func (s *ServerState) setInitConfig(config *LSPConfig) error {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	s.initConfig = config
	s.configGeneration++
	return s.applyEffectiveConfigLocked()
}

func (s *ServerState) setRuntimeConfig(config *LSPConfig) error {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	s.runtimeConfig = config
	s.configGeneration++
	return s.applyEffectiveConfigLocked()
}

func (s *ServerState) setFileConfig(config *LSPConfig, configDirectory string) error {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	s.fileConfig = config
	if configDirectory != "" {
		s.configDirectory = configDirectory
	}
	s.configGeneration++
	return s.applyEffectiveConfigLocked()
}

func (s *ServerState) bumpConfigGeneration() {
	s.configMu.Lock()
	s.configGeneration++
	s.configMu.Unlock()
}

// applyEffectiveConfigLocked computes the effective configuration from all
// layers and updates the lintRequest accordingly. Must be called with configMu
// held.
func (s *ServerState) applyEffectiveConfigLocked() error {
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
	if err := s.loadIgnoreFileIfChanged(cfg); err != nil {
		s.logger.Warn("failed to load ignore file", "error", err)
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
		selectedRS = generateHardModeRuleSet(defaultRuleSets)
	} else if effectiveRuleset != "" {
		ruleset, err := s.loadRulesetForDocument(effectiveRuleset, cfg, defaultRuleSets, s.lintRequest.HTTPClientConfig, "")
		if err != nil {
			return err
		}
		selectedRS = ruleset
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
		resolvedFunctions, err := s.resolveDocumentConfigPath(effectiveFunctions, "")
		if err != nil {
			return err
		}
		pm, err := plugin.LoadFunctions(resolvedFunctions, true)
		if err != nil {
			return fmt.Errorf("failed to load functions from %s: %w", resolvedFunctions, err)
		}
		s.lintRequest.Functions = pm.GetCustomFunctions()
	} else {
		s.lintRequest.Functions = nil
	}

	s.loadedFunctionsPath = effectiveFunctions
	return nil
}

func (s *ServerState) loadIgnoreFileIfChanged(cfg *LSPConfig) error {
	effectiveIgnoreFile := cfg.IgnoreFile

	if effectiveIgnoreFile == s.loadedIgnoreFilePath {
		return nil
	}

	if effectiveIgnoreFile != "" {
		resolvedIgnoreFile, err := s.resolveDocumentConfigPath(effectiveIgnoreFile, "")
		if err != nil {
			return err
		}
		ignoredResults, err := loadIgnoreFileForLSP(resolvedIgnoreFile)
		if err != nil {
			return err
		}
		s.lintRequest.IgnoredResults = ignoredResults
	} else {
		s.lintRequest.IgnoredResults = model.IgnoredItems{}
	}

	s.loadedIgnoreFilePath = effectiveIgnoreFile
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
