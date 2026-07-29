// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package protocol

import (
	"encoding/json"
	"testing"

	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestDidChangeTextDocumentParamsUnmarshal(t *testing.T) {
	var params DidChangeTextDocumentParams
	require.NoError(t, json.Unmarshal([]byte(`{
	  "textDocument":{"uri":"file:///api.yaml","version":3},
	  "contentChanges":[
	    {"range":{"start":{"line":1,"character":2},"end":{"line":1,"character":4}},"rangeLength":2,"text":"new"},
	    {"text":"whole document"}
	  ]
	}`), &params))

	require.Len(t, params.ContentChanges, 2)
	incremental, ok := params.ContentChanges[0].(TextDocumentContentChangeEvent)
	require.True(t, ok)
	assert.Equal(t, "new", incremental.Text)
	assert.Equal(t, UInteger(2), *incremental.RangeLength)
	whole, ok := params.ContentChanges[1].(TextDocumentContentChangeEventWhole)
	require.True(t, ok)
	assert.Equal(t, "whole document", whole.Text)
}
