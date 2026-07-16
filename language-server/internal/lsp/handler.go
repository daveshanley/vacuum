// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/daveshanley/vacuum/language-server/protocol"
)

// NotifyFunc is the compatibility callback used by Vacuum application handlers.
type NotifyFunc func(method string, params any)

// CallFunc is the compatibility callback used by Vacuum application handlers.
type CallFunc func(method string, params any, result any)

// Context contains request state and server-to-client messaging callbacks.
type Context struct {
	Context context.Context
	Method  string
	Params  json.RawMessage
	Notify  NotifyFunc
	Call    CallFunc
}

// Handler contains Vacuum's supported typed LSP callbacks and lifecycle state.
type Handler struct {
	Initialize                         func(*Context, *protocol.InitializeParams) (any, error)
	Initialized                        func(*Context, *protocol.InitializedParams) error
	SetTrace                           func(*Context, *protocol.SetTraceParams) error
	TextDocumentDidOpen                func(*Context, *protocol.DidOpenTextDocumentParams) error
	TextDocumentDidChange              func(*Context, *protocol.DidChangeTextDocumentParams) error
	TextDocumentDidClose               func(*Context, *protocol.DidCloseTextDocumentParams) error
	TextDocumentCompletion             func(*Context, *protocol.CompletionParams) (any, error)
	TextDocumentCodeAction             func(*Context, *protocol.CodeActionParams) (any, error)
	WorkspaceExecuteCommand            func(*Context, *protocol.ExecuteCommandParams) (any, error)
	WorkspaceDidChangeConfiguration    func(*Context, *protocol.DidChangeConfigurationParams) error
	WorkspaceDidChangeWorkspaceFolders func(*Context, *protocol.DidChangeWorkspaceFoldersParams) error

	mu          sync.RWMutex
	initialized bool
	shutdown    bool
}

// Handle dispatches one decoded LSP method.
func (h *Handler) Handle(ctx *Context) (any, *ResponseError, bool) {
	if ctx == nil {
		return nil, rpcError(CodeInvalidRequest, "invalid request", nil), false
	}
	if h.ShutdownReceived() && ctx.Method != protocol.MethodExit {
		return nil, rpcError(CodeInvalidRequest, "server has shut down", nil), false
	}
	if ctx.Method == protocol.MethodInitialize && h.isInitialized() {
		return nil, rpcError(CodeInvalidRequest, "server is already initialized", nil), false
	}
	if !h.isInitialized() && ctx.Method != protocol.MethodInitialize && ctx.Method != protocol.MethodExit {
		return nil, rpcError(CodeInvalidRequest, "server not initialized", nil), false
	}

	switch ctx.Method {
	case protocol.MethodInitialize:
		if h.Initialize == nil {
			return nil, methodNotFound(ctx.Method), false
		}
		var params protocol.InitializeParams
		if err := decodeParams(ctx.Params, &params); err != nil {
			return nil, invalidParams(err), false
		}
		result, err := h.Initialize(ctx, &params)
		if err != nil {
			return nil, internalError(err), false
		}
		h.setInitialized(true)
		return result, nil, false
	case protocol.MethodInitialized:
		if h.Initialized == nil {
			return nil, methodNotFound(ctx.Method), false
		}
		var params protocol.InitializedParams
		if err := decodeParams(ctx.Params, &params); err != nil {
			return nil, invalidParams(err), false
		}
		return nil, callNotification(func() error { return h.Initialized(ctx, &params) }), false
	case protocol.MethodSetTrace:
		if h.SetTrace == nil {
			return nil, methodNotFound(ctx.Method), false
		}
		var params protocol.SetTraceParams
		if err := decodeParams(ctx.Params, &params); err != nil {
			return nil, invalidParams(err), false
		}
		return nil, callNotification(func() error { return h.SetTrace(ctx, &params) }), false
	case protocol.MethodTextDocumentDidOpen:
		var params protocol.DidOpenTextDocumentParams
		if h.TextDocumentDidOpen == nil {
			return nil, methodNotFound(ctx.Method), false
		}
		if err := decodeParams(ctx.Params, &params); err != nil {
			return nil, invalidParams(err), false
		}
		return nil, callNotification(func() error { return h.TextDocumentDidOpen(ctx, &params) }), false
	case protocol.MethodTextDocumentDidChange:
		var params protocol.DidChangeTextDocumentParams
		if h.TextDocumentDidChange == nil {
			return nil, methodNotFound(ctx.Method), false
		}
		if err := decodeParams(ctx.Params, &params); err != nil {
			return nil, invalidParams(err), false
		}
		return nil, callNotification(func() error { return h.TextDocumentDidChange(ctx, &params) }), false
	case protocol.MethodTextDocumentDidClose:
		var params protocol.DidCloseTextDocumentParams
		if h.TextDocumentDidClose == nil {
			return nil, methodNotFound(ctx.Method), false
		}
		if err := decodeParams(ctx.Params, &params); err != nil {
			return nil, invalidParams(err), false
		}
		return nil, callNotification(func() error { return h.TextDocumentDidClose(ctx, &params) }), false
	case protocol.MethodTextDocumentCompletion:
		var params protocol.CompletionParams
		if h.TextDocumentCompletion == nil {
			return nil, methodNotFound(ctx.Method), false
		}
		if err := decodeParams(ctx.Params, &params); err != nil {
			return nil, invalidParams(err), false
		}
		result, err := h.TextDocumentCompletion(ctx, &params)
		if err != nil {
			return nil, internalError(err), false
		}
		return result, nil, false
	case protocol.MethodTextDocumentCodeAction:
		var params protocol.CodeActionParams
		if h.TextDocumentCodeAction == nil {
			return nil, methodNotFound(ctx.Method), false
		}
		if err := decodeParams(ctx.Params, &params); err != nil {
			return nil, invalidParams(err), false
		}
		result, err := h.TextDocumentCodeAction(ctx, &params)
		if err != nil {
			return nil, internalError(err), false
		}
		return result, nil, false
	case protocol.MethodWorkspaceExecuteCommand:
		var params protocol.ExecuteCommandParams
		if h.WorkspaceExecuteCommand == nil {
			return nil, methodNotFound(ctx.Method), false
		}
		if err := decodeParams(ctx.Params, &params); err != nil {
			return nil, invalidParams(err), false
		}
		result, err := h.WorkspaceExecuteCommand(ctx, &params)
		if err != nil {
			return nil, internalError(err), false
		}
		return result, nil, false
	case protocol.MethodWorkspaceDidChangeConfiguration:
		var params protocol.DidChangeConfigurationParams
		if h.WorkspaceDidChangeConfiguration == nil {
			return nil, methodNotFound(ctx.Method), false
		}
		if err := decodeParams(ctx.Params, &params); err != nil {
			return nil, invalidParams(err), false
		}
		return nil, callNotification(func() error {
			return h.WorkspaceDidChangeConfiguration(ctx, &params)
		}), false
	case protocol.MethodWorkspaceDidChangeWorkspaceFolders:
		var params protocol.DidChangeWorkspaceFoldersParams
		if h.WorkspaceDidChangeWorkspaceFolders == nil {
			return nil, methodNotFound(ctx.Method), false
		}
		if err := decodeParams(ctx.Params, &params); err != nil {
			return nil, invalidParams(err), false
		}
		return nil, callNotification(func() error {
			return h.WorkspaceDidChangeWorkspaceFolders(ctx, &params)
		}), false
	case protocol.MethodShutdown:
		h.mu.Lock()
		h.shutdown = true
		h.initialized = false
		h.mu.Unlock()
		return nil, nil, false
	case protocol.MethodExit:
		return nil, nil, true
	default:
		return nil, methodNotFound(ctx.Method), false
	}
}

// ShutdownReceived reports whether a successful shutdown request was handled.
func (h *Handler) ShutdownReceived() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.shutdown
}

func (h *Handler) isInitialized() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.initialized
}

func (h *Handler) setInitialized(value bool) {
	h.mu.Lock()
	h.initialized = value
	if value {
		h.shutdown = false
	}
	h.mu.Unlock()
}

func decodeParams(raw json.RawMessage, target any) error {
	if len(raw) == 0 || string(raw) == "null" {
		raw = json.RawMessage("{}")
	}
	return json.Unmarshal(raw, target)
}

func callNotification(call func() error) *ResponseError {
	if err := call(); err != nil {
		return internalError(err)
	}
	return nil
}

func methodNotFound(method string) *ResponseError {
	return rpcError(CodeMethodNotFound, fmt.Sprintf("method not supported: %s", method), nil)
}

func invalidParams(err error) *ResponseError {
	return rpcError(CodeInvalidParams, err.Error(), nil)
}

func internalError(err error) *ResponseError {
	return rpcError(CodeInternalError, "internal error", err)
}
