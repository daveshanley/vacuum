// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package languageserver

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/language-server/internal/lsp"
	"github.com/daveshanley/vacuum/language-server/protocol"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestServerWireContractPreservesVacuumBehavior(t *testing.T) {
	defaultRuleSets := rulesets.BuildDefaultRuleSets()
	var traceLog bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&traceLog, &slog.HandlerOptions{Level: slog.LevelDebug}))
	state := NewServer("v-test", &utils.LintFileRequest{
		DefaultRuleSets:   defaultRuleSets,
		SelectedRS:        defaultRuleSets.GenerateOpenAPIRecommendedRuleSet(),
		Remote:            true,
		TimeoutFlag:       5,
		LookupTimeoutFlag: 500,
		Logger:            logger,
	})
	selectorContexts := make(chan DocumentContext, 4)
	state.rulesetSelector = func(document *DocumentContext) *rulesets.RuleSet {
		selectorContexts <- *document
		if strings.Contains(string(document.Content), "asyncapi:") {
			return defaultRuleSets.GenerateAsyncAPIRecommendedRuleSet()
		}
		return defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()
	}

	serverConn, clientConn := net.Pipe()
	resultCh := make(chan error, 1)
	go func() {
		resultCh <- state.server.Run(context.Background(), serverConn)
	}()
	client := newContractClient(t, clientConn)
	defer client.Close()

	client.send(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      "initialize-1",
		"method":  protocol.MethodInitialize,
		"params": map[string]any{
			"processId": nil,
			"rootUri":   nil,
			"capabilities": map[string]any{
				"workspace": map[string]any{
					"configuration": true,
					"didChangeConfiguration": map[string]any{
						"dynamicRegistration": true,
					},
				},
			},
			"initializationOptions": map[string]any{"timeout": 3},
			"workspaceFolders":      []any{},
		},
	})
	initialize := client.receive(t)
	assert.Equal(t, "initialize-1", initialize["id"])
	assert.JSONEq(t, `{
		"capabilities": {
			"textDocumentSync": 2,
			"completionProvider": {},
			"codeActionProvider": {"codeActionKinds": ["quickfix"]},
			"executeCommandProvider": {"commands": ["vacuum.openUrl"]},
			"workspace": {
				"workspaceFolders": {
					"supported": true,
					"changeNotifications": true
				}
			}
		},
		"serverInfo": {"name": "vacuum", "version": "v-test"}
	}`, mustMarshalJSON(t, initialize["result"]))
	require.NotNil(t, state.initConfig)
	require.NotNil(t, state.initConfig.Timeout)
	assert.Equal(t, 3, *state.initConfig.Timeout)

	client.send(t, map[string]any{
		"jsonrpc": "2.0",
		"method":  protocol.MethodInitialized,
		"params":  map[string]any{},
	})
	registration := client.receive(t)
	assert.Equal(t, protocol.ServerClientRegisterCapability, registration["method"])
	assert.JSONEq(t, `{
		"registrations": [{
			"id": "vacuum-workspace-configuration",
			"method": "workspace/didChangeConfiguration",
			"registerOptions": {"section": "vacuum"}
		}]
	}`, mustMarshalJSON(t, registration["params"]))
	client.respond(t, registration["id"], nil)

	client.send(t, map[string]any{
		"jsonrpc": "2.0",
		"method":  protocol.MethodSetTrace,
		"params":  map[string]any{"value": "messages"},
	})
	client.send(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      6,
		"method":  protocol.MethodTextDocumentCompletion,
		"params": map[string]any{
			"textDocument": map[string]any{"uri": "file:///tmp/trace.yaml"},
			"position":     map[string]any{"line": 0, "character": 0},
		},
	})
	_ = client.receive(t)
	assert.Contains(t, traceLog.String(), "LSP send")
	assert.Contains(t, traceLog.String(), "bytes=")

	uri := "file:///tmp/owned-lsp-openapi.yaml"
	document := "openapi: 3.1.0\ninfo:\n  title: a😀b\n  version: 1.0.0\npaths: {}\n"
	client.send(t, map[string]any{
		"jsonrpc": "2.0",
		"method":  protocol.MethodTextDocumentDidOpen,
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri": uri, "languageId": "yaml", "version": 1, "text": document,
			},
		},
	})
	client.answerWorkspaceConfiguration(t, uri)
	selectorContext := <-selectorContexts
	assert.Equal(t, uri, selectorContext.URI)
	assert.Equal(t, document, string(selectorContext.Content))
	diagnostics := client.receive(t)
	assert.Equal(t, protocol.ServerTextDocumentPublishDiagnostics, diagnostics["method"])
	openAPIParams := diagnosticsParams(t, diagnostics)
	assert.Equal(t, uri, openAPIParams["uri"])
	_, ok := openAPIParams["diagnostics"].([]any)
	require.True(t, ok)

	client.send(t, map[string]any{
		"jsonrpc": "2.0",
		"method":  protocol.MethodTextDocumentDidChange,
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri, "version": 2},
			"contentChanges": []any{
				map[string]any{
					"range": map[string]any{
						"start": map[string]any{"line": 2, "character": 12},
						"end":   map[string]any{"line": 2, "character": 13},
					},
					"text": "c",
				},
			},
		},
	})
	selectorContext = <-selectorContexts
	assert.Contains(t, string(selectorContext.Content), "title: a😀c")
	diagnostics = client.receive(t)
	assert.Equal(t, protocol.ServerTextDocumentPublishDiagnostics, diagnostics["method"])
	stored, ok := state.documentStore.Get(uri)
	require.True(t, ok)
	stored.mu.RLock()
	assert.Contains(t, stored.Content, "title: a😀c")
	stored.mu.RUnlock()

	wholeDocument := "openapi: 3.1.0\ninfo:\n  title: replaced\n  version: 1.0.0\npaths: {}\n"
	client.send(t, map[string]any{
		"jsonrpc": "2.0",
		"method":  protocol.MethodTextDocumentDidChange,
		"params": map[string]any{
			"textDocument":   map[string]any{"uri": uri, "version": 3},
			"contentChanges": []any{map[string]any{"text": wholeDocument}},
		},
	})
	selectorContext = <-selectorContexts
	assert.Equal(t, wholeDocument, string(selectorContext.Content))
	_ = client.receive(t)
	stored.mu.RLock()
	assert.Equal(t, wholeDocument, stored.Content)
	stored.mu.RUnlock()

	asyncURI := "file:///tmp/owned-lsp-asyncapi.yaml"
	client.send(t, map[string]any{
		"jsonrpc": "2.0",
		"method":  protocol.MethodTextDocumentDidOpen,
		"params": map[string]any{
			"textDocument": map[string]any{
				"uri": asyncURI, "languageId": "yaml", "version": 1,
				"text": "asyncapi: 3.0.0\ninfo:\n  title: test\n  version: 1.0.0\nchannels: {}\noperations: {}\n",
			},
		},
	})
	client.answerWorkspaceConfiguration(t, asyncURI)
	selectorContext = <-selectorContexts
	assert.Equal(t, asyncURI, selectorContext.URI)
	asyncDiagnostics := client.receive(t)
	assert.Equal(t, protocol.ServerTextDocumentPublishDiagnostics, asyncDiagnostics["method"])
	asyncParams := diagnosticsParams(t, asyncDiagnostics)
	assert.Equal(t, asyncURI, asyncParams["uri"])
	_, ok = asyncParams["diagnostics"].([]any)
	require.True(t, ok)

	client.send(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      7,
		"method":  protocol.MethodTextDocumentCodeAction,
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"range": map[string]any{
				"start": map[string]any{"line": 0, "character": 0},
				"end":   map[string]any{"line": 0, "character": 1},
			},
			"context": map[string]any{
				"diagnostics": []any{
					map[string]any{
						"range": map[string]any{
							"start": map[string]any{"line": 0, "character": 0},
							"end":   map[string]any{"line": 0, "character": 1},
						},
						"message":         "test",
						"codeDescription": map[string]any{"href": "https://quobix.com/vacuum/rules/test"},
					},
				},
			},
		},
	})
	codeActions := client.receive(t)
	actions, ok := codeActions["result"].([]any)
	require.True(t, ok)
	require.Len(t, actions, 1)
	action, ok := actions[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "View documentation", action["title"])
	assert.JSONEq(t, `{
		"title": "Open documentation",
		"command": "vacuum.openUrl",
		"arguments": ["https://quobix.com/vacuum/rules/test"]
	}`, mustMarshalJSON(t, action["command"]))

	client.send(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      8,
		"method":  protocol.MethodWorkspaceExecuteCommand,
		"params":  map[string]any{"command": "vacuum.noop"},
	})
	execute := client.receive(t)
	assert.Contains(t, execute, "result")
	assert.Nil(t, execute["result"])

	client.send(t, map[string]any{
		"jsonrpc": "2.0",
		"method":  protocol.MethodWorkspaceDidChangeConfiguration,
		"params":  map[string]any{"settings": map[string]any{"vacuum": map[string]any{"timeout": 4}}},
	})
	assert.ElementsMatch(t, []string{uri, asyncURI}, client.answerRelint(t, 2))

	workspaceURI := "file:///tmp/owned-lsp-workspace"
	client.send(t, map[string]any{
		"jsonrpc": "2.0",
		"method":  protocol.MethodWorkspaceDidChangeWorkspaceFolders,
		"params": map[string]any{
			"event": map[string]any{
				"added":   []any{map[string]any{"uri": workspaceURI, "name": "owned-lsp"}},
				"removed": []any{},
			},
		},
	})
	assert.ElementsMatch(t, []string{uri, asyncURI}, client.answerRelint(t, 2))
	state.workspaceMu.RLock()
	require.Len(t, state.workspaceFolders, 1)
	assert.Equal(t, workspaceURI, state.workspaceFolders[0].URI)
	state.workspaceMu.RUnlock()

	client.send(t, map[string]any{
		"jsonrpc": "2.0",
		"method":  protocol.MethodTextDocumentDidClose,
		"params":  map[string]any{"textDocument": map[string]any{"uri": uri}},
	})
	client.send(t, map[string]any{
		"jsonrpc": "2.0",
		"method":  protocol.MethodTextDocumentDidClose,
		"params":  map[string]any{"textDocument": map[string]any{"uri": asyncURI}},
	})
	client.send(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      9,
		"method":  protocol.MethodTextDocumentCompletion,
		"params": map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     map[string]any{"line": 0, "character": 0},
		},
	})
	_ = client.receive(t)
	_, ok = state.documentStore.Get(uri)
	assert.False(t, ok)
	_, ok = state.documentStore.Get(asyncURI)
	assert.False(t, ok)

	client.shutdown(t)
	require.NoError(t, <-resultCh)
}

type contractClient struct {
	conn   net.Conn
	reader *lsp.FrameReader
	writer *lsp.FrameWriter
}

func newContractClient(t *testing.T, connection net.Conn) *contractClient {
	t.Helper()
	require.NoError(t, connection.SetDeadline(time.Now().Add(20*time.Second)))
	return &contractClient{
		conn:   connection,
		reader: lsp.NewFrameReader(connection, 4<<20),
		writer: lsp.NewFrameWriter(connection),
	}
}

func (c *contractClient) Close() {
	_ = c.conn.Close()
}

func (c *contractClient) send(t *testing.T, value any) {
	t.Helper()
	payload, err := json.Marshal(value)
	require.NoError(t, err)
	require.NoError(t, c.writer.WriteFrame(payload))
}

func (c *contractClient) receive(t *testing.T) map[string]any {
	t.Helper()
	payload, err := c.reader.ReadFrame()
	require.NoError(t, err)
	var value map[string]any
	require.NoError(t, json.Unmarshal(payload, &value))
	return value
}

func (c *contractClient) respond(t *testing.T, id, result any) {
	t.Helper()
	c.send(t, map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
}

func (c *contractClient) answerWorkspaceConfiguration(t *testing.T, expectedURI string) {
	t.Helper()
	request := c.receive(t)
	assert.Equal(t, protocol.ServerWorkspaceConfiguration, request["method"])
	assert.JSONEq(t, `{
		"items": [{
			"scopeUri": "`+expectedURI+`",
			"section": "vacuum"
		}]
	}`, mustMarshalJSON(t, request["params"]))
	c.respond(t, request["id"], []any{nil})
}

func (c *contractClient) answerRelint(t *testing.T, documentCount int) []string {
	t.Helper()
	configurations := 0
	diagnosticURIs := make([]string, 0, documentCount)
	for configurations < documentCount || len(diagnosticURIs) < documentCount {
		message := c.receive(t)
		switch message["method"] {
		case protocol.ServerWorkspaceConfiguration:
			configurations++
			c.respond(t, message["id"], []any{nil})
		case protocol.ServerTextDocumentPublishDiagnostics:
			params := diagnosticsParams(t, message)
			uri, ok := params["uri"].(string)
			require.True(t, ok)
			diagnosticURIs = append(diagnosticURIs, uri)
		default:
			t.Fatalf("unexpected relint message: %#v", message)
		}
	}
	return diagnosticURIs
}

func (c *contractClient) shutdown(t *testing.T) {
	t.Helper()
	c.send(t, map[string]any{
		"jsonrpc": "2.0", "id": 99, "method": protocol.MethodShutdown, "params": nil,
	})
	_ = c.receive(t)
	c.send(t, map[string]any{
		"jsonrpc": "2.0", "method": protocol.MethodExit, "params": nil,
	})
}

func diagnosticsParams(t *testing.T, notification map[string]any) map[string]any {
	t.Helper()
	params, ok := notification["params"].(map[string]any)
	require.True(t, ok)
	return params
}

func mustMarshalJSON(t *testing.T, value any) string {
	t.Helper()
	payload, err := json.Marshal(value)
	require.NoError(t, err)
	return string(payload)
}
