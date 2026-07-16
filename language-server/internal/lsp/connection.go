// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package lsp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/daveshanley/vacuum/language-server/protocol"
)

const (
	defaultCallTimeout = 75 * time.Second
	maxPendingCalls    = 1024
	maxQueuedMessages  = maxPendingCalls
)

type pendingResponse struct {
	message *incomingMessage
	err     error
}

type activeRequest struct {
	cancel context.CancelFunc
}

// Connection owns one framed JSON-RPC stream and its request state.
type Connection struct {
	stream  io.ReadWriteCloser
	reader  *FrameReader
	writer  *FrameWriter
	handler *Handler
	logger  *slog.Logger

	ctx    context.Context
	cancel context.CancelFunc

	nextID  atomic.Int64
	pending map[string]chan pendingResponse
	mu      sync.Mutex

	queuedBytes atomic.Int64
	active      map[string]*activeRequest
	activeMu    sync.Mutex
}

// NewConnection creates a connection with bounded message and pending-call state.
func NewConnection(stream io.ReadWriteCloser, handler *Handler, logger *slog.Logger, maxMessageBytes int) *Connection {
	if logger == nil {
		logger = discardLogger()
	}
	if handler == nil {
		handler = &Handler{}
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Connection{
		stream:  stream,
		reader:  NewFrameReader(stream, maxMessageBytes),
		writer:  NewFrameWriter(stream),
		handler: handler,
		logger:  logger,
		ctx:     ctx,
		cancel:  cancel,
		pending: make(map[string]chan pendingResponse),
		active:  make(map[string]*activeRequest),
	}
}

// Run serves the connection until shutdown, disconnect, cancellation or failure.
func (c *Connection) Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	runCtx, runCancel := context.WithCancel(ctx)
	defer runCancel()
	defer c.close()
	stopContextBridge := context.AfterFunc(runCtx, func() {
		c.cancel()
		_ = c.stream.Close()
	})
	defer stopContextBridge()

	incoming := make(chan *incomingMessage, maxQueuedMessages)
	readErrors := make(chan error, 1)
	go c.readLoop(runCtx, incoming, readErrors)

	for {
		select {
		case <-runCtx.Done():
			return runResult(runCtx)
		case err := <-readErrors:
			return readResult(err)
		case message := <-incoming:
			c.releaseQueuedBytes(message)
			exit, err := c.handleIncoming(message)
			if err != nil {
				if runCtx.Err() != nil {
					return runResult(runCtx)
				}
				if c.ctx.Err() != nil {
					select {
					case readErr := <-readErrors:
						return readResult(readErr)
					default:
					}
				}
				return err
			}
			if exit {
				if c.handler.ShutdownReceived() {
					return nil
				}
				return ExitWithoutShutdownError{}
			}
		}
	}
}

func runResult(ctx context.Context) error {
	if errors.Is(ctx.Err(), context.Canceled) {
		return nil
	}
	return ctx.Err()
}

func readResult(err error) error {
	if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

// Notify sends a server-to-client JSON-RPC notification.
func (c *Connection) Notify(method string, params any) error {
	return c.writeJSON(notificationMessage{
		JSONRPC: jsonRPCVersion,
		Method:  method,
		Params:  params,
	})
}

// Call sends a server-to-client request and waits for its correlated response.
func (c *Connection) Call(ctx context.Context, method string, params, result any) error {
	if ctx == nil {
		ctx = context.Background()
	}
	id := protocol.NewIntegerID(c.nextID.Add(1))
	responseCh, err := c.addPending(id)
	if err != nil {
		return err
	}
	defer c.removePending(id)

	if err := c.writeJSON(requestMessage{
		JSONRPC: jsonRPCVersion,
		ID:      id,
		Method:  method,
		Params:  params,
	}); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.ctx.Done():
		return io.ErrClosedPipe
	case response := <-responseCh:
		if response.err != nil {
			return response.err
		}
		if response.message.Error != nil {
			return response.message.Error
		}
		if result == nil || len(response.message.Result) == 0 {
			return nil
		}
		if err := json.Unmarshal(response.message.Result, result); err != nil {
			return fmt.Errorf("decode %s response: %w", method, err)
		}
		return nil
	}
}

func (c *Connection) readLoop(ctx context.Context, incoming chan<- *incomingMessage, readErrors chan<- error) {
	for {
		payload, err := c.reader.ReadFrame()
		if err != nil {
			c.reportReadError(readErrors, err)
			return
		}
		message, err := decodeMessage(payload)
		if err != nil {
			code := CodeInvalidRequest
			text := "invalid request"
			var decodeErr *messageDecodeError
			if errors.As(err, &decodeErr) && decodeErr.parse {
				code = CodeParseError
				text = "parse error"
			}
			if writeErr := c.writeError(nil, rpcError(code, text, err)); writeErr != nil {
				c.reportReadError(readErrors, writeErr)
				return
			}
			continue
		}
		c.trace("receive", payload, message.Method)
		if message.Method == "" {
			if err := c.deliverResponse(message); err != nil {
				c.logger.Debug("discarding unmatched JSON-RPC response", "error", err)
			}
			continue
		}
		id, hasID, idErr := message.requestID()
		if message.Method == protocol.MethodCancelRequest && !hasID && idErr == nil {
			c.cancelRequest(message.Params)
			continue
		}
		if hasID && idErr == nil {
			message.requestContext, message.activeRequest = c.registerActive(id)
		}
		if !c.reserveQueuedBytes(message, len(payload)) {
			c.releaseRequest(message)
			c.reportReadError(
				readErrors,
				fmt.Errorf("queued LSP messages exceed %d bytes", c.reader.maxMessageBytes),
			)
			return
		}
		select {
		case <-ctx.Done():
			c.releaseQueuedBytes(message)
			c.releaseRequest(message)
			c.reportReadError(readErrors, ctx.Err())
			return
		case incoming <- message:
		default:
			c.releaseQueuedBytes(message)
			c.releaseRequest(message)
			c.reportReadError(
				readErrors,
				fmt.Errorf("too many queued LSP messages: %d", maxQueuedMessages),
			)
			return
		}
	}
}

func (c *Connection) reserveQueuedBytes(message *incomingMessage, size int) bool {
	message.queuedBytes = size
	queued := c.queuedBytes.Add(int64(size))
	if queued <= int64(c.reader.maxMessageBytes) {
		return true
	}
	c.queuedBytes.Add(-int64(size))
	message.queuedBytes = 0
	return false
}

func (c *Connection) releaseQueuedBytes(message *incomingMessage) {
	if message == nil || message.queuedBytes == 0 {
		return
	}
	c.queuedBytes.Add(-int64(message.queuedBytes))
	message.queuedBytes = 0
}

func (c *Connection) reportReadError(readErrors chan<- error, err error) {
	c.cancel()
	select {
	case readErrors <- err:
	default:
	}
}

func (c *Connection) handleIncoming(message *incomingMessage) (bool, error) {
	defer c.releaseRequest(message)
	id, hasID, idErr := message.requestID()
	if idErr != nil {
		if hasID {
			return false, c.writeError(id, rpcError(CodeInvalidRequest, "invalid request", idErr))
		}
		return false, c.writeError(nil, rpcError(CodeInvalidRequest, "invalid request", idErr))
	}

	requestContext := message.requestContext
	if requestContext == nil {
		requestContext = c.ctx
	}
	if hasID && errors.Is(requestContext.Err(), context.Canceled) {
		return false, c.writeError(id, rpcError(CodeRequestCanceled, "request canceled", nil))
	}
	ctx := &Context{
		Context: requestContext,
		Method:  message.Method,
		Params:  message.Params,
		Notify: func(method string, params any) {
			if err := c.Notify(method, params); err != nil {
				c.logger.Warn("LSP notification failed", "method", method, "error", err)
			}
		},
		Call: func(method string, params any, result any) {
			callCtx, cancel := context.WithTimeout(c.ctx, defaultCallTimeout)
			defer cancel()
			if err := c.Call(callCtx, method, params, result); err != nil {
				c.logger.Warn("LSP client call failed", "method", method, "error", err)
			}
		},
	}

	result, responseErr, exit := c.safeHandle(ctx)
	if !hasID || exit {
		if responseErr != nil {
			c.logger.Warn("LSP notification failed", "method", message.Method, "error", responseErr)
		}
		return exit, nil
	}
	if errors.Is(requestContext.Err(), context.Canceled) {
		return false, c.writeError(id, rpcError(CodeRequestCanceled, "request canceled", nil))
	}
	if responseErr != nil {
		return false, c.writeError(id, responseErr)
	}
	return false, c.writeJSON(successResponse{
		JSONRPC: jsonRPCVersion,
		ID:      id,
		Result:  result,
	})
}

func (c *Connection) registerActive(id protocol.ID) (context.Context, *activeRequest) {
	requestContext, cancel := context.WithCancel(c.ctx)
	request := &activeRequest{cancel: cancel}
	c.activeMu.Lock()
	if previous := c.active[id.Key()]; previous != nil {
		previous.cancel()
	}
	c.active[id.Key()] = request
	c.activeMu.Unlock()
	return requestContext, request
}

func (c *Connection) releaseRequest(message *incomingMessage) {
	if message == nil || message.activeRequest == nil {
		return
	}
	id, hasID, err := message.requestID()
	if err == nil && hasID {
		c.activeMu.Lock()
		if c.active[id.Key()] == message.activeRequest {
			delete(c.active, id.Key())
		}
		c.activeMu.Unlock()
	}
	message.activeRequest.cancel()
	message.activeRequest = nil
}

func (c *Connection) cancelRequest(raw json.RawMessage) {
	var params protocol.CancelParams
	if err := decodeParams(raw, &params); err != nil {
		c.logger.Debug("discarding invalid cancellation notification", "error", err)
		return
	}
	c.activeMu.Lock()
	request := c.active[params.ID.Key()]
	c.activeMu.Unlock()
	if request != nil {
		request.cancel()
	}
}

func (c *Connection) safeHandle(ctx *Context) (result any, responseErr *ResponseError, exit bool) {
	defer func() {
		if recovered := recover(); recovered != nil {
			responseErr = rpcError(CodeInternalError, "internal error", fmt.Errorf("panic: %v", recovered))
		}
	}()
	return c.handler.Handle(ctx)
}

func (c *Connection) addPending(id protocol.ID) (chan pendingResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.pending) >= maxPendingCalls {
		return nil, fmt.Errorf("too many pending LSP calls: %d", len(c.pending))
	}
	responseCh := make(chan pendingResponse, 1)
	c.pending[id.Key()] = responseCh
	return responseCh, nil
}

func (c *Connection) removePending(id protocol.ID) {
	c.mu.Lock()
	delete(c.pending, id.Key())
	c.mu.Unlock()
}

func (c *Connection) deliverResponse(message *incomingMessage) error {
	id, hasID, err := message.requestID()
	if err != nil || !hasID {
		return fmt.Errorf("response has invalid id: %w", err)
	}
	c.mu.Lock()
	responseCh := c.pending[id.Key()]
	c.mu.Unlock()
	if responseCh == nil {
		return fmt.Errorf("response id %s is not pending", id.Key())
	}
	select {
	case responseCh <- pendingResponse{message: message}:
		return nil
	default:
		return fmt.Errorf("response id %s was delivered more than once", id.Key())
	}
}

func (c *Connection) writeError(id any, responseErr *ResponseError) error {
	return c.writeJSON(errorResponse{
		JSONRPC: jsonRPCVersion,
		ID:      id,
		Error:   responseErr,
	})
}

func (c *Connection) writeJSON(value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode JSON-RPC message: %w", err)
	}
	method := ""
	switch message := value.(type) {
	case requestMessage:
		method = message.Method
	case notificationMessage:
		method = message.Method
	}
	c.trace("send", payload, method)
	return c.writer.WriteFrame(payload)
}

func (c *Connection) trace(direction string, payload []byte, method string) {
	switch {
	case protocol.HasTraceLevel(protocol.TraceValueVerbose):
		c.logger.Debug("LSP "+direction, "method", method, "payload", string(payload))
	case protocol.HasTraceLevel(protocol.TraceValueMessage):
		c.logger.Debug("LSP "+direction, "method", method, "bytes", len(payload))
	}
}

func (c *Connection) close() {
	c.cancel()
	c.activeMu.Lock()
	for id, request := range c.active {
		request.cancel()
		delete(c.active, id)
	}
	c.activeMu.Unlock()
	c.mu.Lock()
	for id, responseCh := range c.pending {
		select {
		case responseCh <- pendingResponse{err: io.ErrClosedPipe}:
		default:
		}
		delete(c.pending, id)
	}
	c.mu.Unlock()
	_ = c.stream.Close()
}
