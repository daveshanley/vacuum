// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package protocol

import (
	"encoding/json"
	"testing"

	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestVacuumInitializeResultMatchesCurrentWireContract(t *testing.T) {
	version := "v0.29.10"
	quickFix := CodeActionKindQuickFix
	result := InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync:   TextDocumentSyncKindIncremental,
			CompletionProvider: &CompletionOptions{},
			CodeActionProvider: &CodeActionOptions{
				CodeActionKinds: []CodeActionKind{quickFix},
			},
			ExecuteCommandProvider: &ExecuteCommandOptions{
				Commands: []string{"vacuum.openUrl"},
			},
			Workspace: &ServerCapabilitiesWorkspace{
				WorkspaceFolders: &WorkspaceFoldersServerCapabilities{
					Supported:           boolPointer(true),
					ChangeNotifications: &BoolOrString{Value: true},
				},
			},
		},
		ServerInfo: &InitializeResultServerInfo{Name: "vacuum", Version: &version},
	}
	encoded, err := json.Marshal(result)
	require.NoError(t, err)
	assert.JSONEq(t, `{
	  "capabilities":{
	    "codeActionProvider":{"codeActionKinds":["quickfix"]},
	    "completionProvider":{},
	    "executeCommandProvider":{"commands":["vacuum.openUrl"]},
	    "textDocumentSync":2,
	    "workspace":{"workspaceFolders":{"changeNotifications":true,"supported":true}}
	  },
	  "serverInfo":{"name":"vacuum","version":"v0.29.10"}
	}`, string(encoded))
}

func boolPointer(value bool) *bool {
	return &value
}
