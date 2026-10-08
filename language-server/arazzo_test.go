// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package languageserver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/language-server/protocol"
	"github.com/pb33f/testify/require"
)

func TestArazzoEditorDiagnostics(t *testing.T) {
	path, err := filepath.Abs("../model/test_files/arazzo/workflow.yaml")
	require.NoError(t, err)
	spec, err := os.ReadFile(path)
	require.NoError(t, err)
	state := newRuntimeConfigTestState()
	for _, invalid := range []bool{false, true} {
		content := string(spec)
		if invalid {
			content = strings.ReplaceAll(content, "$sourceDescriptions.api.read", "$sourceDescriptions.api.missing")
		}
		published := make(chan protocol.PublishDiagnosticsParams, 1)
		state.runDiagnostic(&Document{URI: fileURI(path), Content: content}, func(_ string, params any) { published <- params.(protocol.PublishDiagnosticsParams) })
		select {
		case got := <-published:
			require.Equal(t, fileURI(path), got.URI)
			if !invalid {
				require.Empty(t, got.Diagnostics)
				continue
			}
			require.Len(t, got.Diagnostics, 1)
			require.Equal(t, "arazzo-reference", got.Diagnostics[0].Code.Value)
			require.Equal(t, protocol.UInteger(17), got.Diagnostics[0].Range.Start.Line)
		case <-time.After(5 * time.Second):
			t.Fatal("no Arazzo diagnostics published")
		}
	}
	content := strings.ReplaceAll(string(spec), "    summary: Read a pet.\n", "")
	content = strings.ReplaceAll(content, "  summary: Read a pet.\n", "")
	published := make(chan protocol.PublishDiagnosticsParams, 1)
	state.runDiagnostic(&Document{URI: fileURI(path), Content: content}, func(_ string, params any) { published <- params.(protocol.PublishDiagnosticsParams) })
	select {
	case got := <-published:
		require.Len(t, got.Diagnostics, 2)
		for _, diagnostic := range got.Diagnostics {
			require.Equal(t, protocol.DiagnosticSeverityHint, *diagnostic.Severity)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no Arazzo hints published")
	}
}

func TestArazzoExternalDiagnosticsLinkToSource(t *testing.T) {
	dir := t.TempDir()
	rootPath := filepath.Join(dir, "main.yaml")
	sourcePath := filepath.Join(dir, "other.yaml")
	root := `arazzo: 1.1.0
info: {title: Root, version: '1', description: Root workflow., summary: Root.}
sourceDescriptions: [{name: other, url: other.yaml, type: arazzo}]
workflows:
  - workflowId: run
    description: Run the source.
    summary: Run.
    steps: [{stepId: call, description: Call., workflowId: $sourceDescriptions.other.remote}]
`
	require.NoError(t, os.WriteFile(sourcePath, []byte(`arazzo: 1.1.0
info: {title: Other, version: '1'}
sourceDescriptions: []
workflows:
  - workflowId: remote
    dependsOn: [missing]
    steps: [{stepId: call, workflowId: absent}]
`), 0600))
	state := newRuntimeConfigTestState()
	published := make(chan protocol.PublishDiagnosticsParams, 1)
	state.runDiagnostic(&Document{URI: fileURI(rootPath), Content: root}, func(_ string, params any) { published <- params.(protocol.PublishDiagnosticsParams) })
	select {
	case got := <-published:
		found := false
		for _, diagnostic := range got.Diagnostics {
			if len(diagnostic.RelatedInformation) == 0 {
				continue
			}
			found = true
			require.Equal(t, protocol.UInteger(2), diagnostic.Range.Start.Line)
			require.Equal(t, fileURI(sourcePath), diagnostic.RelatedInformation[0].Location.URI)
			require.GreaterOrEqual(t, int(diagnostic.RelatedInformation[0].Location.Range.Start.Line), 5)
		}
		require.True(t, found, "expected a linked source finding: %#v", got.Diagnostics)
	case <-time.After(5 * time.Second):
		t.Fatal("no external source diagnostics published")
	}
}
