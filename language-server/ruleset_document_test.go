package languageserver

import (
	"errors"
	"github.com/daveshanley/vacuum/rulesets"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/language-server/protocol"
	"github.com/pb33f/testify/require"
)

func TestRunDiagnostic_RulesetDocuments(t *testing.T) {
	for _, tc := range []struct {
		name, content, code string
		empty               bool
	}{
		{"yaml", "extends: [spectral:oas]\nrules: {operation-description: off}\n", "", true},
		{"json", `{"rules":{"no-title":{"given":"$.info.title","then":{"function":"falsy"}}}}`, "", true},
		{"extends only", "extends: [./child.yml]\n", "", true},
		{"invalid rules", "rules: []\n", "ruleset-error", false},
		{"unsupported module", "extends: [rules.mjs]\n", "ruleset-error", false},
		{"Arazzo", "extends: [spectral:arazzo]\n", "", true},
		{"unsupported API with rules", "openapi: 9.0.0\nrules: {}\n", "build-index", false},
		{"nested rules", "config:\n  rules: {}\n", "document-error", false},
		{"malformed rules", "rules: [\n", "document-error", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := newRuntimeConfigTestState()
			published := make(chan protocol.PublishDiagnosticsParams, 1)
			state.runDiagnostic(&Document{URI: "file:///workspace/main.yaml", Content: tc.content}, func(_ string, params any) { published <- params.(protocol.PublishDiagnosticsParams) })
			select {
			case got := <-published:
				require.Equal(t, "file:///workspace/main.yaml", got.URI)
				if tc.empty {
					require.NotNil(t, got.Diagnostics, "an empty array must clear prior editor diagnostics")
					require.Empty(t, got.Diagnostics)
				} else {
					require.NotEmpty(t, got.Diagnostics)
					require.Equal(t, tc.code, got.Diagnostics[0].Code.Value)
					if tc.code == "document-error" || tc.code == "ruleset-error" {
						require.Nil(t, got.Diagnostics[0].CodeDescription, "tool errors have no rule documentation page")
					}
				}
			case <-time.After(5 * time.Second):
				t.Fatal("no diagnostics published")
			}
		})
	}
}

func TestRunDiagnostic_RulesetDocumentPreservesConfigurationErrors(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		configure func(*ServerState)
		message   string
	}{
		{"TLS", func(s *ServerState) { s.baseConfig.CertFile = "/missing-cert-941.pem" }, "cert-file"},
		{"selector function", func(s *ServerState) {
			s.rulesetSelector = func(*DocumentContext) *rulesets.RuleSet {
				rs, err := rulesets.CreateRuleSetFromData([]byte("rules:\n  broken:\n    given: $\n    then: {function: missing941}\n"))
				if err != nil {
					panic(err)
				}
				return rs
			}
		}, "missing941"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			state := newRuntimeConfigTestState()
			testCase.configure(state)
			published := make(chan protocol.PublishDiagnosticsParams, 1)
			state.runDiagnostic(&Document{URI: "file:///workspace/rules.yaml", Content: "rules: {}\n"}, func(_ string, p any) { published <- p.(protocol.PublishDiagnosticsParams) })
			select {
			case got := <-published:
				require.NotEmpty(t, got.Diagnostics)
				require.Contains(t, got.Diagnostics[0].Message, testCase.message)
			case <-time.After(5 * time.Second):
				t.Fatal("configuration error was not published")
			}
		})
	}
}

func TestOnlyUnsupportedDocumentErrors(t *testing.T) {
	unsupported := errors.New("spec type not supported by libopenapi, sorry")
	require.True(t, onlyUnsupportedDocumentErrors([]error{errors.Join(unsupported, unsupported)}))
	require.False(t, onlyUnsupportedDocumentErrors([]error{errors.Join(unsupported, errors.New("TLS configuration failed"))}))
	require.False(t, onlyUnsupportedDocumentErrors(nil))
}

func TestRunDiagnostic_RulesetShapeCanBeLintedAsFragment(t *testing.T) {
	state := newRuntimeConfigTestState()
	state.baseConfig.SkipCheck = boolPtr(true)
	state.rulesetSelector = func(*DocumentContext) *rulesets.RuleSet {
		rs, err := rulesets.CreateRuleSetFromData([]byte("rules:\n  fragment-rule:\n    given: $.rules\n    then: {function: falsy}\n"))
		require.NoError(t, err)
		return rs
	}
	published := make(chan protocol.PublishDiagnosticsParams, 1)
	state.runDiagnostic(&Document{URI: "file:///workspace/fragment.yaml", Content: "rules: {enabled: true}\n"}, func(_ string, p any) { published <- p.(protocol.PublishDiagnosticsParams) })
	select {
	case got := <-published:
		require.Len(t, got.Diagnostics, 1)
		require.Equal(t, "fragment-rule", got.Diagnostics[0].Code.Value)
	case <-time.After(5 * time.Second):
		t.Fatal("fragment result was not published")
	}
}
