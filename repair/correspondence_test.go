package repair

import (
	"encoding/json"
	"github.com/pb33f/testify/require"
	"testing"
)

func TestTraceChangesPreservesMovedAndNestedSources(t *testing.T) {
	source := []byte("items:\n  - name: first\n    values: [a, a, b]\n  - name: second\n    values: [c, d]\n")
	plan := NewPlan("test.yaml", source)
	plan.Operations = []Operation{{Op: "remove", Path: "/items/0/values/1"}, {Op: "move", From: "/items/1", Path: "/items/0"}, {Op: "add", Path: "/items/0/new", Value: json.RawMessage(`true`)}}
	candidate, err := Apply("test.yaml", source, plan)
	require.NoError(t, err)
	mapping, err := TraceChanges(source, candidate, plan)
	require.NoError(t, err)
	for after, before := range map[string]string{"/items/0/name": "/items/1/name", "/items/1/name": "/items/0/name", "/items/1/values/1": "/items/0/values/2"} {
		origin, ok := mapping.OriginalPointer(after)
		require.True(t, ok, after)
		require.Equal(t, before, origin)
	}
	_, ok := mapping.OriginalPointer("/items/0/new")
	require.False(t, ok, "new fields cannot claim an authored origin")
}

func TestTraceChangesRejectsUnknownSequenceCorrespondence(t *testing.T) {
	source := []byte("items: [a, b]\n")
	mapping, err := TraceChanges(source, []byte("items: [x, a, b]\n"), NewPlan("test.yaml", source))
	require.NoError(t, err)
	_, ok := mapping.OriginalPointer("/items/1")
	require.False(t, ok, "unknown plugin shifts must not borrow another item's identity")
}
