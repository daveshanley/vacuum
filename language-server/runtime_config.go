// Copyright 2024-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package languageserver

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/plugin"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"go.yaml.in/yaml/v4"
)

const lspConfigurationSection = "vacuum"

type documentRuntimeConfig struct {
	generation                     uint64
	config                         *LSPConfig
	defaultRuleSets                rulesets.RuleSets
	selectedRS                     *rulesets.RuleSet
	functions                      map[string]model.RuleFunction
	ignoredResults                 model.IgnoredItems
	httpClientConfig               utils.HTTPClientConfig
	logger                         *slog.Logger
	remote                         bool
	skipCheck                      bool
	ignoreArrayCircleRef           bool
	ignorePolymorphCircleRef       bool
	extensionRefs                  bool
	timeoutSecondsValue            int
	lookupTimeoutMillisecondsValue int
}

type lintRequestSnapshot struct {
	defaultRuleSets          rulesets.RuleSets
	selectedRS               *rulesets.RuleSet
	functions                map[string]model.RuleFunction
	ignoredResults           model.IgnoredItems
	httpClientConfig         utils.HTTPClientConfig
	logger                   *slog.Logger
	remote                   bool
	skipCheck                bool
	ignoreArrayCircleRef     bool
	ignorePolymorphCircleRef bool
	extensionRefs            bool
	timeoutFlag              int
	lookupTimeoutFlag        int
}

func (c *documentRuntimeConfig) timeoutSeconds() int {
	return c.timeoutSecondsValue
}

func (c *documentRuntimeConfig) lookupTimeoutMilliseconds() int {
	return c.lookupTimeoutMillisecondsValue
}

func (s *ServerState) setClientCapabilities(capabilities protocol.ClientCapabilities) {
	if capabilities.Workspace == nil {
		return
	}
	workspace := capabilities.Workspace
	if workspace.Configuration != nil {
		s.workspaceConfigurationSupported = *workspace.Configuration
	}
	if workspace.DidChangeConfiguration != nil && workspace.DidChangeConfiguration.DynamicRegistration != nil {
		s.didChangeConfigurationDynamicRegistrationSupported = *workspace.DidChangeConfiguration.DynamicRegistration
	}
}

func (s *ServerState) registerConfigurationChangeNotifications(call glsp.CallFunc) {
	if call == nil || !s.didChangeConfigurationDynamicRegistrationSupported {
		return
	}

	var result any
	call(string(protocol.ServerClientRegisterCapability), protocol.RegistrationParams{
		Registrations: []protocol.Registration{
			{
				ID:     "vacuum-workspace-configuration",
				Method: string(protocol.MethodWorkspaceDidChangeConfiguration),
				RegisterOptions: map[string]any{
					"section": lspConfigurationSection,
				},
			},
		},
	}, &result)
}

func (s *ServerState) runtimeConfigForDocument(uri protocol.DocumentUri) (*documentRuntimeConfig, error) {
	for {
		if cached := s.cachedDocumentRuntimeConfig(uri); cached != nil {
			return cached, nil
		}

		config, snapshot, generation := s.baseEffectiveConfig(!s.workspaceConfigurationSupported)
		if workspaceConfig, ok := s.pullWorkspaceConfiguration(uri); ok {
			MergeConfig(config, workspaceConfig)
		}

		runtimeConfig, err := s.buildDocumentRuntimeConfig(config, uri, snapshot)
		if err != nil {
			return nil, err
		}

		if s.cacheDocumentRuntimeConfig(uri, runtimeConfig, generation) {
			return runtimeConfig, nil
		}
	}
}

func (s *ServerState) defaultRuntimeConfig(uri protocol.DocumentUri) *documentRuntimeConfig {
	config, snapshot, _ := s.baseEffectiveConfig(!s.workspaceConfigurationSupported)
	runtimeConfig, err := s.buildDocumentRuntimeConfig(config, uri, snapshot)
	if err == nil {
		return runtimeConfig
	}
	return s.fallbackRuntimeConfig(config, snapshot)
}

func (s *ServerState) fallbackRuntimeConfig(config *LSPConfig, snapshot lintRequestSnapshot) *documentRuntimeConfig {
	defaultRuleSets := snapshot.defaultRuleSets
	if defaultRuleSets == nil {
		defaultRuleSets = rulesets.BuildDefaultRuleSetsWithLogger(s.logger)
	}
	selectedRS := snapshot.selectedRS
	if config != nil && config.HardMode != nil {
		if *config.HardMode {
			selectedRS = generateHardModeRuleSet(defaultRuleSets)
		} else if config.Ruleset == "" {
			defaultRuleSets = rulesets.BuildDefaultRuleSetsWithLogger(s.logger)
			selectedRS = defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()
		}
	}
	if selectedRS == nil {
		selectedRS = defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()
	}
	if config == nil {
		config = &LSPConfig{
			Remote:        boolPtr(snapshot.remote),
			SkipCheck:     boolPtr(snapshot.skipCheck),
			Timeout:       intPtr(nonZeroInt(snapshot.timeoutFlag, 5)),
			LookupTimeout: intPtr(nonZeroInt(snapshot.lookupTimeoutFlag, 500)),
		}
	}
	return &documentRuntimeConfig{
		config:                         config,
		defaultRuleSets:                defaultRuleSets,
		selectedRS:                     selectedRS,
		functions:                      snapshot.functions,
		ignoredResults:                 snapshot.ignoredResults,
		httpClientConfig:               snapshot.httpClientConfig,
		logger:                         snapshot.logger,
		remote:                         boolValue(config.Remote, snapshot.remote),
		skipCheck:                      boolValue(config.SkipCheck, snapshot.skipCheck),
		ignoreArrayCircleRef:           boolValue(config.IgnoreArrayCircleRef, snapshot.ignoreArrayCircleRef),
		ignorePolymorphCircleRef:       boolValue(config.IgnorePolymorphCircleRef, snapshot.ignorePolymorphCircleRef),
		extensionRefs:                  boolValue(config.ExtensionRefs, snapshot.extensionRefs),
		timeoutSecondsValue:            intValue(config.Timeout, nonZeroInt(snapshot.timeoutFlag, 5)),
		lookupTimeoutMillisecondsValue: intValue(config.LookupTimeout, nonZeroInt(snapshot.lookupTimeoutFlag, 500)),
	}
}

func (s *ServerState) baseEffectiveConfig(includeRuntime bool) (*LSPConfig, lintRequestSnapshot, uint64) {
	effective := &LSPConfig{
		Remote:  boolPtr(true),
		Timeout: intPtr(5),
	}

	s.configMu.RLock()
	generation := s.configGeneration
	snapshot := s.lintRequestSnapshotLocked()
	MergeConfig(effective, s.baseConfig)
	MergeConfig(effective, s.fileConfig)
	MergeConfig(effective, s.initConfig)
	if includeRuntime {
		MergeConfig(effective, s.runtimeConfig)
	}

	if effective.LookupTimeout == nil || *effective.LookupTimeout == 0 {
		effective.LookupTimeout = intPtr(nonZeroInt(snapshot.lookupTimeoutFlag, 500))
	}
	if effective.Timeout == nil || *effective.Timeout == 0 {
		effective.Timeout = intPtr(nonZeroInt(snapshot.timeoutFlag, 5))
	}
	s.configMu.RUnlock()
	return effective, snapshot, generation
}

func (s *ServerState) lintRequestSnapshotLocked() lintRequestSnapshot {
	snapshot := lintRequestSnapshot{
		logger:            s.logger,
		remote:            true,
		timeoutFlag:       5,
		lookupTimeoutFlag: 500,
	}
	if s.lintRequest == nil {
		return snapshot
	}

	snapshot.defaultRuleSets = s.lintRequest.DefaultRuleSets
	snapshot.selectedRS = s.lintRequest.SelectedRS
	snapshot.functions = s.lintRequest.Functions
	snapshot.ignoredResults = s.lintRequest.IgnoredResults
	snapshot.httpClientConfig = s.lintRequest.HTTPClientConfig
	if s.lintRequest.Logger != nil {
		snapshot.logger = s.lintRequest.Logger
	}
	snapshot.remote = s.lintRequest.Remote
	snapshot.skipCheck = s.lintRequest.SkipCheckFlag
	snapshot.ignoreArrayCircleRef = s.lintRequest.IgnoreArrayCircleRef
	snapshot.ignorePolymorphCircleRef = s.lintRequest.IgnorePolymorphCircleRef
	snapshot.extensionRefs = s.lintRequest.ExtensionRefs
	snapshot.timeoutFlag = s.lintRequest.TimeoutFlag
	snapshot.lookupTimeoutFlag = s.lintRequest.LookupTimeoutFlag
	return snapshot
}

func boolValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func intValue(value *int, fallback int) int {
	if value == nil || *value == 0 {
		return fallback
	}
	return *value
}

func nonZeroInt(value int, fallback int) int {
	if value == 0 {
		return fallback
	}
	return value
}

func generateHardModeRuleSet(defaultRuleSets rulesets.RuleSets) *rulesets.RuleSet {
	base := defaultRuleSets.GenerateOpenAPIDefaultRuleSet()
	owaspRules := rulesets.GetAllOWASPRules()
	selectedRS := *base
	selectedRS.Rules = make(map[string]*model.Rule, len(base.Rules)+len(owaspRules))
	for k, v := range base.Rules {
		selectedRS.Rules[k] = v
	}
	for k, v := range owaspRules {
		selectedRS.Rules[k] = v
	}
	return &selectedRS
}

func (s *ServerState) pullWorkspaceConfiguration(uri protocol.DocumentUri) (*LSPConfig, bool) {
	if !s.workspaceConfigurationSupported {
		return nil, false
	}
	call := s.getCallFunc()
	if call == nil {
		return nil, false
	}

	scopeURI := uri
	section := lspConfigurationSection
	var result []any
	call(string(protocol.ServerWorkspaceConfiguration), protocol.ConfigurationParams{
		Items: []protocol.ConfigurationItem{
			{
				ScopeURI: &scopeURI,
				Section:  &section,
			},
		},
	}, &result)
	if len(result) == 0 || result[0] == nil {
		return nil, true
	}

	config, err := ParseLSPConfig(result[0])
	if err != nil {
		s.logger.Warn("failed to parse workspace configuration", "uri", uri, "error", err)
		return nil, true
	}
	return config, true
}

func (s *ServerState) buildDocumentRuntimeConfig(config *LSPConfig, uri protocol.DocumentUri, snapshot lintRequestSnapshot) (*documentRuntimeConfig, error) {
	httpClientConfig := snapshot.httpClientConfig
	if config.CertFile != "" {
		httpClientConfig.CertFile = config.CertFile
	}
	if config.KeyFile != "" {
		httpClientConfig.KeyFile = config.KeyFile
	}
	if config.CAFile != "" {
		httpClientConfig.CAFile = config.CAFile
	}
	if config.Insecure != nil {
		httpClientConfig.Insecure = *config.Insecure
	}

	defaultRuleSets := snapshot.defaultRuleSets
	if defaultRuleSets == nil {
		defaultRuleSets = rulesets.BuildDefaultRuleSetsWithLogger(s.logger)
	}
	hardMode := config.HardMode != nil && *config.HardMode
	selectedRS := snapshot.selectedRS
	if hardMode {
		selectedRS = generateHardModeRuleSet(defaultRuleSets)
	} else if config.Ruleset != "" {
		ruleset, err := s.loadRulesetForDocument(config.Ruleset, config, defaultRuleSets, httpClientConfig, uri)
		if err != nil {
			return nil, err
		}
		selectedRS = ruleset
	} else if config.HardMode != nil {
		defaultRuleSets = rulesets.BuildDefaultRuleSetsWithLogger(s.logger)
		selectedRS = defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()
	} else if selectedRS == nil {
		selectedRS = defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()
	}

	functions := snapshot.functions
	if config.Functions != "" {
		resolvedFunctions, err := s.resolveDocumentConfigPath(config.Functions, uri)
		if err != nil {
			return nil, err
		}
		pm, err := plugin.LoadFunctions(resolvedFunctions, true)
		if err != nil {
			return nil, fmt.Errorf("failed to load functions from %s: %w", resolvedFunctions, err)
		}
		functions = pm.GetCustomFunctions()
	}

	ignoredResults := snapshot.ignoredResults
	if config.IgnoreFile != "" {
		resolvedIgnoreFile, err := s.resolveDocumentConfigPath(config.IgnoreFile, uri)
		if err != nil {
			return nil, err
		}
		ignoredResults, err = loadIgnoreFileForLSP(resolvedIgnoreFile)
		if err != nil {
			return nil, err
		}
	}

	return &documentRuntimeConfig{
		config:                         config,
		defaultRuleSets:                defaultRuleSets,
		selectedRS:                     selectedRS,
		functions:                      functions,
		ignoredResults:                 ignoredResults,
		httpClientConfig:               httpClientConfig,
		logger:                         snapshot.logger,
		remote:                         boolValue(config.Remote, snapshot.remote),
		skipCheck:                      boolValue(config.SkipCheck, snapshot.skipCheck),
		ignoreArrayCircleRef:           boolValue(config.IgnoreArrayCircleRef, snapshot.ignoreArrayCircleRef),
		ignorePolymorphCircleRef:       boolValue(config.IgnorePolymorphCircleRef, snapshot.ignorePolymorphCircleRef),
		extensionRefs:                  boolValue(config.ExtensionRefs, snapshot.extensionRefs),
		timeoutSecondsValue:            intValue(config.Timeout, nonZeroInt(snapshot.timeoutFlag, 5)),
		lookupTimeoutMillisecondsValue: intValue(config.LookupTimeout, nonZeroInt(snapshot.lookupTimeoutFlag, 500)),
	}, nil
}

func (s *ServerState) loadRulesetForDocument(rulesetLocation string, config *LSPConfig, defaultRuleSets rulesets.RuleSets, httpClientConfig utils.HTTPClientConfig, uri protocol.DocumentUri) (*rulesets.RuleSet, error) {
	httpClient, err := utils.CreateHTTPClientIfNeeded(httpClientConfig)
	if err != nil {
		return nil, err
	}

	if isRemoteConfigLocation(rulesetLocation) {
		if config.Remote != nil && !*config.Remote {
			return nil, fmt.Errorf("remote ruleset specified but remote resolution is disabled")
		}
		downloadedRS, err := rulesets.DownloadRemoteRuleSet(context.Background(), rulesetLocation, httpClient)
		if err != nil {
			return nil, err
		}
		return defaultRuleSets.GenerateRuleSetFromSuppliedRuleSetWithHTTPClient(downloadedRS, httpClient), nil
	}

	resolvedRuleset, err := s.resolveDocumentConfigPath(rulesetLocation, uri)
	if err != nil {
		return nil, err
	}
	rsBytes, err := os.ReadFile(resolvedRuleset)
	if err != nil {
		return nil, fmt.Errorf("failed to read ruleset %s: %w", resolvedRuleset, err)
	}
	userRS, err := rulesets.CreateRuleSetFromData(rsBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ruleset: %w", err)
	}
	return defaultRuleSets.GenerateRuleSetFromSuppliedRuleSetWithHTTPClient(userRS, httpClient), nil
}

func loadIgnoreFileForLSP(ignoreFile string) (model.IgnoredItems, error) {
	ignoredItems := model.IgnoredItems{}
	if ignoreFile == "" {
		return ignoredItems, nil
	}

	raw, err := os.ReadFile(ignoreFile)
	if err != nil {
		return ignoredItems, fmt.Errorf("failed to read ignore file %s: %w", ignoreFile, err)
	}
	if err := yaml.Unmarshal(raw, &ignoredItems); err != nil {
		return ignoredItems, fmt.Errorf("failed to parse ignore file %s: %w", ignoreFile, err)
	}
	return ignoredItems, nil
}

func (s *ServerState) resolveDocumentConfigPath(raw string, uri protocol.DocumentUri) (string, error) {
	if raw == "" || isRemoteConfigLocation(raw) {
		return raw, nil
	}

	expanded, err := expandLSPPath(raw)
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(expanded) {
		return filepath.Clean(expanded), nil
	}

	dirs := uniqueDirs([]string{
		s.workspaceFolderPathForURI(uri),
		filepath.Dir(fileURIToPath(uri)),
		s.configDirectory,
		currentWorkingDirectory(),
	})
	for _, dir := range dirs {
		candidate := filepath.Clean(filepath.Join(dir, expanded))
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate, nil
		}
	}
	if len(dirs) > 0 {
		return filepath.Clean(filepath.Join(dirs[0], expanded)), nil
	}
	return expanded, nil
}

func expandLSPPath(pathValue string) (string, error) {
	expanded := strings.TrimSpace(pathValue)
	if (strings.HasPrefix(expanded, `"`) && strings.HasSuffix(expanded, `"`)) ||
		(strings.HasPrefix(expanded, `'`) && strings.HasSuffix(expanded, `'`)) {
		expanded = expanded[1 : len(expanded)-1]
	}

	expanded = os.ExpandEnv(expanded)
	if expanded == "~" || strings.HasPrefix(expanded, "~/") || strings.HasPrefix(expanded, "~\\") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("unable to resolve home directory: %w", err)
		}
		if expanded == "~" {
			expanded = home
		} else {
			expanded = filepath.Join(home, expanded[2:])
		}
	}
	return expanded, nil
}

func isRemoteConfigLocation(value string) bool {
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}

func uniqueDirs(dirs []string) []string {
	seen := map[string]struct{}{}
	var unique []string
	for _, dir := range dirs {
		if dir == "" || dir == "." {
			continue
		}
		clean := filepath.Clean(dir)
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		unique = append(unique, clean)
	}
	return unique
}

func currentWorkingDirectory() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return cwd
}

func fileURIToPath(uri protocol.DocumentUri) string {
	raw := string(uri)
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "file" {
		return strings.TrimPrefix(raw, "file://")
	}

	pathValue := filepath.FromSlash(parsed.Path)
	if parsed.Host != "" {
		if os.PathSeparator == '\\' {
			pathValue = `\\` + parsed.Host + pathValue
		} else {
			pathValue = "//" + parsed.Host + pathValue
		}
	}
	if os.PathSeparator == '\\' && len(pathValue) >= 3 && pathValue[0] == '\\' && pathValue[2] == ':' {
		pathValue = pathValue[1:]
	}
	return pathValue
}

func (s *ServerState) setWorkspaceFolders(rootURI *protocol.DocumentUri, folders []protocol.WorkspaceFolder) {
	if len(folders) == 0 && rootURI != nil && *rootURI != "" {
		pathValue := fileURIToPath(*rootURI)
		folders = []protocol.WorkspaceFolder{
			{
				URI:  *rootURI,
				Name: filepath.Base(pathValue),
			},
		}
	}

	s.workspaceMu.Lock()
	s.workspaceFolders = append([]protocol.WorkspaceFolder(nil), folders...)
	s.workspaceMu.Unlock()
}

func (s *ServerState) updateWorkspaceFolders(added, removed []protocol.WorkspaceFolder) {
	s.workspaceMu.Lock()
	defer s.workspaceMu.Unlock()

	removedURIs := map[protocol.DocumentUri]struct{}{}
	for _, folder := range removed {
		removedURIs[folder.URI] = struct{}{}
	}

	next := make([]protocol.WorkspaceFolder, 0, len(s.workspaceFolders)+len(added))
	for _, folder := range s.workspaceFolders {
		if _, ok := removedURIs[folder.URI]; !ok {
			next = append(next, folder)
		}
	}
	next = append(next, added...)
	s.workspaceFolders = next
}

func (s *ServerState) workspaceFolderPathForURI(uri protocol.DocumentUri) string {
	documentPath := fileURIToPath(uri)
	if documentPath == "" {
		return ""
	}

	s.workspaceMu.RLock()
	folders := append([]protocol.WorkspaceFolder(nil), s.workspaceFolders...)
	s.workspaceMu.RUnlock()

	var best string
	for _, folder := range folders {
		folderPath := fileURIToPath(folder.URI)
		if folderPath == "" {
			continue
		}
		rel, err := filepath.Rel(folderPath, documentPath)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			continue
		}
		if len(folderPath) > len(best) {
			best = folderPath
		}
	}
	return best
}

func (s *ServerState) clearDocumentRuntimeConfig(uri protocol.DocumentUri) {
	s.documentRuntimeConfigMu.Lock()
	if s.documentRuntimeConfigs != nil {
		delete(s.documentRuntimeConfigs, uri)
	}
	s.documentRuntimeConfigMu.Unlock()
}

func (s *ServerState) cachedDocumentRuntimeConfig(uri protocol.DocumentUri) *documentRuntimeConfig {
	s.configMu.RLock()
	generation := s.configGeneration
	s.documentRuntimeConfigMu.RLock()
	cached := s.documentRuntimeConfigs[uri]
	if cached != nil && cached.generation != generation {
		cached = nil
	}
	s.documentRuntimeConfigMu.RUnlock()
	s.configMu.RUnlock()
	return cached
}

func (s *ServerState) cacheDocumentRuntimeConfig(uri protocol.DocumentUri, runtimeConfig *documentRuntimeConfig, generation uint64) bool {
	s.configMu.RLock()
	if s.configGeneration != generation {
		s.configMu.RUnlock()
		return false
	}

	runtimeConfig.generation = generation
	s.documentRuntimeConfigMu.Lock()
	if s.documentRuntimeConfigs == nil {
		s.documentRuntimeConfigs = map[protocol.DocumentUri]*documentRuntimeConfig{}
	}
	s.documentRuntimeConfigs[uri] = runtimeConfig
	s.documentRuntimeConfigMu.Unlock()
	s.configMu.RUnlock()
	return true
}

func (s *ServerState) clearDocumentRuntimeConfigCache() {
	s.documentRuntimeConfigMu.Lock()
	s.documentRuntimeConfigs = map[protocol.DocumentUri]*documentRuntimeConfig{}
	s.documentRuntimeConfigMu.Unlock()
}

func (s *ServerState) setCallFunc(call glsp.CallFunc) {
	s.callMu.Lock()
	s.callFunc = call
	s.callMu.Unlock()
}

func (s *ServerState) getCallFunc() glsp.CallFunc {
	s.callMu.RLock()
	defer s.callMu.RUnlock()
	return s.callFunc
}
