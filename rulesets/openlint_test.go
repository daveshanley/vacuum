// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package rulesets

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/testify/require"
)

func loadOpenLintTestSet(t *testing.T, files map[string]string) (*RuleSet, string) {
	t.Helper()
	dir := t.TempDir()
	for name, text := range files {
		target := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(target), 0700))
		require.NoError(t, os.WriteFile(target, []byte(text), 0600))
	}
	rs, err := LoadLocalRuleSetWithContract(context.Background(), filepath.Join(dir, "rules.yaml"), model.OpenLintV1Draft)
	require.NoError(t, err)
	rs = BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(rs)
	require.NoError(t, rs.LoadError())
	return rs, dir
}

func TestOpenLintSchemaAndNativeRemainSeparate(t *testing.T) {
	data := []byte("rules: {}\ndocumentationUrl: https://example.com/docs\nx-tool: true\n")
	rs, err := CreateRuleSetFromDataWithContract(data, model.OpenLintV1Draft)
	require.NoError(t, err)
	require.Equal(t, model.OpenLintV1Draft, rs.Contract)
	_, err = CreateRuleSetFromDataWithContract([]byte("rules: {}\ndocumentationUrl: http://127.0.0.1/docs\n"), model.OpenLintV1Draft)
	require.Error(t, err)
	_, err = CreateRuleSetFromDataWithContract([]byte("description: metadata only\n"), model.OpenLintV1Draft)
	require.Error(t, err)
	_, err = CreateRuleSetFromDataWithContract([]byte("rules: {}\n---\nrules: {}\n"), model.OpenLintV1Draft)
	require.Error(t, err)
	_, err = CreateRuleSetFromData([]byte("description: metadata only\n"))
	require.NoError(t, err)
}

func TestOpenLintOrderedInheritanceAndDisabledDefinitions(t *testing.T) {
	rs, dir := loadOpenLintTestSet(t, map[string]string{
		"base.yaml":  "rules:\n  r:\n    given: $\n    severity: info\n    message: inherited\n    then: {function: falsy}\n  disabled:\n    given: $\n    severity: off\n    then: {function: falsy}\n",
		"next.yaml":  "extends: ./base.yaml\nrules:\n  r: false\n",
		"rules.yaml": "extends: [./base.yaml, ./next.yaml]\nrules:\n  r: true\n  disabled: true\n",
	})
	compiled, err := rs.ForOpenLintDocument(context.Background(), filepath.Join(dir, "document.yaml"), nil)
	require.NoError(t, err)
	require.Len(t, compiled.Rules, 1)
	require.Equal(t, "info", compiled.Rules["r"].Severity)
	require.Equal(t, "inherited", compiled.Rules["r"].Message)
}

func TestOpenLintOverridesAreDocumentLocal(t *testing.T) {
	rs, dir := loadOpenLintTestSet(t, map[string]string{"rules.yaml": `aliases: {Info: ['$.info']}
rules:
  r: {given: '#Info', then: {function: falsy}}
overrides:
  - files: ['legacy/**']
    aliases: {Info: ['$.elsewhere']}
    rules:
      r: error
      added: {given: '#Info', then: {function: falsy}}
  - files: ['never/**']
    rules: {}
    foreign: true
`})

	type outcome struct {
		file  string
		rules *RuleSet
		err   error
	}
	outcomes := make(chan outcome, 2)
	for _, file := range []string{"legacy/api.yaml", "api.yaml"} {
		go func(file string) {
			compiled, err := rs.ForOpenLintDocument(context.Background(), filepath.Join(dir, file), nil)
			outcomes <- outcome{file, compiled, err}
		}(file)
	}
	for range 2 {
		result := <-outcomes
		require.NoError(t, result.err)
		compiled := result.rules
		require.Equal(t, []string{"$.info"}, compiled.Rules["r"].Given)
		if result.file == "api.yaml" {
			require.Len(t, compiled.Rules, 1)
			require.Equal(t, "warn", compiled.Rules["r"].Severity)
		} else {
			require.Len(t, compiled.Rules, 2)
			require.Equal(t, "error", compiled.Rules["r"].Severity)
			require.Equal(t, []string{"$.elsewhere"}, compiled.Rules["added"].Given)
		}
	}

	require.Len(t, rs.Rules, 1)
	require.Equal(t, "warn", rs.Rules["r"].Severity)
}

func TestOpenLintMixedOverridesAreFileScoped(t *testing.T) {
	for _, patterns := range []string{
		"['*.yaml', '*.yaml#/a']",
		"['other.yaml', 'api.yaml#/a']",
	} {
		t.Run(patterns, func(t *testing.T) {
			rs, dir := loadOpenLintTestSet(t, map[string]string{"rules.yaml": fmt.Sprintf(`rules:
  disabled: {given: $, then: {function: falsy}}
overrides:
  - files: %s
    rules:
      disabled: off
      added: {given: $.b, severity: error, then: {function: falsy}}
`, patterns)})
			compiled, err := rs.ForOpenLintDocument(context.Background(), filepath.Join(dir, "api.yaml"), nil)
			require.NoError(t, err)
			require.Len(t, compiled.Rules, 1)
			require.Contains(t, compiled.Rules, "added")
			require.Equal(t, "error", compiled.Rules["added"].Severity)
			require.Equal(t, []string{"$.b"}, compiled.Rules["added"].Given)
			require.Empty(t, compiled.openLint.pointers)
		})
	}
}

func TestOpenLintMixedOverrideExtendsReplacesInheritedRules(t *testing.T) {
	rs, dir := loadOpenLintTestSet(t, map[string]string{
		"base.yaml": "rules: {base: {given: $, then: {function: falsy}}}",
		"next.yaml": "rules: {next: {given: $, then: {function: truthy}}}",
		"rules.yaml": `extends: ./base.yaml
rules: {own: {given: $, then: {function: falsy}}}
overrides:
  - files: ['other.yaml', 'api.yaml#/a']
    extends: ./next.yaml
`,
	})
	compiled, err := rs.ForOpenLintDocument(context.Background(), filepath.Join(dir, "api.yaml"), nil)
	require.NoError(t, err)
	require.Len(t, compiled.Rules, 2)
	require.Contains(t, compiled.Rules, "own")
	require.Contains(t, compiled.Rules, "next")
	require.Empty(t, compiled.openLint.pointers)
}

func TestOpenLintPointerScopesCompareSegments(t *testing.T) {
	rs, dir := loadOpenLintTestSet(t, map[string]string{"rules.yaml": `rules: {r: {given: $, then: {function: falsy}}}
overrides:
  - files: ['api.yaml#/paths/~1a']
    rules: {r: off}
`})
	compiled, err := rs.ForOpenLintDocument(context.Background(), filepath.Join(dir, "api.yaml"), nil)
	require.NoError(t, err)
	require.Equal(t, "off", compiled.OpenLintSeverity("r", []string{"paths", "/a", "get"}, "warn"))
	require.Equal(t, "warn", compiled.OpenLintSeverity("r", []string{"paths", "/ab", "get"}, "warn"))
	require.Equal(t, "warn", compiled.OpenLintSeverity("r", []string{"paths"}, "warn"))
}

func TestOpenLintAliasTargetsAndDetectionScope(t *testing.T) {
	rs, dir := loadOpenLintTestSet(t, map[string]string{"rules.yaml": `formats: [oas3]
aliases:
  Pick:
    targets:
      - formats: [oas3]
        given: [$.first]
      - formats: [oas3]
        given: [$.second]
  Other:
    targets:
      - formats: [asyncapi2]
        given: [$.third]
rules: {r: {given: '#Pick', then: {function: falsy}}}
`})
	require.Equal(t, []string{"oas3"}, rs.OpenLintFormats())
	compiled, err := rs.ForOpenLintDocument(context.Background(), filepath.Join(dir, "api.yaml"), func([]string) map[string]bool { return map[string]bool{"oas3": true} })
	require.NoError(t, err)
	require.Equal(t, []string{"$.second"}, compiled.Rules["r"].Given)
	compiled, err = rs.ForOpenLintDocument(context.Background(), filepath.Join(dir, "api.yaml"), func([]string) map[string]bool { return map[string]bool{} })
	require.NoError(t, err)
	require.Empty(t, compiled.Rules)
}

func TestOpenLintNamedFunctionsUseExistingScope(t *testing.T) {
	rs, dir := loadOpenLintTestSet(t, map[string]string{
		"rules.yaml":         "functions: [check]\nrules: {r: {given: $, then: {function: check}}}\n",
		"functions/check.js": "module.exports = async value => [{message: 'checked'}];",
	})
	compiled, err := rs.ForOpenLintDocument(context.Background(), filepath.Join(dir, "api.yaml"), nil)
	require.NoError(t, err)
	require.NotNil(t, compiled.Executable)
	scope, err := compiled.Executable.Open(context.Background(), nil)
	require.NoError(t, err)
	defer scope.Close()
	action := compiled.Rules["r"].Then.(map[string]any)
	require.Contains(t, scope.Functions(), action["function"])
	require.Contains(t, compiled.Executable.Functions(), action["function"])
}

func TestOpenLintRemoteFunctionsNeverReadLocalModules(t *testing.T) {
	for _, route := range []string{"root", "ancestor", "override"} {
		t.Run(route, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if route == "override" {
					fmt.Fprint(w, "rules: {}\noverrides:\n  - files: ['**/*.yaml']\n    functions: [check]\n    functionsDir: ../../private\n    rules: {r: {given: '$', then: {function: check}}}\n")
				} else {
					fmt.Fprint(w, "functions: [check]\nfunctionsDir: ../../private\nrules: {r: {given: '$', then: {function: check}}}\n")
				}
			}))
			defer server.Close()
			var rs *RuleSet
			var err error
			if route == "ancestor" {
				rs, err = CreateRuleSetFromDataWithContract([]byte(fmt.Sprintf("extends: '%s/rules.yaml'\nrules: {}\n", server.URL)), model.OpenLintV1Draft)
			} else {
				rs, err = DownloadRemoteRuleSetWithContract(context.Background(), server.URL+"/rules.yaml", server.Client(), model.OpenLintV1Draft)
			}
			require.NoError(t, err)
			reads := 0
			rs.readLocalFile = func(name string) ([]byte, error) {
				reads++
				return []byte("module.exports = () => [];"), nil
			}
			rs = BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSetWithHTTPClient(rs, server.Client())
			err = rs.LoadError()
			if err == nil {
				source, absErr := filepath.Abs("api.yaml")
				require.NoError(t, absErr)
				_, err = rs.ForOpenLintDocument(context.Background(), source, nil)
			}
			require.ErrorContains(t, err, "cannot load local JavaScript functions")
			require.Zero(t, reads)
		})
	}
}

func TestOpenLintOverrideFetchUsesExecutionContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	release := make(chan struct{})
	requestCancelled := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cancel()
		select {
		case <-r.Context().Done():
			close(requestCancelled)
		case <-release:
			fmt.Fprint(w, "rules: {}")
		}
	}))
	defer server.Close()
	defer close(release)
	rs, dir := loadOpenLintTestSet(t, map[string]string{"rules.yaml": fmt.Sprintf("rules: {}\noverrides:\n  - files: ['api.yaml']\n    extends: '%s/base.yaml'\n", server.URL)})
	result := make(chan error, 1)
	go func() {
		_, err := rs.ForOpenLintDocument(ctx, filepath.Join(dir, "api.yaml"), nil)
		result <- err
	}()
	select {
	case err := <-result:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("override loading ignored execution cancellation")
	}
	select {
	case <-requestCancelled:
	case <-time.After(time.Second):
		t.Fatal("override HTTP request was not cancelled")
	}
}

func TestOpenLintFunctionPreparationUsesExecutionContext(t *testing.T) {
	rs, dir := loadOpenLintTestSet(t, map[string]string{
		"rules.yaml":         "functions: [check]\nrules: {r: {given: '$', then: {function: check}}}\n",
		"functions/check.js": "while (true) {}\nmodule.exports = () => [];",
	})
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err := rs.ForOpenLintDocument(ctx, filepath.Join(dir, "api.yaml"), nil)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Less(t, time.Since(started), time.Second, "module initialization must use the execution deadline, not its five-second default")
}
