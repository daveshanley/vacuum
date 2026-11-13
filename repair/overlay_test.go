// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package repair

import (
	"encoding/json"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/overlay"
	"github.com/pb33f/testify/require"
)

func TestRenderOverlayReplaysReviewedChanges(t *testing.T) {
	source := []byte(`openapi: 3.1.0
info: {title: API, version: '1'}
paths:
  /pets:
    get: {summary: Before}
x-old: true
x-items: [a,b,c]
x-reserved: {"true": old}
x-ref: {$ref: 'models.yaml#/Pet'}
x-kind: []
`)
	candidate := []byte(`openapi: 3.1.0
info: {title: API, version: '1'}
paths:
  /pets:
    get: {summary: After, description: Pets}
x-items: [a,c]
x-reserved: {"true": new}
x-ref: {$ref: 'models.yaml#/Pet'}
x-kind: {large: 9007199254740993}
`)
	for _, format := range []string{"yaml", "json"} {
		data, err := RenderOverlay(source, candidate, format)
		require.NoError(t, err)
		require.Equal(t, format == "json", json.Valid(data))
		require.NotContains(t, string(data), "models.yaml")
		ov, err := libopenapi.NewOverlayDocument(data)
		require.NoError(t, err)
		require.Equal(t, "1.1.0", ov.Overlay)
		applied, err := overlay.Apply(source, ov)
		require.NoError(t, err)
		require.Empty(t, applied.Warnings)
		actual, err := RenderJSON(applied.Bytes)
		require.NoError(t, err)
		want, err := RenderJSON(candidate)
		require.NoError(t, err)
		require.Equal(t, string(want), string(actual))
	}
}

func TestOverlayRemovesOnlyTheDuplicateArrayItem(t *testing.T) {
	source := []byte("openapi: 3.1.0\nx-date: 2026-11-13\nx-values: [A,A,B,C]\n")
	candidate := []byte("openapi: 3.1.0\nx-date: 2026-11-13\nx-values: [A,B,C]\n")
	data, err := RenderOverlay(source, candidate, "yaml")
	require.NoError(t, err)
	ov, err := libopenapi.NewOverlayDocument(data)
	require.NoError(t, err)
	require.Len(t, ov.Actions, 1)
	require.Equal(t, `$["x-values"][1]`, ov.Actions[0].Target)
	require.True(t, ov.Actions[0].Remove)
}

func TestOverlayReplaysMappingOrderAndKeyTypes(t *testing.T) {
	before := []byte("openapi: 3.1.0\npaths: {'/pets/{id}': {}, /pets/mine: {}}\ntags: [{name: zebra}, {name: alpaca}]\nresponses: {200: {description: OK}, '201': {description: Created}}\n")
	after := []byte("openapi: 3.1.0\npaths: {/pets/mine: {}, '/pets/{id}': {}}\ntags: [{name: alpaca}, {name: zebra}]\nresponses: {'200': {description: OK}, '201': {description: Created}}\n")
	for _, format := range []string{"yaml", "json"} {
		data, err := RenderOverlay(before, after, format)
		require.NoError(t, err)
		ov, err := libopenapi.NewOverlayDocument(data)
		require.NoError(t, err)
		replayed, err := overlay.Apply(before, ov)
		require.NoError(t, err)
		require.Empty(t, replayed.Warnings)
		doc, err := ParseDocument(replayed.Bytes)
		require.NoError(t, err)
		paths, _ := at(doc.Content[0], "/paths")
		require.Equal(t, "/pets/mine", paths.Content[0].Value)
		tag, _ := at(doc.Content[0], "/tags/0/name")
		require.Equal(t, "alpaca", tag.Value)
		responses, _ := at(doc.Content[0], "/responses")
		require.Equal(t, "200", responses.Content[0].Value)
		require.Equal(t, "!!str", responses.Content[0].Tag)
	}
}

func TestOverlayMemberUsesJSONEscapes(t *testing.T) {
	require.Equal(t, `$["a\u0001b"]`, overlayMember("$", "a\x01b"))
	source := []byte(`{"openapi":"3.1.0","a\u0001b":"before"}`)
	candidate := []byte(`{"openapi":"3.1.0","a\u0001b":"after"}`)
	_, err := RenderOverlay(source, candidate, "json")
	require.NoError(t, err)
}
