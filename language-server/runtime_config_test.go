// Copyright 2024-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package languageserver

import (
	"errors"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/daveshanley/vacuum/language-server/internal/lsp"
	"github.com/daveshanley/vacuum/language-server/protocol"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestRuntimeConfigForDocument_PullsWorkspaceConfiguration(t *testing.T) {
	tempDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "ruleset.yaml"), []byte("extends: [[vacuum:oas, off]]"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "ignore.yaml"), []byte("oas3-schema:\n  - $.info.title\n"), 0o600))

	defaultRuleSets := rulesets.BuildDefaultRuleSets()
	state := &ServerState{
		lintRequest: &utils.LintFileRequest{
			DefaultRuleSets:      defaultRuleSets,
			SelectedRS:           defaultRuleSets.GenerateOpenAPIRecommendedRuleSet(),
			Remote:               true,
			TimeoutFlag:          5,
			LookupTimeoutFlag:    500,
			HTTPClientConfig:     utils.HTTPClientConfig{},
			Logger:               slog.New(slog.NewTextHandler(io.Discard, nil)),
			SkipCheckFlag:        false,
			ExtensionRefs:        false,
			IgnoreArrayCircleRef: false,
		},
		logger:                          slog.New(slog.NewTextHandler(io.Discard, nil)),
		baseConfig:                      &LSPConfig{Remote: boolPtr(true), Timeout: intPtr(5), LookupTimeout: intPtr(500)},
		workspaceConfigurationSupported: true,
		documentRuntimeConfigs:          map[protocol.DocumentUri]*documentRuntimeConfig{},
	}
	rootURI := fileURI(tempDir)
	specURI := fileURI(filepath.Join(tempDir, "openapi.yaml"))
	state.setWorkspaceFolders(nil, []protocol.WorkspaceFolder{{URI: rootURI, Name: "test-workspace"}})
	state.setCallFunc(lsp.CallFunc(func(method string, params any, result any) error {
		assert.Equal(t, string(protocol.ServerWorkspaceConfiguration), method)
		configResult, ok := result.(*[]any)
		require.True(t, ok)
		*configResult = []any{
			map[string]any{
				"ruleset":       "ruleset.yaml",
				"ignoreFile":    "ignore.yaml",
				"remote":        false,
				"skipCheck":     true,
				"timeout":       2,
				"lookupTimeout": 25,
				"extensionRefs": true,
			},
		}
		return nil
	}))

	runtimeConfig, err := state.runtimeConfigForDocument(specURI)

	require.NoError(t, err)
	require.NotNil(t, runtimeConfig.selectedRS)
	assert.Len(t, runtimeConfig.ignoredResults, 1)
	assert.False(t, *runtimeConfig.config.Remote)
	assert.True(t, *runtimeConfig.config.SkipCheck)
	assert.Equal(t, 2, runtimeConfig.timeoutSeconds())
	assert.Equal(t, 25, runtimeConfig.lookupTimeoutMilliseconds())
	assert.True(t, *runtimeConfig.config.ExtensionRefs)
}

func TestRuntimeConfigForDocument_NullWorkspaceDefaultsDoNotOverrideFileConfig(t *testing.T) {
	state := newRuntimeConfigTestState()
	require.NoError(t, state.setFileConfig(&LSPConfig{
		Remote:        boolPtr(false),
		SkipCheck:     boolPtr(true),
		HardMode:      boolPtr(true),
		LookupTimeout: intPtr(42),
	}, ""))
	state.workspaceConfigurationSupported = true
	state.setCallFunc(lsp.CallFunc(func(method string, params any, result any) error {
		configResult, ok := result.(*[]any)
		require.True(t, ok)
		*configResult = []any{
			map[string]any{
				"ruleset":       nil,
				"ignoreFile":    nil,
				"functions":     nil,
				"base":          nil,
				"remote":        nil,
				"skipCheck":     nil,
				"timeout":       nil,
				"lookupTimeout": nil,
				"hardMode":      nil,
				"extensionRefs": nil,
			},
		}
		return nil
	}))

	runtimeConfig, err := state.runtimeConfigForDocument(fileURI(filepath.Join(t.TempDir(), "openapi.yaml")))

	require.NoError(t, err)
	assert.False(t, runtimeConfig.remote)
	assert.True(t, runtimeConfig.skipCheck)
	assert.True(t, *runtimeConfig.config.HardMode)
	assert.Equal(t, 42, runtimeConfig.lookupTimeoutMilliseconds())
}

func TestRuntimeConfigForDocument_UserWorkspaceValuesOverrideFileConfig(t *testing.T) {
	state := newRuntimeConfigTestState()
	require.NoError(t, state.setFileConfig(&LSPConfig{
		Remote:        boolPtr(false),
		SkipCheck:     boolPtr(true),
		LookupTimeout: intPtr(42),
	}, ""))
	state.workspaceConfigurationSupported = true
	state.setCallFunc(lsp.CallFunc(func(method string, params any, result any) error {
		configResult, ok := result.(*[]any)
		require.True(t, ok)
		*configResult = []any{
			map[string]any{
				"remote":        true,
				"skipCheck":     false,
				"lookupTimeout": 25,
			},
		}
		return nil
	}))

	runtimeConfig, err := state.runtimeConfigForDocument(fileURI(filepath.Join(t.TempDir(), "openapi.yaml")))

	require.NoError(t, err)
	assert.True(t, runtimeConfig.remote)
	assert.False(t, runtimeConfig.skipCheck)
	assert.Equal(t, 25, runtimeConfig.lookupTimeoutMilliseconds())
}

func TestRuntimeConfigForDocument_WorkspaceHardModeFalseRebuildsRecommendedRuleset(t *testing.T) {
	state := newRuntimeConfigTestState()
	require.NoError(t, state.setFileConfig(&LSPConfig{HardMode: boolPtr(true)}, ""))
	state.workspaceConfigurationSupported = true
	state.setCallFunc(lsp.CallFunc(func(method string, params any, result any) error {
		configResult, ok := result.(*[]any)
		require.True(t, ok)
		*configResult = []any{
			map[string]any{
				"hardMode": false,
			},
		}
		return nil
	}))

	runtimeConfig, err := state.runtimeConfigForDocument(fileURI(filepath.Join(t.TempDir(), "openapi.yaml")))

	require.NoError(t, err)
	require.NotNil(t, runtimeConfig.selectedRS)
	require.NotNil(t, runtimeConfig.config.HardMode)
	assert.False(t, *runtimeConfig.config.HardMode)
	assert.NotContains(t, runtimeConfig.selectedRS.Rules, rulesets.OwaspNoNumericIDs)
}

func TestRuntimeConfigForDocument_RebuildsWhenConfigGenerationChangesDuringBuild(t *testing.T) {
	state := newRuntimeConfigTestState()
	state.workspaceConfigurationSupported = true
	specURI := fileURI(filepath.Join(t.TempDir(), "openapi.yaml"))
	calls := 0
	state.setCallFunc(lsp.CallFunc(func(method string, params any, result any) error {
		calls++
		configResult, ok := result.(*[]any)
		require.True(t, ok)
		if calls == 1 {
			*configResult = []any{
				map[string]any{
					"remote": false,
				},
			}
			state.bumpConfigGeneration()
			return nil
		}
		*configResult = []any{
			map[string]any{
				"remote": true,
			},
		}
		return nil
	}))

	runtimeConfig, err := state.runtimeConfigForDocument(specURI)

	require.NoError(t, err)
	assert.Equal(t, 2, calls)
	assert.True(t, runtimeConfig.remote)
	require.NotNil(t, state.cachedDocumentRuntimeConfig(specURI))
	assert.True(t, state.cachedDocumentRuntimeConfig(specURI).remote)
}

func TestRuntimeConfigForDocument_CachesWorkspaceCallFailureUntilGenerationChanges(t *testing.T) {
	state := newRuntimeConfigTestState()
	state.workspaceConfigurationSupported = true
	specURI := fileURI(filepath.Join(t.TempDir(), "openapi.yaml"))
	callFailure := errors.New("client disconnected")
	calls := 0
	state.setCallFunc(func(method string, params any, result any) error {
		calls++
		return callFailure
	})

	runtimeConfig, err := state.runtimeConfigForDocument(specURI)
	assert.Nil(t, runtimeConfig)
	require.ErrorIs(t, err, callFailure)
	require.NotNil(t, state.cachedDocumentRuntimeConfig(specURI))

	runtimeConfig, err = state.runtimeConfigForDocument(specURI)
	require.NoError(t, err)
	require.NotNil(t, runtimeConfig)
	assert.Equal(t, 1, calls)

	state.bumpConfigGeneration()
	runtimeConfig, err = state.runtimeConfigForDocument(specURI)
	assert.Nil(t, runtimeConfig)
	require.ErrorIs(t, err, callFailure)
	assert.Equal(t, 2, calls)
}

func TestRegisterConfigurationChangeNotificationsReturnsClientFailure(t *testing.T) {
	state := newRuntimeConfigTestState()
	state.didChangeConfigurationDynamicRegistrationSupported = true
	callFailure := errors.New("registration rejected")

	err := state.registerConfigurationChangeNotifications(
		func(method string, params any, result any) error {
			assert.Equal(t, string(protocol.ServerClientRegisterCapability), method)
			return callFailure
		},
	)

	require.ErrorIs(t, err, callFailure)
}

func TestRuntimeConfigForDocument_PreservesCLIProvidedLookupTimeout(t *testing.T) {
	state := newRuntimeConfigTestState()
	state.baseConfig = nil
	state.lintRequest.LookupTimeoutFlag = 123

	runtimeConfig, err := state.runtimeConfigForDocument(fileURI(filepath.Join(t.TempDir(), "openapi.yaml")))

	require.NoError(t, err)
	assert.Equal(t, 123, runtimeConfig.lookupTimeoutMilliseconds())
}

func TestApplyWorkspaceFolderCapabilities(t *testing.T) {
	capabilities := protocol.ServerCapabilities{}

	applyWorkspaceFolderCapabilities(&capabilities)

	require.NotNil(t, capabilities.Workspace)
	require.NotNil(t, capabilities.Workspace.WorkspaceFolders)
	require.NotNil(t, capabilities.Workspace.WorkspaceFolders.Supported)
	assert.True(t, *capabilities.Workspace.WorkspaceFolders.Supported)
	require.NotNil(t, capabilities.Workspace.WorkspaceFolders.ChangeNotifications)
	assert.Equal(t, true, capabilities.Workspace.WorkspaceFolders.ChangeNotifications.Value)
}

func TestResolveDocumentConfigPath_UsesWorkspaceFolderForRelativePaths(t *testing.T) {
	tempDir := t.TempDir()
	nestedDir := filepath.Join(tempDir, "apis")
	require.NoError(t, os.MkdirAll(nestedDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "ruleset.yaml"), []byte("extends: [[vacuum:oas, off]]"), 0o600))

	state := &ServerState{}
	state.setWorkspaceFolders(nil, []protocol.WorkspaceFolder{{URI: fileURI(tempDir), Name: "test-workspace"}})

	resolved, err := state.resolveDocumentConfigPath("ruleset.yaml", fileURI(filepath.Join(nestedDir, "openapi.yaml")))

	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tempDir, "ruleset.yaml"), resolved)
}

func newRuntimeConfigTestState() *ServerState {
	defaultRuleSets := rulesets.BuildDefaultRuleSets()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return &ServerState{
		lintRequest: &utils.LintFileRequest{
			DefaultRuleSets:   defaultRuleSets,
			SelectedRS:        defaultRuleSets.GenerateOpenAPIRecommendedRuleSet(),
			Remote:            true,
			TimeoutFlag:       5,
			LookupTimeoutFlag: 500,
			HTTPClientConfig:  utils.HTTPClientConfig{},
			Logger:            logger,
		},
		logger:                 logger,
		baseConfig:             &LSPConfig{Remote: boolPtr(true), Timeout: intPtr(5), LookupTimeout: intPtr(500)},
		documentRuntimeConfigs: map[protocol.DocumentUri]*documentRuntimeConfig{},
	}
}

func fileURI(pathValue string) protocol.DocumentUri {
	return protocol.DocumentUri((&url.URL{Scheme: "file", Path: filepath.ToSlash(pathValue)}).String())
}
