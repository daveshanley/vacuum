// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// SPDX-License-Identifier: MIT

package protocol

import (
	"encoding/json"
	"testing"

	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestIntegerOrStringRoundTrip(t *testing.T) {
	for _, value := range []any{Integer(42), "oas3-schema"} {
		encoded, err := json.Marshal(IntegerOrString{Value: value})
		require.NoError(t, err)

		var decoded IntegerOrString
		require.NoError(t, json.Unmarshal(encoded, &decoded))
		assert.Equal(t, value, decoded.Value)
	}
}

func TestBoolOrStringRoundTrip(t *testing.T) {
	for _, value := range []any{true, "workspace-folders"} {
		encoded, err := json.Marshal(BoolOrString{Value: value})
		require.NoError(t, err)

		var decoded BoolOrString
		require.NoError(t, json.Unmarshal(encoded, &decoded))
		assert.Equal(t, value, decoded.Value)
	}
}

func TestIDPreservesStringAndNumericValues(t *testing.T) {
	for _, raw := range []string{`17`, `"client-request"`} {
		id, err := ParseID(json.RawMessage(raw))
		require.NoError(t, err)
		encoded, err := json.Marshal(id)
		require.NoError(t, err)
		assert.Equal(t, raw, string(encoded))
		assert.Equal(t, raw, id.Key())
	}
}

func TestIDUnmarshalRejectsNullAndObjects(t *testing.T) {
	var id ID
	require.NoError(t, json.Unmarshal([]byte(`"request-1"`), &id))
	assert.Equal(t, `"request-1"`, id.Key())
	require.Error(t, json.Unmarshal([]byte(`null`), &id))
	require.Error(t, json.Unmarshal([]byte(`{"request":1}`), &id))
}
