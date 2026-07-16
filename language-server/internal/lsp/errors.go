// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package lsp

import "fmt"

const (
	CodeParseError      = -32700
	CodeInvalidRequest  = -32600
	CodeMethodNotFound  = -32601
	CodeInvalidParams   = -32602
	CodeInternalError   = -32603
	CodeRequestCanceled = -32800
)

// ResponseError is a JSON-RPC error object.
type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *ResponseError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("JSON-RPC error %d: %s", e.Code, e.Message)
}

func rpcError(code int, message string, err error) *ResponseError {
	response := &ResponseError{Code: code, Message: message}
	if err != nil {
		response.Data = err.Error()
	}
	return response
}

// ExitWithoutShutdownError reports an LSP exit notification received before a
// successful shutdown request.
type ExitWithoutShutdownError struct{}

func (ExitWithoutShutdownError) Error() string {
	return "language server exited before shutdown"
}
