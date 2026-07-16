// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package lsp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestFrameReaderHandlesPartialAndConsecutiveFrames(t *testing.T) {
	first := `{"jsonrpc":"2.0","method":"initialized","params":{}}`
	second := `{"jsonrpc":"2.0","method":"exit"}`
	wire := fmt.Sprintf(
		"content-length: %d\r\nX-Test: value\r\n\r\n%sContent-Length: %d\r\n\r\n%s",
		len(first), first, len(second), second,
	)
	reader := NewFrameReader(&chunkReader{reader: strings.NewReader(wire), size: 3}, 1024)

	payload, err := reader.ReadFrame()
	require.NoError(t, err)
	assert.Equal(t, first, string(payload))
	payload, err = reader.ReadFrame()
	require.NoError(t, err)
	assert.Equal(t, second, string(payload))
}

func TestFrameReaderRejectsInvalidHeadersAndLengths(t *testing.T) {
	tests := []struct {
		name string
		wire string
		max  int
	}{
		{name: "missing length", wire: "X-Test: true\r\n\r\n"},
		{name: "duplicate length", wire: "Content-Length: 1\r\nContent-Length: 1\r\n\r\nx"},
		{name: "invalid header", wire: "broken\r\n\r\n"},
		{name: "negative length", wire: "Content-Length: -1\r\n\r\n"},
		{name: "signed length", wire: "Content-Length: +1\r\n\r\nx"},
		{name: "invalid length", wire: "Content-Length: nope\r\n\r\n"},
		{name: "overflowing length", wire: "Content-Length: 999999999999999999999999\r\n\r\n"},
		{name: "too large", wire: "Content-Length: 10\r\n\r\n0123456789", max: 5},
		{name: "missing separator", wire: "Content-Length: 1\r\nx"},
		{name: "unterminated first header", wire: "Content-Length: 1"},
		{name: "oversized header", wire: strings.Repeat("X", maxHeaderBytes+1)},
		{name: "short body", wire: "Content-Length: 4\r\n\r\nabc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewFrameReader(strings.NewReader(tt.wire), tt.max).ReadFrame()
			require.Error(t, err)
		})
	}
}

func TestFrameReaderAcceptsZeroLengthBody(t *testing.T) {
	payload, err := NewFrameReader(strings.NewReader("Content-Length: 0\r\n\r\n"), 1).ReadFrame()
	require.NoError(t, err)
	assert.Empty(t, payload)
}

func TestFrameReaderCountsCRLFHeaderBytesExactly(t *testing.T) {
	padding := strings.Repeat("x", maxHeaderBytes-len("X: \r\n\r\n"))
	wire := "X: " + padding + "\r\n\r\n"
	_, err := NewFrameReader(strings.NewReader(wire), 1).ReadFrame()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing LSP Content-Length")

	wire = "X: " + padding + "x\r\n\r\n"
	_, err = NewFrameReader(strings.NewReader(wire), 1).ReadFrame()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "headers exceed")
}

func TestFrameReaderAllocatesOnlyConfiguredMaximumBody(t *testing.T) {
	const maximum = 1 << 20
	body := bytes.Repeat([]byte("x"), maximum)
	var wire bytes.Buffer
	require.NoError(t, NewFrameWriter(&wire).WriteFrame(body))

	payload, err := NewFrameReader(&wire, maximum).ReadFrame()
	require.NoError(t, err)
	assert.Len(t, payload, maximum)
	assert.Equal(t, maximum, cap(payload))
}

func TestFrameWriterHandlesConcurrentWritesWithoutInterleaving(t *testing.T) {
	var output bytes.Buffer
	writer := NewFrameWriter(&output)
	const count = 40
	var wg sync.WaitGroup
	errorsCh := make(chan error, count)
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			errorsCh <- writer.WriteFrame([]byte(fmt.Sprintf(`{"index":%d}`, index)))
		}(i)
	}
	wg.Wait()
	close(errorsCh)
	for err := range errorsCh {
		require.NoError(t, err)
	}

	reader := NewFrameReader(&output, 1024)
	seen := make(map[string]struct{}, count)
	for i := 0; i < count; i++ {
		payload, err := reader.ReadFrame()
		require.NoError(t, err)
		seen[string(payload)] = struct{}{}
	}
	assert.Len(t, seen, count)
}

func TestFrameWriterReportsShortWrites(t *testing.T) {
	err := NewFrameWriter(shortWriter{}).WriteFrame([]byte(`{}`))
	require.Error(t, err)
	assert.True(t, errors.Is(err, io.ErrShortWrite))
}

type chunkReader struct {
	reader io.Reader
	size   int
}

func (r *chunkReader) Read(p []byte) (int, error) {
	if len(p) > r.size {
		p = p[:r.size]
	}
	return r.reader.Read(p)
}

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return len(p) - 1, nil
}

func FuzzFrameReader(f *testing.F) {
	f.Add([]byte("Content-Length: 2\r\n\r\n{}"))
	f.Add([]byte("content-length: 0\n\n"))
	f.Add([]byte("broken"))
	f.Fuzz(func(_ *testing.T, wire []byte) {
		_, _ = NewFrameReader(bytes.NewReader(wire), 1<<20).ReadFrame()
	})
}

func BenchmarkFrameReader(b *testing.B) {
	payload := []byte(`{"jsonrpc":"2.0","method":"initialized","params":{}}`)
	var wire bytes.Buffer
	require.NoError(b, NewFrameWriter(&wire).WriteFrame(payload))
	frame := append([]byte(nil), wire.Bytes()...)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		decoded, err := NewFrameReader(bytes.NewReader(frame), 1<<20).ReadFrame()
		if err != nil || len(decoded) != len(payload) {
			b.Fatalf("decode frame: %v", err)
		}
	}
}
