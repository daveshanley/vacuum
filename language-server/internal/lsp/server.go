// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package lsp

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
)

// Server creates one owned JSON-RPC connection per run.
type Server struct {
	handler         *Handler
	logger          *slog.Logger
	maxMessageBytes int
}

// NewServer creates a stdio-capable server for the supplied handler.
func NewServer(handler *Handler, logger *slog.Logger) *Server {
	if logger == nil {
		logger = discardLogger()
	}
	return &Server{
		handler:         handler,
		logger:          logger,
		maxMessageBytes: DefaultMaxMessageBytes,
	}
}

// Run serves an injected stream until the LSP session ends.
func (s *Server) Run(ctx context.Context, stream io.ReadWriteCloser) error {
	if stream == nil {
		return fmt.Errorf("LSP stream is nil")
	}
	connection := NewConnection(stream, s.handler, s.logger, s.maxMessageBytes)
	return connection.Run(ctx)
}

// RunStdio serves the process standard input and output streams.
func (s *Server) RunStdio(ctx context.Context) error {
	return s.Run(ctx, stdioStream{})
}

type stdioStream struct{}

func (stdioStream) Read(p []byte) (int, error) {
	return os.Stdin.Read(p)
}

func (stdioStream) Write(p []byte) (int, error) {
	return os.Stdout.Write(p)
}

func (stdioStream) Close() error {
	return nil
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
