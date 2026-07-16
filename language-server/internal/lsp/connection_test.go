// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package lsp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/language-server/protocol"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestConnectionLifecycleAndErrors(t *testing.T) {
	handler := &Handler{
		Initialize: func(_ *Context, _ *protocol.InitializeParams) (any, error) {
			return map[string]any{"capabilities": map[string]any{}}, nil
		},
		TextDocumentCompletion: func(_ *Context, _ *protocol.CompletionParams) (any, error) {
			return nil, nil
		},
	}
	client, resultCh := runPipeServer(t, handler)
	defer client.Close()

	client.send(t, map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": protocol.MethodTextDocumentCompletion, "params": map[string]any{},
	})
	response := client.receive(t)
	assert.Equal(t, float64(CodeInvalidRequest), responseErrorCode(t, response))

	client.send(t, map[string]any{
		"jsonrpc": "2.0", "id": nil, "method": protocol.MethodInitialize, "params": map[string]any{},
	})
	response = client.receive(t)
	assert.Nil(t, response["id"])
	assert.Equal(t, float64(CodeInvalidRequest), responseErrorCode(t, response))

	client.send(t, initializeRequest(2))
	response = client.receive(t)
	assert.Equal(t, float64(2), response["id"])
	assert.NotNil(t, response["result"])

	client.send(t, initializeRequest(22))
	response = client.receive(t)
	assert.Equal(t, float64(CodeInvalidRequest), responseErrorCode(t, response))

	client.send(t, map[string]any{
		"jsonrpc": "2.0", "id": "unknown", "method": "vacuum/unknown", "params": map[string]any{},
	})
	response = client.receive(t)
	assert.Equal(t, "unknown", response["id"])
	assert.Equal(t, float64(CodeMethodNotFound), responseErrorCode(t, response))

	client.send(t, map[string]any{
		"jsonrpc": "2.0", "id": 3, "method": protocol.MethodTextDocumentDidOpen, "params": []any{},
	})
	response = client.receive(t)
	assert.Equal(t, float64(CodeMethodNotFound), responseErrorCode(t, response))

	client.send(t, map[string]any{
		"jsonrpc": "2.0", "id": 4, "method": protocol.MethodShutdown, "params": nil,
	})
	response = client.receive(t)
	assert.Equal(t, float64(4), response["id"])
	assert.Contains(t, response, "result")
	assert.Nil(t, response["result"])

	client.send(t, map[string]any{
		"jsonrpc": "2.0", "id": 5, "method": protocol.MethodTextDocumentCompletion, "params": map[string]any{},
	})
	response = client.receive(t)
	assert.Equal(t, float64(CodeInvalidRequest), responseErrorCode(t, response))

	client.send(t, map[string]any{
		"jsonrpc": "2.0", "method": protocol.MethodExit, "params": nil,
	})
	require.NoError(t, <-resultCh)
}

func TestConnectionExitWithoutShutdownReturnsTypedError(t *testing.T) {
	client, resultCh := runPipeServer(t, &Handler{})
	client.send(t, map[string]any{"jsonrpc": "2.0", "method": protocol.MethodExit})
	client.Close()

	var exitErr ExitWithoutShutdownError
	assert.True(t, errors.As(<-resultCh, &exitErr))
}

func TestConnectionSupportsServerCallsAndNotifications(t *testing.T) {
	handler := &Handler{
		Initialize: func(_ *Context, _ *protocol.InitializeParams) (any, error) {
			return protocol.InitializeResult{}, nil
		},
		Initialized: func(ctx *Context, _ *protocol.InitializedParams) error {
			var result []any
			ctx.Call(protocol.ServerWorkspaceConfiguration, protocol.ConfigurationParams{
				Items: []protocol.ConfigurationItem{{}},
			}, &result)
			ctx.Notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
				URI:         "file:///api.yaml",
				Diagnostics: []protocol.Diagnostic{},
			})
			return nil
		},
	}
	client, resultCh := runPipeServer(t, handler)
	defer client.Close()
	client.send(t, initializeRequest(1))
	_ = client.receive(t)
	client.send(t, map[string]any{
		"jsonrpc": "2.0", "method": protocol.MethodInitialized, "params": map[string]any{},
	})

	request := client.receive(t)
	assert.Equal(t, protocol.ServerWorkspaceConfiguration, request["method"])
	client.send(t, map[string]any{
		"jsonrpc": "2.0", "id": request["id"], "result": []any{map[string]any{"hardMode": true}},
	})
	notification := client.receive(t)
	assert.Equal(t, protocol.ServerTextDocumentPublishDiagnostics, notification["method"])

	client.shutdown(t)
	require.NoError(t, <-resultCh)
}

func TestConnectionReadsResponsesPastInterleavedNotifications(t *testing.T) {
	closed := make(chan struct{})
	handler := &Handler{
		Initialize: func(_ *Context, _ *protocol.InitializeParams) (any, error) {
			return protocol.InitializeResult{}, nil
		},
		Initialized: func(ctx *Context, _ *protocol.InitializedParams) error {
			var result []any
			ctx.Call(protocol.ServerWorkspaceConfiguration, protocol.ConfigurationParams{
				Items: []protocol.ConfigurationItem{{}},
			}, &result)
			return nil
		},
		TextDocumentDidClose: func(_ *Context, _ *protocol.DidCloseTextDocumentParams) error {
			close(closed)
			return nil
		},
	}
	client, resultCh := runPipeServer(t, handler)
	defer client.Close()
	client.send(t, initializeRequest(1))
	_ = client.receive(t)
	client.send(t, map[string]any{
		"jsonrpc": "2.0", "method": protocol.MethodInitialized, "params": map[string]any{},
	})
	request := client.receive(t)

	client.send(t, map[string]any{
		"jsonrpc": "2.0",
		"method":  protocol.MethodTextDocumentDidClose,
		"params":  map[string]any{"textDocument": map[string]any{"uri": "file:///api.yaml"}},
	})
	client.send(t, map[string]any{
		"jsonrpc": "2.0", "id": request["id"], "result": []any{},
	})
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("interleaved notification was not dispatched")
	}

	client.shutdown(t)
	require.NoError(t, <-resultCh)
}

func TestConnectionDisconnectCancelsCallInsideHandler(t *testing.T) {
	handler := &Handler{
		Initialize: func(_ *Context, _ *protocol.InitializeParams) (any, error) {
			return protocol.InitializeResult{}, nil
		},
		Initialized: func(ctx *Context, _ *protocol.InitializedParams) error {
			var result []any
			ctx.Call(protocol.ServerWorkspaceConfiguration, protocol.ConfigurationParams{
				Items: []protocol.ConfigurationItem{{}},
			}, &result)
			return nil
		},
	}
	client, resultCh := runPipeServer(t, handler)
	client.send(t, initializeRequest(1))
	_ = client.receive(t)
	client.send(t, map[string]any{
		"jsonrpc": "2.0", "method": protocol.MethodInitialized, "params": map[string]any{},
	})
	_ = client.receive(t)
	require.NoError(t, client.Close())

	select {
	case err := <-resultCh:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("disconnect did not release handler call")
	}
}

func TestConnectionRunContextCancelsBlockedHandler(t *testing.T) {
	started := make(chan struct{})
	handler := &Handler{
		Initialize: func(_ *Context, _ *protocol.InitializeParams) (any, error) {
			return protocol.InitializeResult{}, nil
		},
		TextDocumentCompletion: func(ctx *Context, _ *protocol.CompletionParams) (any, error) {
			close(started)
			<-ctx.Context.Done()
			return nil, ctx.Context.Err()
		},
	}
	serverConn, clientConn := net.Pipe()
	runContext, cancelRun := context.WithCancel(context.Background())
	resultCh := make(chan error, 1)
	server := NewServer(handler, slog.New(slog.NewTextHandler(io.Discard, nil)))
	go func() {
		resultCh <- server.Run(runContext, serverConn)
	}()
	client := &pipeClient{
		conn:   clientConn,
		reader: NewFrameReader(clientConn, 1<<20),
		writer: NewFrameWriter(clientConn),
	}
	client.send(t, initializeRequest(1))
	_ = client.receive(t)
	client.send(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      "blocked",
		"method":  protocol.MethodTextDocumentCompletion,
		"params": map[string]any{
			"textDocument": map[string]any{"uri": "file:///api.yaml"},
			"position":     map[string]any{"line": 0, "character": 0},
		},
	})
	<-started
	cancelRun()

	select {
	case err := <-resultCh:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("Run context did not cancel blocked handler")
	}
	_ = client.Close()
}

func TestConnectionDisconnectCancelsBlockedClientRequest(t *testing.T) {
	started := make(chan struct{})
	handler := &Handler{
		Initialize: func(_ *Context, _ *protocol.InitializeParams) (any, error) {
			return protocol.InitializeResult{}, nil
		},
		TextDocumentCompletion: func(ctx *Context, _ *protocol.CompletionParams) (any, error) {
			close(started)
			<-ctx.Context.Done()
			return nil, ctx.Context.Err()
		},
	}
	client, resultCh := runPipeServer(t, handler)
	client.send(t, initializeRequest(1))
	_ = client.receive(t)
	client.send(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      "blocked",
		"method":  protocol.MethodTextDocumentCompletion,
		"params": map[string]any{
			"textDocument": map[string]any{"uri": "file:///api.yaml"},
			"position":     map[string]any{"line": 0, "character": 0},
		},
	})
	<-started
	require.NoError(t, client.Close())

	select {
	case err := <-resultCh:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("disconnect did not cancel blocked client request")
	}
}

func TestConnectionCancelsActiveClientRequest(t *testing.T) {
	started := make(chan struct{})
	handler := &Handler{
		Initialize: func(_ *Context, _ *protocol.InitializeParams) (any, error) {
			return protocol.InitializeResult{}, nil
		},
		TextDocumentCompletion: func(ctx *Context, _ *protocol.CompletionParams) (any, error) {
			close(started)
			<-ctx.Context.Done()
			return nil, ctx.Context.Err()
		},
	}
	client, resultCh := runPipeServer(t, handler)
	defer client.Close()
	client.send(t, initializeRequest(1))
	_ = client.receive(t)

	client.send(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      "completion-1",
		"method":  protocol.MethodTextDocumentCompletion,
		"params": map[string]any{
			"textDocument": map[string]any{"uri": "file:///api.yaml"},
			"position":     map[string]any{"line": 0, "character": 0},
		},
	})
	<-started
	client.send(t, map[string]any{
		"jsonrpc": "2.0",
		"method":  protocol.MethodCancelRequest,
		"params":  map[string]any{"id": "completion-1"},
	})

	response := client.receive(t)
	assert.Equal(t, "completion-1", response["id"])
	assert.Equal(t, float64(CodeRequestCanceled), responseErrorCode(t, response))

	client.shutdown(t)
	require.NoError(t, <-resultCh)
}

func TestConnectionDoesNotDispatchRequestCanceledWhileQueued(t *testing.T) {
	started := make(chan struct{})
	var calls atomic.Int32
	handler := &Handler{
		Initialize: func(_ *Context, _ *protocol.InitializeParams) (any, error) {
			return protocol.InitializeResult{}, nil
		},
		TextDocumentCompletion: func(ctx *Context, _ *protocol.CompletionParams) (any, error) {
			if calls.Add(1) == 1 {
				close(started)
				<-ctx.Context.Done()
			}
			return nil, ctx.Context.Err()
		},
	}
	client, resultCh := runPipeServer(t, handler)
	defer client.Close()
	client.send(t, initializeRequest(1))
	_ = client.receive(t)

	for _, id := range []string{"first", "queued"} {
		client.send(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      id,
			"method":  protocol.MethodTextDocumentCompletion,
			"params": map[string]any{
				"textDocument": map[string]any{"uri": "file:///api.yaml"},
				"position":     map[string]any{"line": 0, "character": 0},
			},
		})
		if id == "first" {
			<-started
		}
	}
	for _, id := range []string{"queued", "first"} {
		client.send(t, map[string]any{
			"jsonrpc": "2.0",
			"method":  protocol.MethodCancelRequest,
			"params":  map[string]any{"id": id},
		})
	}
	for range 2 {
		response := client.receive(t)
		assert.Equal(t, float64(CodeRequestCanceled), responseErrorCode(t, response))
	}
	assert.Equal(t, int32(1), calls.Load())

	client.shutdown(t)
	require.NoError(t, <-resultCh)
}

func TestConnectionOutboundCallTimeoutCleansPendingState(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	connection := NewConnection(
		serverConn,
		&Handler{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		1<<20,
	)
	resultCh := make(chan error, 1)
	go func() {
		resultCh <- connection.Run(context.Background())
	}()
	client := &pipeClient{
		conn:   clientConn,
		reader: NewFrameReader(clientConn, 1<<20),
		writer: NewFrameWriter(clientConn),
	}

	callContext, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	callResult := make(chan error, 1)
	go func() {
		callResult <- connection.Call(callContext, "vacuum/test", map[string]any{}, nil)
	}()

	request := client.receive(t)
	assert.Equal(t, "vacuum/test", request["method"])
	assert.True(t, errors.Is(<-callResult, context.DeadlineExceeded))

	connection.mu.Lock()
	assert.Empty(t, connection.pending)
	connection.mu.Unlock()
	require.NoError(t, client.Close())
	require.NoError(t, <-resultCh)
}

func TestConnectionDisconnectReleasesPendingOutboundCall(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	connection := NewConnection(
		serverConn,
		&Handler{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		1<<20,
	)
	resultCh := make(chan error, 1)
	go func() {
		resultCh <- connection.Run(context.Background())
	}()
	client := &pipeClient{
		conn:   clientConn,
		reader: NewFrameReader(clientConn, 1<<20),
		writer: NewFrameWriter(clientConn),
	}

	callResult := make(chan error, 1)
	go func() {
		callResult <- connection.Call(context.Background(), "vacuum/test", nil, nil)
	}()
	_ = client.receive(t)
	require.NoError(t, client.Close())
	assert.True(t, errors.Is(<-callResult, io.ErrClosedPipe))
	require.NoError(t, <-resultCh)
}

func TestConnectionCorrelatesConcurrentOutboundResponses(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	connection := NewConnection(
		serverConn,
		&Handler{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		1<<20,
	)
	resultCh := make(chan error, 1)
	go func() {
		resultCh <- connection.Run(context.Background())
	}()
	client := &pipeClient{
		conn:   clientConn,
		reader: NewFrameReader(clientConn, 1<<20),
		writer: NewFrameWriter(clientConn),
	}

	type callResult struct {
		method string
		value  string
		err    error
	}
	calls := make(chan callResult, 2)
	for _, method := range []string{"vacuum/first", "vacuum/second"} {
		go func(method string) {
			var result string
			err := connection.Call(context.Background(), method, nil, &result)
			calls <- callResult{method: method, value: result, err: err}
		}(method)
	}

	first := client.receive(t)
	second := client.receive(t)
	client.send(t, map[string]any{
		"jsonrpc": "2.0", "id": second["id"], "result": second["method"],
	})
	client.send(t, map[string]any{
		"jsonrpc": "2.0", "id": first["id"], "result": first["method"],
	})

	for range 2 {
		result := <-calls
		require.NoError(t, result.err)
		assert.Equal(t, result.method, result.value)
	}
	require.NoError(t, client.Close())
	require.NoError(t, <-resultCh)
}

func TestConnectionReturnsOutboundClientError(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	connection := NewConnection(
		serverConn,
		&Handler{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		1<<20,
	)
	resultCh := make(chan error, 1)
	go func() {
		resultCh <- connection.Run(context.Background())
	}()
	client := &pipeClient{
		conn:   clientConn,
		reader: NewFrameReader(clientConn, 1<<20),
		writer: NewFrameWriter(clientConn),
	}

	callResult := make(chan error, 1)
	go func() {
		callResult <- connection.Call(context.Background(), "vacuum/fail", nil, nil)
	}()
	request := client.receive(t)
	client.send(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      request["id"],
		"error":   map[string]any{"code": CodeInternalError, "message": "client failed"},
	})
	var responseErr *ResponseError
	require.ErrorAs(t, <-callResult, &responseErr)
	assert.Equal(t, CodeInternalError, responseErr.Code)

	require.NoError(t, client.Close())
	require.NoError(t, <-resultCh)
}

func TestConnectionBoundsPendingOutboundCalls(t *testing.T) {
	connection := NewConnection(
		nopReadWriteCloser{Reader: &bytes.Buffer{}, Writer: io.Discard},
		&Handler{},
		nil,
		1<<20,
	)
	for index := int64(1); index <= maxPendingCalls; index++ {
		_, err := connection.addPending(protocol.NewIntegerID(index))
		require.NoError(t, err)
	}
	_, err := connection.addPending(protocol.NewIntegerID(maxPendingCalls + 1))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too many pending LSP calls")
}

func TestConnectionBoundsQueuedMessageBytes(t *testing.T) {
	connection := NewConnection(
		nopReadWriteCloser{Reader: &bytes.Buffer{}, Writer: io.Discard},
		&Handler{},
		nil,
		10,
	)
	first := &incomingMessage{}
	second := &incomingMessage{}
	assert.True(t, connection.reserveQueuedBytes(first, 8))
	assert.False(t, connection.reserveQueuedBytes(second, 3))
	assert.Equal(t, int64(8), connection.queuedBytes.Load())
	connection.releaseQueuedBytes(first)
	assert.Equal(t, int64(0), connection.queuedBytes.Load())
}

func TestConnectionReturnsParseAndPanicErrors(t *testing.T) {
	handler := &Handler{
		Initialize: func(_ *Context, _ *protocol.InitializeParams) (any, error) {
			return protocol.InitializeResult{}, nil
		},
		TextDocumentCompletion: func(_ *Context, _ *protocol.CompletionParams) (any, error) {
			panic("boom")
		},
	}
	client, resultCh := runPipeServer(t, handler)
	defer client.Close()

	require.NoError(t, client.writer.WriteFrame([]byte(`{`)))
	response := client.receive(t)
	assert.Equal(t, float64(CodeParseError), responseErrorCode(t, response))
	assert.Nil(t, response["id"])

	client.send(t, initializeRequest(1))
	_ = client.receive(t)
	client.send(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      "invalid-params",
		"method":  protocol.MethodTextDocumentCompletion,
		"params":  []any{},
	})
	response = client.receive(t)
	errorValue, ok := response["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(CodeInvalidParams), errorValue["code"])
	assert.Contains(t, errorValue["message"], "json: cannot unmarshal array")
	assert.NotContains(t, errorValue, "data")

	client.send(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  protocol.MethodTextDocumentCompletion,
		"params": map[string]any{
			"textDocument": map[string]any{"uri": "file:///api.yaml"},
			"position":     map[string]any{"line": 0, "character": 0},
		},
	})
	response = client.receive(t)
	assert.Equal(t, float64(CodeInternalError), responseErrorCode(t, response))

	client.shutdown(t)
	require.NoError(t, <-resultCh)
}

func TestConnectionNotificationDoesNotReceiveResponse(t *testing.T) {
	handler := &Handler{
		Initialize: func(_ *Context, _ *protocol.InitializeParams) (any, error) {
			return protocol.InitializeResult{}, nil
		},
		TextDocumentDidClose: func(_ *Context, _ *protocol.DidCloseTextDocumentParams) error {
			return nil
		},
	}
	client, resultCh := runPipeServer(t, handler)
	client.send(t, initializeRequest(1))
	_ = client.receive(t)
	client.send(t, map[string]any{
		"jsonrpc": "2.0",
		"method":  protocol.MethodTextDocumentDidClose,
		"params":  map[string]any{"textDocument": map[string]any{"uri": "file:///api.yaml"}},
	})

	require.NoError(t, client.conn.SetReadDeadline(time.Now().Add(50*time.Millisecond)))
	_, err := client.reader.ReadFrame()
	require.Error(t, err)
	require.NoError(t, client.conn.SetReadDeadline(time.Time{}))

	client.shutdown(t)
	require.NoError(t, <-resultCh)
}

type pipeClient struct {
	conn   net.Conn
	reader *FrameReader
	writer *FrameWriter
}

type nopReadWriteCloser struct {
	io.Reader
	io.Writer
}

func (nopReadWriteCloser) Close() error {
	return nil
}

func (c *pipeClient) Close() error {
	return c.conn.Close()
}

func runPipeServer(t *testing.T, handler *Handler) (*pipeClient, <-chan error) {
	t.Helper()
	serverConn, clientConn := net.Pipe()
	resultCh := make(chan error, 1)
	server := NewServer(handler, slog.New(slog.NewTextHandler(io.Discard, nil)))
	go func() {
		resultCh <- server.Run(context.Background(), serverConn)
	}()
	return &pipeClient{
		conn:   clientConn,
		reader: NewFrameReader(clientConn, 1<<20),
		writer: NewFrameWriter(clientConn),
	}, resultCh
}

func (c *pipeClient) send(t *testing.T, value any) {
	t.Helper()
	payload, err := json.Marshal(value)
	require.NoError(t, err)
	require.NoError(t, c.writer.WriteFrame(payload))
}

func (c *pipeClient) receive(t *testing.T) map[string]any {
	t.Helper()
	payload, err := c.reader.ReadFrame()
	require.NoError(t, err)
	var value map[string]any
	require.NoError(t, json.Unmarshal(payload, &value))
	return value
}

func (c *pipeClient) shutdown(t *testing.T) {
	t.Helper()
	c.send(t, map[string]any{
		"jsonrpc": "2.0", "id": 99, "method": protocol.MethodShutdown, "params": nil,
	})
	_ = c.receive(t)
	c.send(t, map[string]any{
		"jsonrpc": "2.0", "method": protocol.MethodExit, "params": nil,
	})
}

func initializeRequest(id int) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  protocol.MethodInitialize,
		"params": map[string]any{
			"processId":        nil,
			"rootUri":          nil,
			"capabilities":     map[string]any{},
			"workspaceFolders": []any{},
		},
	}
}

func responseErrorCode(t *testing.T, response map[string]any) float64 {
	t.Helper()
	errorValue, ok := response["error"].(map[string]any)
	require.True(t, ok)
	code, ok := errorValue["code"].(float64)
	require.True(t, ok)
	return code
}
