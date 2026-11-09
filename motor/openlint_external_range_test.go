// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package motor

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/model/reports"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/testify/require"
)

func TestOpenLintExternalSourceLocations(t *testing.T) {
	const external = "# external header\n# more\n# more\ndefs:\n  X:\n    title: \"😀abcdefghijklmnopqrstuvwxyz\"\n"
	for _, remote := range []bool{false, true} {
		t.Run(fmt.Sprintf("remote=%t", remote), func(t *testing.T) {
			dir := t.TempDir()
			source := filepath.Join(dir, "external.yaml")
			reference := "./external.yaml"
			var requests atomic.Int32
			if remote {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					w.Header().Set("Content-Type", "application/yaml")
					_, _ = io.WriteString(w, external)
				}))
				defer server.Close()
				source = server.URL + "/external.yaml"
				reference = source
			} else {
				require.NoError(t, os.WriteFile(source, []byte(external), 0600))
			}
			document := []byte(fmt.Sprintf("item:\n  $ref: %s#/defs/X\n", reference))
			raw, err := rulesets.CreateRuleSetFromData([]byte(`rules:
  r: {given: $.item.title, then: {function: undefined}}
  s: {given: $.item.title, then: {function: undefined}}
`), model.OpenLintV1Draft)
			require.NoError(t, err)
			rs := rulesets.BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(raw)
			result := ApplyRulesToRuleSetWithOptions(&RuleSetExecution{RuleSet: rs, Spec: document, SpecFileName: filepath.Join(dir, "root.yaml"), Base: dir, AllowLookup: true, SilenceLogs: true}, &ExecutionOptions{Contract: model.OpenLintV1Draft})
			defer result.ReleaseOwnedResources()
			require.Empty(t, result.Errors)
			report := model.NewRuleResultSet(result.Results).GenerateSpectralReport(filepath.Join(dir, "root.yaml"))
			require.Len(t, report, 2)
			for _, finding := range report {
				require.Equal(t, source, finding.Source)
				require.Equal(t, []string{"defs", "X", "title"}, finding.Path)
				require.Equal(t, reports.RangeItem{Line: 5, Char: 11}, finding.Range.Start)
				require.Equal(t, reports.RangeItem{Line: 5, Char: 41}, finding.Range.End)
			}
			// Position recovery must use cached source bytes after the file is removed,
			// and it must not issue any more remote requests for repeated findings.
			if !remote {
				require.NoError(t, os.Remove(source))
			}
			before := requests.Load()
			original, _, valid := parseOpenLintDocument(document)
			require.True(t, valid)
			locations := newOpenLintSourceLocations(original, filepath.Join(dir, "root.yaml"))
			for i := range result.Results {
				finding := result.Results[i]
				require.NoError(t, locations.setResultLocation(&finding))
				require.Equal(t, "$.defs.X.title", finding.Path)
				require.Equal(t, reports.RangeItem{Line: 5, Char: 41}, finding.Range.End)
			}
			require.Len(t, locations.external, 1)
			require.Equal(t, before, requests.Load())
		})
	}
}

func TestBuildRolodexRemoteFilesOwnSourceMetadata(t *testing.T) {
	documents := map[string]string{"/first.yaml": "type: string\n", "/second.yaml": "type: integer\n"}
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = io.WriteString(w, documents[r.URL.Path])
	}))
	defer server.Close()
	source := []byte(fmt.Sprintf("first: {$ref: '%s/first.yaml'}\nsecond: {$ref: '%s/second.yaml'}\n", server.URL, server.URL))
	info, _, valid := parseOpenLintDocument(source)
	require.True(t, valid)
	config := index.CreateOpenAPIIndexConfig()
	config.AllowFileLookup = false
	config.SpecInfo = info
	rolodex, err := BuildRolodexFromIndexConfig(config, nil)
	require.NoError(t, err)
	defer rolodex.Release()
	rolodex.SetRootNode(info.RootNode)
	require.NoError(t, rolodex.IndexTheRolodex(context.Background()))
	require.Same(t, info, config.SpecInfo, "remote setup must preserve the caller's root metadata")
	require.Len(t, rolodex.GetIndexes(), 2)
	var firstInfo *datamodel.SpecInfo
	for _, idx := range rolodex.GetIndexes() {
		owned := idx.GetConfig().SpecInfo
		require.NotNil(t, owned)
		require.NotSame(t, info, owned)
		if firstInfo == nil {
			firstInfo = owned
		} else {
			require.NotSame(t, firstInfo, owned)
		}
		require.NotNil(t, owned.SpecBytes)
		require.Equal(t, documents[idx.GetSpecAbsolutePath()[len(server.URL):]], string(*owned.SpecBytes))
	}
	require.Equal(t, int32(2), requests.Load(), "each remote source is loaded once")
}

func TestOpenLintResolvedLocalSourcePaths(t *testing.T) {
	for _, targetFirst := range []bool{true, false} {
		for _, tc := range []struct{ name, given, action, want string }{
			{"scalar", "$.Use", "{field: description, function: length, functionOptions: {max: 1}}", "$.Target.description"},
			{"container", "$.Use", "{function: falsy}", "$.Target"},
			{"missing", "$.Use", "{field: missing, function: defined}", "$.Target"},
			{"repeated", "$[\"Use\",\"UseAgain\"]", "{field: description, function: length, functionOptions: {max: 1}}", "$.Target.description"},
			{"empty", "$.EmptyUse", "{function: falsy}", "$.Empty"},
			{"chain-container", "$.Chain", "{function: falsy}", "$.Target"},
			{"chain", "$.Chain", "{field: description, function: length, functionOptions: {max: 1}}", "$.Target.description"},
			{"repeated-container", "$[\"Use\",\"UseAgain\"]", "{function: falsy}", "$.Target"},
		} {
			t.Run(fmt.Sprintf("%s/targetFirst=%t", tc.name, targetFirst), func(t *testing.T) {
				target := "Target:\n  type: object\n  description: long\n"
				uses := "Use: {$ref: '#/Target'}\nUseAgain: {$ref: '#/Target'}\nChain: {$ref: '#/Use'}\nSibling:\n  $ref: '#/Target'\n  description: local\nEmpty: {}\nEmptyUse: {$ref: '#/Empty'}\n"
				document := target + uses
				if !targetFirst {
					document = uses + target
				}
				raw, err := rulesets.CreateRuleSetFromData([]byte(fmt.Sprintf("rules:\n  r: {given: '%s', then: %s}\n", tc.given, tc.action)), model.OpenLintV1Draft)
				require.NoError(t, err)
				rs := rulesets.BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(raw)
				result := ApplyRulesToRuleSetWithOptions(&RuleSetExecution{RuleSet: rs, Spec: []byte(document), SilenceLogs: true}, &ExecutionOptions{Contract: model.OpenLintV1Draft})
				defer result.ReleaseOwnedResources()
				require.Empty(t, result.Errors)
				require.Len(t, result.Results, 1)
				finding := result.Results[0]
				require.Equal(t, tc.want, finding.Path)
				for _, path := range finding.Paths {
					require.Equal(t, tc.want, path)
				}
				offset := strings.Index(document, "long")
				if tc.name == "container" || tc.name == "missing" || tc.name == "repeated-container" || tc.name == "chain-container" {
					offset = strings.Index(document, "type:")
				}
				if tc.name == "empty" {
					offset = strings.Index(document, "{}")
				}
				prefix := document[:offset]
				require.Equal(t, strings.Count(prefix, "\n"), finding.Range.Start.Line)
				require.Equal(t, offset-strings.LastIndex(prefix, "\n")-1, finding.Range.Start.Char)
			})
		}
	}
}

func TestOpenLintSourcePointerOverride(t *testing.T) {
	for _, scope := range []string{"Target", "Use"} {
		t.Run(scope, func(t *testing.T) {
			raw, err := rulesets.CreateRuleSetFromData([]byte(fmt.Sprintf(`rules:
  r: {given: $.Use, then: {field: description, function: length, functionOptions: {max: 1}}}
overrides:
  - files: ['**/api.yaml#/%s']
    rules: {r: off}
`, scope)), model.OpenLintV1Draft)
			require.NoError(t, err)
			rs := rulesets.BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(raw)
			result := ApplyRulesToRuleSetWithOptions(&RuleSetExecution{RuleSet: rs, Spec: []byte("Target: {description: long}\nUse: {$ref: '#/Target'}\n"), SpecFileName: filepath.Join(t.TempDir(), "api.yaml"), SilenceLogs: true}, &ExecutionOptions{Contract: model.OpenLintV1Draft})
			defer result.ReleaseOwnedResources()
			require.Empty(t, result.Errors)
			if scope == "Target" {
				require.Empty(t, result.Results)
			} else {
				require.Len(t, result.Results, 1)
				require.Equal(t, "$.Target.description", result.Results[0].Path)
			}
		})
	}
}

func TestOpenLintUnresolvedSiblingRetainsSource(t *testing.T) {
	raw, err := rulesets.CreateRuleSetFromData([]byte(`rules:
  r: {resolved: false, given: $.Use.description, then: {function: falsy}}
`), model.OpenLintV1Draft)
	require.NoError(t, err)
	rs := rulesets.BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(raw)
	result := ApplyRulesToRuleSetWithOptions(&RuleSetExecution{RuleSet: rs, Spec: []byte("Target: {description: target}\nUse: {$ref: '#/Target', description: local}\n"), SilenceLogs: true}, &ExecutionOptions{Contract: model.OpenLintV1Draft})
	defer result.ReleaseOwnedResources()
	require.Empty(t, result.Errors)
	require.Len(t, result.Results, 1)
	require.Equal(t, "$.Use.description", result.Results[0].Path)
	require.Equal(t, "local", result.Results[0].StartNode.Value)
}

func TestOpenLintResolvedContainersKeepSourceOwnership(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "external.yaml")
	require.NoError(t, os.WriteFile(source, []byte("X: {description: external}\n"), 0600))
	raw, err := rulesets.CreateRuleSetFromData([]byte("rules:\n  r: {given: '$.*', then: {function: falsy}}\n"), model.OpenLintV1Draft)
	require.NoError(t, err)
	rs := rulesets.BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(raw)
	root := filepath.Join(dir, "root.yaml")
	result := ApplyRulesToRuleSetWithOptions(&RuleSetExecution{RuleSet: rs, Spec: []byte("Local: {description: local}\nUse: {$ref: './external.yaml#/X'}\nUseAgain: {$ref: './external.yaml#/X'}\n"), SpecFileName: root, Base: dir, AllowLookup: true, SilenceLogs: true}, &ExecutionOptions{Contract: model.OpenLintV1Draft})
	defer result.ReleaseOwnedResources()
	require.Empty(t, result.Errors)
	report := model.NewRuleResultSet(result.Results).GenerateSpectralReport(root)
	require.Len(t, report, 2)
	for _, finding := range report {
		if finding.Source == source {
			require.Equal(t, []string{"X"}, finding.Path)
			require.Equal(t, reports.RangeItem{Line: 0, Char: 3}, finding.Range.Start)
			require.Equal(t, reports.RangeItem{Line: 0, Char: 26}, finding.Range.End)
		} else {
			require.Equal(t, root, finding.Source)
			require.Equal(t, []string{"Local"}, finding.Path)
			require.Equal(t, reports.RangeItem{Line: 0, Char: 7}, finding.Range.Start)
		}
	}
}

func TestOpenLintContainerResolutionOptions(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		resolved, force, skip bool
		want                  string
	}{
		{"unresolved", false, false, false, "$.Use"},
		{"forced", false, true, false, "$.Target"},
		{"skipped", true, false, true, "$.Use"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := rulesets.CreateRuleSetFromData([]byte(fmt.Sprintf("rules:\n  r: {resolved: %t, given: $.Use, then: {function: falsy}}\n", tc.resolved)), model.OpenLintV1Draft)
			require.NoError(t, err)
			rs := rulesets.BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(raw)
			result := ApplyRulesToRuleSetWithOptions(&RuleSetExecution{RuleSet: rs, Spec: []byte("Target: {}\nUse: {$ref: '#/Target'}\n"), SkipResolve: tc.skip, SilenceLogs: true}, &ExecutionOptions{Contract: model.OpenLintV1Draft, ResolveAllRefs: tc.force})
			defer result.ReleaseOwnedResources()
			require.Empty(t, result.Errors)
			require.Len(t, result.Results, 1)
			require.Equal(t, tc.want, result.Results[0].Path)
		})
	}
}

func TestOpenLintDirectAndReferencedContainersCollapse(t *testing.T) {
	for _, relative := range []bool{false, true} {
		for _, external := range []bool{false, true} {
			t.Run(fmt.Sprintf("external=%t/relative=%t", external, relative), func(t *testing.T) {
				dir := t.TempDir()
				raw, err := rulesets.CreateRuleSetFromData([]byte("rules:\n  r: {given: '$.*', then: {function: falsy}}\n"), model.OpenLintV1Draft)
				require.NoError(t, err)
				rs := rulesets.BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(raw)
				document := "Target: {}\nUse: {$ref: '#/Target'}\nUseAgain: {$ref: '#/Target'}\nChain: {$ref: '#/Use'}\n"
				if external {
					require.NoError(t, os.WriteFile(filepath.Join(dir, "external.yaml"), []byte("X: {}\nBack: {$ref: './api.yaml#/Target'}\n"), 0600))
					document += "External: {$ref: './external.yaml#/X'}\nReturn: {$ref: './external.yaml#/Back'}\n"
				}
				filename := filepath.Join(dir, "api.yaml")
				require.NoError(t, os.WriteFile(filename, []byte(document), 0600))
				if relative {
					t.Chdir(dir)
					filename = "api.yaml"
				}
				result := ApplyRulesToRuleSetWithOptions(&RuleSetExecution{RuleSet: rs, Spec: []byte(document), SpecFileName: filename, Base: dir, AllowLookup: true, SilenceLogs: true}, &ExecutionOptions{Contract: model.OpenLintV1Draft})
				defer result.ReleaseOwnedResources()
				require.Empty(t, result.Errors)
				expected := 1
				if external {
					expected++
				}
				require.Len(t, result.Results, expected)
				count := 0
				for _, finding := range result.Results {
					if finding.Path == "$.Target" {
						count++
					}
				}
				require.Equal(t, 1, count)
			})
		}
	}
}
