// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package asyncapi

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libasyncapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectFormatSkipsNonAsyncAPIDocumentsWithoutParsing(t *testing.T) {
	format, err := DetectFormat([]byte("{"))

	require.NoError(t, err)
	assert.Empty(t, format)
}

func TestDetectFormatReturnsAsyncAPIModelConstants(t *testing.T) {
	format, err := DetectFormat([]byte("asyncapi: 3.1.0\ninfo:\n  title: Test\n  version: 1.0.0\n"))

	require.NoError(t, err)
	assert.Equal(t, model.AsyncAPI31, format)
}

func TestDetectFormatRejectsAsyncAPI2(t *testing.T) {
	_, err := DetectFormat([]byte("asyncapi: 2.6.0\ninfo:\n  title: Test\n  version: 1.0.0\n"))

	require.Error(t, err)
	assert.True(t, errors.Is(err, libasyncapi.ErrAsyncAPI2NotSupported))
}

func TestDetectFormatRejectsInvalidAsyncAPIMinor(t *testing.T) {
	_, err := DetectFormat([]byte("asyncapi: 3.x.0\ninfo:\n  title: Test\n  version: 1.0.0\n"))

	require.Error(t, err)
	assert.True(t, errors.Is(err, libasyncapi.ErrInvalidAsyncAPIVersion))
}

func BenchmarkDetectFormatAndNewContext(b *testing.B) {
	spec := []byte(benchmarkAsyncAPISpec(200))
	b.ReportAllocs()
	for b.Loop() {
		format, err := DetectFormat(spec)
		if err != nil {
			b.Fatal(err)
		}
		ctx, err := NewContext(spec, "", nil)
		if err != nil {
			b.Fatal(err)
		}
		if ctx.Format != format {
			b.Fatalf("format mismatch: %s != %s", ctx.Format, format)
		}
	}
}

func BenchmarkNewContextOnly(b *testing.B) {
	spec := []byte(benchmarkAsyncAPISpec(200))
	b.ReportAllocs()
	for b.Loop() {
		ctx, err := NewContext(spec, "", nil)
		if err != nil {
			b.Fatal(err)
		}
		if ctx.Format != model.AsyncAPI31 {
			b.Fatalf("unexpected format: %s", ctx.Format)
		}
	}
}

func benchmarkAsyncAPISpec(size int) string {
	var builder strings.Builder
	builder.WriteString(`asyncapi: 3.1.0
info:
  title: Benchmark API
  version: 1.0.0
defaultContentType: application/json
servers:
  production:
    host: api.example.com
    protocol: mqtt
channels:
`)
	for i := range size {
		fmt.Fprintf(&builder, `  channel%d:
    address: events/%d/{id}
    parameters:
      id:
        description: Event id
    messages:
      event%d:
        $ref: '#/components/messages/Message%d'
`, i, i, i, i)
	}
	builder.WriteString("operations:\n")
	for i := range size {
		fmt.Fprintf(&builder, `  sendEvent%d:
    action: send
    channel:
      $ref: '#/channels/channel%d'
    messages:
      - $ref: '#/components/messages/Message%d'
`, i, i, i)
	}
	builder.WriteString("components:\n  messages:\n")
	for i := range size {
		fmt.Fprintf(&builder, `    Message%d:
      payload:
        type: object
        properties:
          id:
            type: string
          value:
            type: number
`, i)
	}
	return builder.String()
}
