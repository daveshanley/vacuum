// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package repair

import (
	"fmt"
	"strconv"

	"github.com/daveshanley/vacuum/model"
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
		data, err = RenderJSON(data)
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
		members := make(map[string]*yaml.Node, len(before.Content)/2)
		for i := 0; i < len(before.Content); i += 2 {
			key := before.Content[i].Value
			if members[key] != nil {
				return fmt.Errorf("an overlay cannot target a duplicate key %q; save the specification instead", key)
			}
			members[key] = before.Content[i+1]
		}
		add := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		for i := 0; i < len(after.Content); i += 2 {
			key, value := after.Content[i], after.Content[i+1]
			old := members[key.Value]
			delete(members, key.Value)
			target := overlayMember(path, key.Value)
			if old != nil && old.Kind == value.Kind {
				if err := diffOverlay(doc, target, old, value); err != nil {
					return err
				}
				continue
			}
			if old != nil {
				doc.Actions = append(doc.Actions, &high.Action{Target: target, Remove: true})
			}
			add.Content = append(add.Content, key, value)
		}
		for i := 0; i < len(before.Content); i += 2 {
			key := before.Content[i].Value
			if members[key] != nil {
				doc.Actions = append(doc.Actions, &high.Action{Target: overlayMember(path, key), Remove: true})
			}
		}
		if len(add.Content) > 0 {
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

// Overlay application may reorder object members and change YAML presentation.
// Scalar tags and values, key types, and array order must survive replay.
func sameOverlayValue(a, b *yaml.Node) bool {
	if a.Kind != b.Kind || a.Tag != b.Tag || a.Value != b.Value || len(a.Content) != len(b.Content) {
		return false
	}
	for i, child := range a.Content {
		if a.Kind == yaml.MappingNode && i%2 == 0 && (child.Value != b.Content[i].Value || child.Tag != b.Content[i].Tag) {
			// Only reordered mappings need an index. Most source keys stay in place.
			members := make(map[string]int, (len(b.Content)-i)/2)
			for j := i; j < len(b.Content); j += 2 {
				members[b.Content[j].Value] = j
			}
			for j := i; j < len(a.Content); j += 2 {
				k, ok := members[a.Content[j].Value]
				if !ok || a.Content[j].Tag != b.Content[k].Tag || !sameOverlayValue(a.Content[j+1], b.Content[k+1]) {
					return false
				}
				delete(members, a.Content[j].Value)
			}
			return true
		}
		if !sameOverlayValue(child, b.Content[i]) {
			return false
		}
	}
	return true
}

// Quoted member selectors also handle names such as "null" that the JSONPath
// parser treats as reserved tokens in dot notation.
func overlayMember(path, key string) string {
	return path + "[" + strconv.Quote(key) + "]"
}
