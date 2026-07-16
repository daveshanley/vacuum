// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package lsp

import (
	"errors"
	"testing"

	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestDecodeMessageClassifiesJSONAndEnvelopeErrors(t *testing.T) {
	tests := []struct {
		name      string
		payload   string
		wantParse bool
	}{
		{name: "syntax", payload: `{`, wantParse: true},
		{name: "trailing JSON", payload: `{} {}`, wantParse: true},
		{name: "wrong version", payload: `{"jsonrpc":"1.0","id":1,"result":null}`},
		{name: "missing method and id", payload: `{"jsonrpc":"2.0"}`},
		{name: "request with result", payload: `{"jsonrpc":"2.0","id":1,"method":"test","result":null}`},
		{name: "response without outcome", payload: `{"jsonrpc":"2.0","id":1}`},
		{name: "response with both outcomes", payload: `{"jsonrpc":"2.0","id":1,"result":null,"error":{"code":-32603,"message":"failed"}}`},
		{name: "method wrong type", payload: `{"jsonrpc":"2.0","id":1,"method":7}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decodeMessage([]byte(tt.payload))
			require.Error(t, err)
			var decodeErr *messageDecodeError
			require.True(t, errors.As(err, &decodeErr))
			assert.Equal(t, tt.wantParse, decodeErr.parse)
		})
	}
}

func TestDecodeMessageAcceptsCurrentJSONRPCShapes(t *testing.T) {
	for _, payload := range []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","method":"initialized","params":{}}`,
		`{"jsonrpc":"2.0","id":"server-1","result":null}`,
		`{"jsonrpc":"2.0","id":2,"error":{"code":-32603,"message":"failed"}}`,
	} {
		message, err := decodeMessage([]byte(payload))
		require.NoError(t, err)
		require.NotNil(t, message)
	}
}

func FuzzDecodeMessage(f *testing.F) {
	for _, seed := range []string{
		`{"jsonrpc":"2.0","method":"initialized","params":{}}`,
		`{"jsonrpc":"2.0","id":1,"result":null}`,
		`{`,
	} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(_ *testing.T, payload []byte) {
		_, _ = decodeMessage(payload)
	})
}
