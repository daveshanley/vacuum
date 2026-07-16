// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package lsp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
)

const (
	// DefaultMaxMessageBytes is the default maximum framed payload size.
	DefaultMaxMessageBytes = 128 << 20
	maxHeaderBytes         = 32 << 10
)

// ErrMessageTooLarge reports a payload exceeding the configured maximum.
var ErrMessageTooLarge = errors.New("LSP message exceeds configured limit")

// FrameReader reads Content-Length framed LSP messages.
type FrameReader struct {
	reader          *bufio.Reader
	maxMessageBytes int
}

// NewFrameReader creates a bounded frame reader.
func NewFrameReader(reader io.Reader, maxMessageBytes int) *FrameReader {
	if maxMessageBytes <= 0 {
		maxMessageBytes = DefaultMaxMessageBytes
	}
	return &FrameReader{
		reader:          bufio.NewReader(reader),
		maxMessageBytes: maxMessageBytes,
	}
}

// ReadFrame reads and validates one complete LSP frame.
func (r *FrameReader) ReadFrame() ([]byte, error) {
	contentLength := -1
	headerBytes := 0
	for {
		line, consumed, err := r.readHeaderLine(maxHeaderBytes - headerBytes)
		if err != nil {
			if errors.Is(err, io.EOF) && headerBytes == 0 && len(line) == 0 {
				return nil, io.EOF
			}
			return nil, fmt.Errorf("read LSP header: %w", err)
		}
		headerBytes += consumed
		if len(line) == 0 {
			break
		}
		name, value, ok := strings.Cut(string(line), ":")
		if !ok {
			return nil, fmt.Errorf("invalid LSP header %q", line)
		}
		if !strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			continue
		}
		if contentLength >= 0 {
			return nil, errors.New("duplicate LSP Content-Length header")
		}
		rawLength := strings.TrimSpace(value)
		validLength := rawLength != ""
		for index := 0; index < len(rawLength) && validLength; index++ {
			validLength = rawLength[index] >= '0' && rawLength[index] <= '9'
		}
		if !validLength {
			return nil, fmt.Errorf("invalid LSP Content-Length %q", rawLength)
		}
		parsed, parseErr := strconv.ParseUint(rawLength, 10, 64)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid LSP Content-Length %q", rawLength)
		}
		if parsed > uint64(r.maxMessageBytes) {
			return nil, fmt.Errorf("%w: %d > %d", ErrMessageTooLarge, parsed, r.maxMessageBytes)
		}
		contentLength = int(parsed)
	}
	if contentLength < 0 {
		return nil, errors.New("missing LSP Content-Length header")
	}
	payload := make([]byte, contentLength)
	if _, err := io.ReadFull(r.reader, payload); err != nil {
		return nil, fmt.Errorf("read LSP payload: %w", err)
	}
	return payload, nil
}

func (r *FrameReader) readHeaderLine(remaining int) ([]byte, int, error) {
	if remaining <= 0 {
		return nil, 0, fmt.Errorf("LSP headers exceed %d bytes", maxHeaderBytes)
	}
	var line []byte
	consumed := 0
	for {
		fragment, err := r.reader.ReadSlice('\n')
		consumed += len(fragment)
		if consumed > remaining {
			return nil, consumed, fmt.Errorf("LSP headers exceed %d bytes", maxHeaderBytes)
		}
		lineEnd := len(fragment)
		if err == nil && lineEnd > 0 && fragment[lineEnd-1] == '\n' {
			lineEnd--
			if lineEnd > 0 && fragment[lineEnd-1] == '\r' {
				lineEnd--
			}
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			line = append(line, fragment...)
			continue
		}
		if len(line) == 0 {
			line = fragment[:lineEnd]
		} else {
			line = append(line, fragment[:lineEnd]...)
		}
		if err != nil {
			return line, consumed, err
		}
		return line, consumed, nil
	}
}

// FrameWriter serializes complete LSP frame writes.
type FrameWriter struct {
	writer io.Writer
	mu     sync.Mutex
}

// NewFrameWriter creates a serialized frame writer.
func NewFrameWriter(writer io.Writer) *FrameWriter {
	return &FrameWriter{writer: writer}
}

// WriteFrame writes one complete Content-Length framed payload.
func (w *FrameWriter) WriteFrame(payload []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	header := make([]byte, 0, 32)
	header = append(header, "Content-Length: "...)
	header = strconv.AppendInt(header, int64(len(payload)), 10)
	header = append(header, '\r', '\n', '\r', '\n')
	if err := writeFull(w.writer, header); err != nil {
		return fmt.Errorf("write LSP frame header: %w", err)
	}
	if err := writeFull(w.writer, payload); err != nil {
		return fmt.Errorf("write LSP frame: %w", err)
	}
	return nil
}

func writeFull(writer io.Writer, payload []byte) error {
	for len(payload) > 0 {
		written, err := writer.Write(payload)
		if err != nil {
			return err
		}
		if written <= 0 || written > len(payload) {
			return io.ErrShortWrite
		}
		payload = payload[written:]
	}
	return nil
}
