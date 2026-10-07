package languageserver

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/language-server/protocol"
	"github.com/pb33f/testify/require"
)

func TestRunDiagnostic_RejectsInvalidRuleset(t *testing.T) {
	for _, tc := range []struct{ name, rules, message string }{
		{"missing-extension", "extends: [./missing-extension-945.yaml]\n", "missing-extension-945.yaml"},
		{"unknown-function", "rules:\n  typo:\n    given: '$.missing'\n    severity: info\n    then: {function: patternd}\n", "patternd"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			rules := filepath.Join(dir, "rules.yaml")
			require.NoError(t, os.WriteFile(rules, []byte(tc.rules), 0600))
			state := newRuntimeConfigTestState()
			state.baseConfig.Ruleset = rules
			published := make(chan protocol.PublishDiagnosticsParams, 1)
			uri := fileURI(filepath.Join(dir, "api.yaml"))
			state.runDiagnostic(&Document{URI: uri, Content: "openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\n"},
				func(_ string, params any) { published <- params.(protocol.PublishDiagnosticsParams) })
			select {
			case got := <-published:
				require.Equal(t, uri, got.URI)
				require.Len(t, got.Diagnostics, 1, "default rules must not replace an invalid custom ruleset")
				require.Equal(t, "document-error", got.Diagnostics[0].Code.Value)
				require.Contains(t, got.Diagnostics[0].Message, tc.message)
				require.NotNil(t, got.Diagnostics[0].Severity)
				require.Equal(t, protocol.DiagnosticSeverityError, *got.Diagnostics[0].Severity)
			case <-time.After(5 * time.Second):
				t.Fatal("no diagnostic published for invalid ruleset")
			}
		})
	}
}

func TestRunDiagnostic_DoesNotCacheInvalidWorkspaceFallback(t *testing.T) {
	dir := t.TempDir()
	rules := filepath.Join(dir, "rules.yaml")
	require.NoError(t, os.WriteFile(rules, []byte("extends: [./missing-workspace-extension.yaml]\n"), 0600))
	state := newRuntimeConfigTestState()
	state.baseConfig.Ruleset = rules
	state.workspaceConfigurationSupported = true
	state.setCallFunc(func(string, any, any) error { return errors.New("workspace configuration unavailable") })
	uri := fileURI(filepath.Join(dir, "api.yaml"))
	doc := &Document{URI: uri, Content: "openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\n"}
	for range 2 {
		published := make(chan protocol.PublishDiagnosticsParams, 1)
		state.runDiagnostic(doc, func(_ string, params any) { published <- params.(protocol.PublishDiagnosticsParams) })
		select {
		case got := <-published:
			require.Len(t, got.Diagnostics, 1)
			require.Equal(t, "document-error", got.Diagnostics[0].Code.Value)
			require.Contains(t, got.Diagnostics[0].Message, "missing-workspace-extension.yaml")
		case <-time.After(5 * time.Second):
			t.Fatal("no configuration diagnostic published")
		}
	}
}

func TestRuntimeConfig_RelativeRulesetExtensions(t *testing.T) {
	dir := t.TempDir()
	rules := filepath.Join(dir, "main.yaml")
	require.NoError(t, os.WriteFile(rules, []byte("extends: [./child.yaml]\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "child.yaml"), []byte("rules:\n  child-rule:\n    given: $.info.title\n    then: {function: falsy}\n"), 0600))
	state := newRuntimeConfigTestState()
	state.baseConfig.Ruleset = rules
	config, err := state.runtimeConfigForDocument(fileURI(filepath.Join(dir, "api.yaml")))
	require.NoError(t, err)
	require.Contains(t, config.selectedRS.Rules, "child-rule")
}
