// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package lsp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/daveshanley/vacuum/language-server/protocol"
)

const jsonRPCVersion = "2.0"

type messageDecodeError struct {
	err   error
	parse bool
}

func (e *messageDecodeError) Error() string {
	return e.err.Error()
}

func (e *messageDecodeError) Unwrap() error {
	return e.err
}

type incomingMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	Result  json.RawMessage `json:"result"`
	Error   *ResponseError  `json:"error"`

	requestContext context.Context
	activeRequest  *activeRequest
	queuedBytes    int
}

func decodeMessage(payload []byte) (*incomingMessage, error) {
	var message incomingMessage
	if err := json.Unmarshal(payload, &message); err != nil {
		var syntaxErr *json.SyntaxError
		return nil, &messageDecodeError{err: err, parse: errors.As(err, &syntaxErr)}
	}
	if message.JSONRPC != jsonRPCVersion {
		return nil, &messageDecodeError{
			err: fmt.Errorf("unsupported JSON-RPC version %q", message.JSONRPC),
		}
	}
	if message.Method == "" && len(message.ID) == 0 {
		return nil, &messageDecodeError{
			err: fmt.Errorf("JSON-RPC message has neither method nor id"),
		}
	}
	if message.Method != "" && (len(message.Result) > 0 || message.Error != nil) {
		return nil, &messageDecodeError{
			err: fmt.Errorf("JSON-RPC request contains response fields"),
		}
	}
	if message.Method == "" && len(message.Result) == 0 && message.Error == nil {
		return nil, &messageDecodeError{
			err: fmt.Errorf("JSON-RPC response contains neither result nor error"),
		}
	}
	if message.Method == "" && len(message.Result) > 0 && message.Error != nil {
		return nil, &messageDecodeError{
			err: fmt.Errorf("JSON-RPC response contains both result and error"),
		}
	}
	return &message, nil
}

func (m *incomingMessage) requestID() (protocol.ID, bool, error) {
	if len(m.ID) == 0 {
		return protocol.ID{}, false, nil
	}
	id, err := protocol.ParseID(m.ID)
	return id, true, err
}

type requestMessage struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      protocol.ID `json:"id"`
	Method  string      `json:"method"`
	Params  any         `json:"params,omitempty"`
}

type notificationMessage struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type successResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      protocol.ID `json:"id"`
	Result  any         `json:"result"`
}

type errorResponse struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      any            `json:"id"`
	Error   *ResponseError `json:"error"`
}
