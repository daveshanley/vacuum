// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package repair

import (
	"encoding/json"
	"testing"

	"github.com/pb33f/testify/require"
)

func TestRenderJSONExactNumbers(t *testing.T) {
	data, err := RenderJSON([]byte("integer: 9007199254740993\nlarge: 999999999999999999999999999999\nfraction: 0.12345678901234567890123456789\nhex: 0x20000000000001\n"))
	require.NoError(t, err)
	var values map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &values))
	require.Equal(t, "9007199254740993", string(values["integer"]))
	require.Equal(t, "999999999999999999999999999999", string(values["large"]))
	require.Equal(t, "0.12345678901234567890123456789", string(values["fraction"]))
	require.Equal(t, "9007199254740993", string(values["hex"]))
}

func TestRenderJSONRejectsLossyOrAmbiguousValues(t *testing.T) {
	for _, source := range []string{"value: .inf", "value: .nan", "a: 1\na: 2", "a: &a {b: 1}\nc: *a", "a: 1\n---\nb: 2"} {
		_, err := RenderJSON([]byte(source))
		require.Error(t, err, source)
	}
}
