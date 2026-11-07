// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package utils

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/pb33f/go-yaml"
)

// RenderRepairJSON renders a JSON-compatible node tree without converting its
// numeric lexemes through float64. It preserves mapping order and rejects
// aliases, duplicate keys and non-JSON scalar values instead of coercing them.
func RenderRepairJSON(root *yaml.Node, original ...[]byte) ([]byte, error) {
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	writeString := func(value string) error {
		if err := encoder.Encode(value); err != nil {
			return err
		}
		out.Truncate(out.Len() - 1) // Encode appends one newline after the string.
		return nil
	}
	active := make(map[*yaml.Node]bool)
	var write func(*yaml.Node, int) error
	write = func(n *yaml.Node, depth int) error {
		if n == nil || depth > 512 || active[n] {
			return fmt.Errorf("invalid or excessive JSON repair tree")
		}
		active[n] = true
		defer delete(active, n)
		switch n.Kind {
		case yaml.DocumentNode:
			if len(n.Content) != 1 {
				return fmt.Errorf("JSON repair requires one document")
			}
			return write(n.Content[0], depth+1)
		case yaml.MappingNode:
			if len(n.Content)%2 != 0 {
				return fmt.Errorf("invalid mapping")
			}
			out.WriteByte('{')
			seen := make(map[string]bool, len(n.Content)/2)
			for i := 0; i < len(n.Content); i += 2 {
				k := n.Content[i]
				if k.Kind != yaml.ScalarNode || (k.Tag != "!!str" && k.Tag != "!!int") {
					return fmt.Errorf("JSON repair requires scalar string keys")
				}
				if seen[k.Value] {
					return fmt.Errorf("duplicate JSON key %q", k.Value)
				}
				seen[k.Value] = true
				if i > 0 {
					out.WriteByte(',')
				}
				if err := writeString(k.Value); err != nil {
					return err
				}
				out.WriteByte(':')
				if err := write(n.Content[i+1], depth+1); err != nil {
					return err
				}
			}
			out.WriteByte('}')
		case yaml.SequenceNode:
			out.WriteByte('[')
			for i, child := range n.Content {
				if i > 0 {
					out.WriteByte(',')
				}
				if err := write(child, depth+1); err != nil {
					return err
				}
			}
			out.WriteByte(']')
		case yaml.ScalarNode:
			switch n.Tag {
			case "!!str":
				if err := writeString(n.Value); err != nil {
					return err
				}
			case "!!null":
				out.WriteString("null")
			case "!!bool":
				var value bool
				if err := n.Decode(&value); err != nil {
					return err
				}
				if value {
					out.WriteString("true")
				} else {
					out.WriteString("false")
				}
			case "!!int", "!!float":
				if len(n.Value) == 0 || (n.Value[0] != '-' && (n.Value[0] < '0' || n.Value[0] > '9')) || !json.Valid([]byte(n.Value)) {
					return fmt.Errorf("non-JSON number %q in repaired document", n.Value)
				}
				out.WriteString(n.Value)
			default:
				return fmt.Errorf("unsupported JSON scalar tag %q", n.Tag)
			}
		default:
			return fmt.Errorf("unsupported JSON node kind %d", n.Kind)
		}
		return nil
	}
	if err := write(root, 0); err != nil {
		return nil, err
	}
	indent := "  "
	if len(original) > 0 {
		indent = repairJSONIndent(original[0])
		if indent == "" {
			return repairLineEndings(out.Bytes(), original[0]), nil
		}
	}
	var formatted bytes.Buffer
	if err := json.Indent(&formatted, out.Bytes(), "", indent); err != nil {
		return nil, err
	}
	if len(original) > 0 {
		return repairLineEndings(formatted.Bytes(), original[0]), nil
	}
	return formatted.Bytes(), nil
}
