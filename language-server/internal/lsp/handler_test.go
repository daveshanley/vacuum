// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package lsp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/daveshanley/vacuum/language-server/protocol"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestHandlerDispatchesEveryVacuumMethod(t *testing.T) {
	called := make(map[string]int)
	record := func(method string) {
		called[method]++
	}
	handler := &Handler{
		Initialize: func(_ *Context, _ *protocol.InitializeParams) (any, error) {
			record(protocol.MethodInitialize)
			return protocol.InitializeResult{}, nil
		},
		Initialized: func(_ *Context, _ *protocol.InitializedParams) error {
			record(protocol.MethodInitialized)
			return nil
		},
		SetTrace: func(_ *Context, _ *protocol.SetTraceParams) error {
			record(protocol.MethodSetTrace)
			return nil
		},
		TextDocumentDidOpen: func(_ *Context, _ *protocol.DidOpenTextDocumentParams) error {
			record(protocol.MethodTextDocumentDidOpen)
			return nil
		},
		TextDocumentDidChange: func(_ *Context, _ *protocol.DidChangeTextDocumentParams) error {
			record(protocol.MethodTextDocumentDidChange)
			return nil
		},
		TextDocumentDidClose: func(_ *Context, _ *protocol.DidCloseTextDocumentParams) error {
			record(protocol.MethodTextDocumentDidClose)
			return nil
		},
		TextDocumentCompletion: func(_ *Context, _ *protocol.CompletionParams) (any, error) {
			record(protocol.MethodTextDocumentCompletion)
			return nil, nil
		},
		TextDocumentCodeAction: func(_ *Context, _ *protocol.CodeActionParams) (any, error) {
			record(protocol.MethodTextDocumentCodeAction)
			return []protocol.CodeAction{}, nil
		},
		WorkspaceExecuteCommand: func(_ *Context, _ *protocol.ExecuteCommandParams) (any, error) {
			record(protocol.MethodWorkspaceExecuteCommand)
			return nil, nil
		},
		WorkspaceDidChangeConfiguration: func(_ *Context, _ *protocol.DidChangeConfigurationParams) error {
			record(protocol.MethodWorkspaceDidChangeConfiguration)
			return nil
		},
		WorkspaceDidChangeWorkspaceFolders: func(_ *Context, _ *protocol.DidChangeWorkspaceFoldersParams) error {
			record(protocol.MethodWorkspaceDidChangeWorkspaceFolders)
			return nil
		},
	}

	cases := []struct {
		method protocol.Method
		params string
	}{
		{protocol.MethodInitialize, `{"processId":null,"rootUri":null,"capabilities":{}}`},
		{protocol.MethodInitialized, `{}`},
		{protocol.MethodSetTrace, `{"value":"messages"}`},
		{protocol.MethodTextDocumentDidOpen, `{"textDocument":{"uri":"file:///api.yaml","languageId":"yaml","version":1,"text":""}}`},
		{protocol.MethodTextDocumentDidChange, `{"textDocument":{"uri":"file:///api.yaml","version":2},"contentChanges":[{"text":"next"}]}`},
		{protocol.MethodTextDocumentDidClose, `{"textDocument":{"uri":"file:///api.yaml"}}`},
		{protocol.MethodTextDocumentCompletion, `{"textDocument":{"uri":"file:///api.yaml"},"position":{"line":0,"character":0}}`},
		{protocol.MethodTextDocumentCodeAction, `{"textDocument":{"uri":"file:///api.yaml"},"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":1}},"context":{"diagnostics":[]}}`},
		{protocol.MethodWorkspaceExecuteCommand, `{"command":"vacuum.noop"}`},
		{protocol.MethodWorkspaceDidChangeConfiguration, `{"settings":{}}`},
		{protocol.MethodWorkspaceDidChangeWorkspaceFolders, `{"event":{"added":[],"removed":[]}}`},
	}
	for _, testCase := range cases {
		result, responseErr, exit := handler.Handle(&Context{
			Context: context.Background(),
			Method:  testCase.method,
			Params:  json.RawMessage(testCase.params),
		})
		assert.False(t, exit)
		require.Nil(t, responseErr)
		_ = result
	}

	for _, testCase := range cases {
		assert.Equal(t, 1, called[string(testCase.method)])
	}
}

func TestHandlerDistinguishesInvalidParamsAndUnknownMethods(t *testing.T) {
	handler := &Handler{
		Initialize: func(_ *Context, _ *protocol.InitializeParams) (any, error) {
			return protocol.InitializeResult{}, nil
		},
		TextDocumentDidOpen: func(_ *Context, _ *protocol.DidOpenTextDocumentParams) error {
			return nil
		},
	}
	_, responseErr, _ := handler.Handle(&Context{
		Context: context.Background(),
		Method:  protocol.MethodInitialize,
		Params:  json.RawMessage(`{"processId":null,"rootUri":null,"capabilities":{}}`),
	})
	require.Nil(t, responseErr)

	_, responseErr, _ = handler.Handle(&Context{
		Context: context.Background(),
		Method:  protocol.MethodTextDocumentDidOpen,
		Params:  json.RawMessage(`[]`),
	})
	require.NotNil(t, responseErr)
	assert.Equal(t, CodeInvalidParams, responseErr.Code)
	assert.Contains(t, responseErr.Message, "json: cannot unmarshal array")
	assert.Nil(t, responseErr.Data)

	_, responseErr, _ = handler.Handle(&Context{
		Context: context.Background(),
		Method:  "vacuum/unknown",
		Params:  json.RawMessage(`{}`),
	})
	require.NotNil(t, responseErr)
	assert.Equal(t, CodeMethodNotFound, responseErr.Code)
}

func TestHandlerIgnoresShutdownNotifications(t *testing.T) {
	handler := &Handler{}
	handler.setInitialized(true)

	_, responseErr, exit := handler.Handle(&Context{
		Context: context.Background(),
		Method:  protocol.MethodShutdown,
		Params:  json.RawMessage(`{}`),
	})

	assert.False(t, exit)
	assert.Nil(t, responseErr)
	assert.False(t, handler.ShutdownReceived())
}

func BenchmarkHandlerDispatch(b *testing.B) {
	handler := &Handler{
		Initialize: func(_ *Context, _ *protocol.InitializeParams) (any, error) {
			return protocol.InitializeResult{}, nil
		},
		TextDocumentCompletion: func(_ *Context, _ *protocol.CompletionParams) (any, error) {
			return nil, nil
		},
	}
	_, responseErr, _ := handler.Handle(&Context{
		Context: context.Background(),
		Method:  protocol.MethodInitialize,
		Params:  json.RawMessage(`{"processId":null,"rootUri":null,"capabilities":{}}`),
	})
	if responseErr != nil {
		b.Fatal(responseErr)
	}
	params := json.RawMessage(
		`{"textDocument":{"uri":"file:///api.yaml"},"position":{"line":0,"character":0}}`,
	)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, responseErr, _ := handler.Handle(&Context{
			Context: context.Background(),
			Method:  protocol.MethodTextDocumentCompletion,
			Params:  params,
		})
		if responseErr != nil {
			b.Fatal(responseErr)
		}
	}
}
