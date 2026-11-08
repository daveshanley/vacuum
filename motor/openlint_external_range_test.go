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
