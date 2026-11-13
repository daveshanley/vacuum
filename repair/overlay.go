// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package repair

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi"
	high "github.com/pb33f/libopenapi/datamodel/high/overlay"
	"github.com/pb33f/libopenapi/overlay"
)

// RenderOverlay exports the difference between two OpenAPI documents. It applies
// the serialized actions to the source and checks the result before returning.
func RenderOverlay(source, candidate []byte, format string) ([]byte, error) {
	if format != "" && format != "yaml" && format != "json" {
		return nil, fmt.Errorf("unsupported overlay format %q", format)
	}
	before, err := ParseDocument(source)
	if err != nil {
		return nil, err
	}
	after, err := ParseDocument(candidate)
	if err != nil {
		return nil, err
	}
	root := before.Content[0]
	if root.Kind != yaml.MappingNode || after.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("overlay export requires an OpenAPI specification")
	}
	_, oas := mapIndex(root, "openapi")
	_, swagger := mapIndex(root, "swagger")
	if !oas && !swagger {
		return nil, fmt.Errorf("overlay export requires an OpenAPI specification")
	}
	doc := &high.Overlay{Overlay: "1.1.0", Info: &high.Info{Title: "Reviewed vacuum fixes", Version: "1.0.0"}}
	if err := diffOverlay(doc, "$", root, after.Content[0]); err != nil {
		return nil, err
	}
	if len(doc.Actions) == 0 {
		return nil, fmt.Errorf("selected fixes have no overlay actions")
	}
	data, err := doc.Render()
	if err == nil && format == "json" {
		var document *yaml.Node
		document, err = ParseDocument(data)
		if err == nil {
			data, err = utils.RenderRepairJSON(document)
		}
	}
	if err != nil {
		return nil, err
	}
	parsed, err := libopenapi.NewOverlayDocument(data)
	if err != nil {
		return nil, err
	}
	applied, err := overlay.Apply(source, parsed)
	if err != nil {
		return nil, fmt.Errorf("cannot apply generated overlay: %w", err)
	}
	if len(applied.Warnings) > 0 {
		return nil, fmt.Errorf("generated overlay: %s", applied.Warnings[0].Message)
	}
	replayed, err := ParseDocument(applied.Bytes)
	if err != nil {
		return nil, err
	}
	if !sameOverlayValue(replayed, after) {
		return nil, fmt.Errorf("applying the overlay does not reproduce the reviewed specification")
	}
	return data, nil
}

// Walk corresponding objects once. A missing member is inserted by updating its
// parent. A change of node kind requires removing the old member first.
func diffOverlay(doc *high.Overlay, path string, before, after *yaml.Node) error {
	switch before.Kind {
	case yaml.MappingNode:
		// Overlay updates append new properties. Keep the longest candidate
		// prefix already in source order, then append the remaining members.
		members := make(map[string]int, len(before.Content)/2)
		for i := 0; i < len(before.Content); i += 2 {
			key := before.Content[i].Value
			if _, exists := members[key]; exists {
				return fmt.Errorf("an overlay cannot target a duplicate key %q; save the specification instead", key)
			}
			members[key] = i
		}
		tail, last := 0, -1
		for ; tail < len(after.Content); tail += 2 {
			key, value := after.Content[tail], after.Content[tail+1]
			i, exists := members[key.Value]
			if !exists || i <= last || before.Content[i].Tag != key.Tag || before.Content[i+1].Kind != value.Kind {
				break
			}
			old := before.Content[i+1]
			if value.Kind == yaml.ScalarNode && !sameOverlayValue(old, value) {
				// A parent update also works for property names with control
				// characters that the overlay engine cannot select directly.
				change := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: after.Content[tail : tail+2]}
				doc.Actions = append(doc.Actions, &high.Action{Target: path, Update: change})
			} else if value.Kind != yaml.ScalarNode {
				if err := diffOverlay(doc, overlayMember(path, key.Value), old, value); err != nil {
					return err
				}
			}
			delete(members, key.Value)
			last = i
		}
		for i := 0; i < len(before.Content); i += 2 {
			key := before.Content[i].Value
			if _, remove := members[key]; remove {
				doc.Actions = append(doc.Actions, &high.Action{Target: overlayMember(path, key), Remove: true})
			}
		}
		if tail < len(after.Content) {
			add := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: after.Content[tail:]}
			doc.Actions = append(doc.Actions, &high.Action{Target: path, Update: add})
		}
	case yaml.SequenceNode:
		// Equal-length arrays can keep their positions. For structural edits,
		// keep the common prefix, remove from the end, then append the new tail.
		i := 0
		for ; i < len(before.Content) && i < len(after.Content); i++ {
			old, value := before.Content[i], after.Content[i]
			if len(before.Content) == len(after.Content) && old.Kind == value.Kind {
				if err := diffOverlay(doc, model.AppendResultPathIndex(path, i), old, value); err != nil {
					return err
				}
				continue
			}
			if !sameOverlayValue(old, value) {
				break
			}
		}
		// Deleting items in the middle leaves the matching suffix in place.
		end := len(before.Content)
		delta := end - len(after.Content)
		if delta > 0 {
			j := i
			for ; j < len(after.Content) && sameOverlayValue(before.Content[j+delta], after.Content[j]); j++ {
			}
			if j == len(after.Content) {
				for k := i + delta - 1; k >= i; k-- {
					doc.Actions = append(doc.Actions, &high.Action{Target: model.AppendResultPathIndex(path, k), Remove: true})
				}
				return nil
			}
		}
		for j := len(before.Content) - 1; j >= i; j-- {
			doc.Actions = append(doc.Actions, &high.Action{Target: model.AppendResultPathIndex(path, j), Remove: true})
		}
		if i < len(after.Content) {
			doc.Actions = append(doc.Actions, &high.Action{Target: path, Update: &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Content: after.Content[i:]}})
		}
	case yaml.ScalarNode:
		if before.Tag != after.Tag || before.Value != after.Value {
			doc.Actions = append(doc.Actions, &high.Action{Target: path, Update: after})
		}
	default:
		return fmt.Errorf("unsupported YAML node at %s", path)
	}
	return nil
}

// Compare content in order, including mapping keys and their YAML types.
// Reordered paths can produce different lint results.
func sameOverlayValue(a, b *yaml.Node) bool {
	if a.Kind != b.Kind || a.Tag != b.Tag || a.Value != b.Value || len(a.Content) != len(b.Content) {
		return false
	}
	for i, child := range a.Content {
		if !sameOverlayValue(child, b.Content[i]) {
			return false
		}
	}
	return true
}

// Quoted member selectors also handle names such as "null" that the JSONPath
// parser treats as reserved tokens in dot notation.
func overlayMember(path, key string) string {
	var quoted bytes.Buffer
	encoder := json.NewEncoder(&quoted)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(key)
	return path + "[" + strings.TrimSuffix(quoted.String(), "\n") + "]"
}
