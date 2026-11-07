package repair

import (
	"encoding/json"
	"fmt"
	"github.com/pb33f/go-yaml"
	"math/big"
	"sort"
	"strconv"
	"strings"
)

// EscapeToken encodes one JSON Pointer path segment.
func EscapeToken(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "~", "~0"), "/", "~1")
}
func tokens(path string) ([]string, error) {
	if path == "" {
		return nil, nil
	}
	if !strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("invalid JSON Pointer %q", path)
	}
	parts := strings.Split(path[1:], "/")
	for i, s := range parts {
		for j := 0; j < len(s); j++ {
			if s[j] == '~' {
				if j+1 == len(s) || (s[j+1] != '0' && s[j+1] != '1') {
					return nil, fmt.Errorf("invalid pointer escape")
				}
				j++
			}
		}
		parts[i] = strings.ReplaceAll(strings.ReplaceAll(s, "~1", "/"), "~0", "~")
	}
	return parts, nil
}
func mapIndex(n *yaml.Node, key string) (int, bool) {
	for i := 0; i < len(n.Content); i += 2 {
		if n.Content[i].Value == key {
			return i, true
		}
	}
	return 0, false
}
func arrayIndex(s string, length int, add bool) (int, error) {
	if add && s == "-" {
		return length, nil
	}
	if s == "" || (len(s) > 1 && s[0] == '0') {
		return 0, fmt.Errorf("invalid array index %q", s)
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid array index %q", s)
		}
	}
	i, err := strconv.Atoi(s)
	if err != nil || i < 0 || i > length || (!add && i == length) {
		return 0, fmt.Errorf("array index out of bounds")
	}
	return i, nil
}
func at(root *yaml.Node, path string) (*yaml.Node, error) {
	parts, err := tokens(path)
	if err != nil {
		return nil, err
	}
	return descend(root, parts)
}
func descend(n *yaml.Node, parts []string) (*yaml.Node, error) {
	for _, part := range parts {
		switch n.Kind {
		case yaml.MappingNode:
			i, ok := mapIndex(n, part)
			if !ok {
				return nil, fmt.Errorf("member %q not found", part)
			}
			n = n.Content[i+1]
		case yaml.SequenceNode:
			i, err := arrayIndex(part, len(n.Content), false)
			if err != nil {
				return nil, err
			}
			n = n.Content[i]
		default:
			return nil, fmt.Errorf("path traverses a scalar")
		}
	}
	return n, nil
}
func parentAt(root *yaml.Node, path string) (*yaml.Node, string, error) {
	parts, err := tokens(path)
	if err != nil {
		return nil, "", err
	}
	if len(parts) == 0 {
		return nil, "", fmt.Errorf("root has no parent")
	}
	p, err := descend(root, parts[:len(parts)-1])
	return p, parts[len(parts)-1], err
}
func valueNode(value json.RawMessage) (*yaml.Node, error) {
	if !json.Valid(value) {
		return nil, fmt.Errorf("operation requires valid JSON value")
	}
	doc, err := ParseDocument(value)
	if err != nil {
		return nil, err
	}
	if err := validateTree(doc.Content[0], make(map[*yaml.Node]bool)); err != nil {
		return nil, err
	}
	clearValueStyle(doc.Content[0])
	return doc.Content[0], nil
}

// Values arrive as JSON, but presentation is chosen by the destination document.
func clearValueStyle(n *yaml.Node) {
	n.Style = 0
	n.Line, n.Column = 0, 0
	for _, child := range n.Content {
		clearValueStyle(child)
	}
}

// Reuse unchanged authored children when a collection replacement only changes
// order or a subset of members. This preserves their styles and source spans.
func retainAuthoredChildren(before, after *yaml.Node) {
	if before.Kind != after.Kind {
		return
	}
	if before.Kind == yaml.MappingNode {
		indices := make(map[string]int, len(before.Content)/2)
		for i := 0; i < len(before.Content); i += 2 {
			indices[before.Content[i].Value] = i
		}
		for i := 0; i < len(after.Content); i += 2 {
			if j, ok := indices[after.Content[i].Value]; ok {
				after.Content[i] = before.Content[j]
				if same, err := equalNodes(before.Content[j+1], after.Content[i+1]); err == nil && same {
					after.Content[i+1] = before.Content[j+1]
				} else {
					retainAuthoredChildren(before.Content[j+1], after.Content[i+1])
				}
			}
		}
	}
}

func clone(n *yaml.Node) *yaml.Node {
	c := *n
	c.Content = make([]*yaml.Node, len(n.Content))
	for i, child := range n.Content {
		c.Content[i] = clone(child)
	}
	return &c
}
func applyOperation(root *yaml.Node, op Operation) (*yaml.Node, error) {
	switch op.Op {
	case "test":
		n, err := at(root, op.Path)
		if err != nil {
			return nil, err
		}
		v, err := valueNode(op.Value)
		if err != nil {
			return nil, err
		}
		equal, err := equalNodes(n, v)
		if err != nil {
			return nil, err
		}
		if !equal {
			return nil, fmt.Errorf("test failed")
		}
		return root, nil
	case "add", "replace":
		v, err := valueNode(op.Value)
		if err != nil {
			return nil, err
		}
		return set(root, op.Path, v, op.Op == "add")
	case "remove":
		return remove(root, op.Path)
	case "copy", "move":
		source, err := at(root, op.From)
		if err != nil {
			return nil, err
		}
		if op.Op == "move" {
			from, err := tokens(op.From)
			if err != nil {
				return nil, err
			}
			to, err := tokens(op.Path)
			if err != nil {
				return nil, err
			}
			if len(to) > len(from) {
				prefix := true
				for i := range from {
					if from[i] != to[i] {
						prefix = false
						break
					}
				}
				if prefix {
					return nil, fmt.Errorf("cannot move into a descendant")
				}
			}
			if op.From == op.Path {
				return root, nil
			}
			// Moving a mapping member to an absent sibling only renames its
			// key. Preserve its position, comments and source coordinates.
			fromParent, oldKey, fromErr := parentAt(root, op.From)
			toParent, newKey, toErr := parentAt(root, op.Path)
			if fromErr == nil && toErr == nil && fromParent == toParent && fromParent.Kind == yaml.MappingNode {
				if _, exists := mapIndex(toParent, newKey); !exists {
					i, _ := mapIndex(fromParent, oldKey)
					fromParent.Content[i].Value = newKey
					return root, nil
				}
			}
			root, err = remove(root, op.From)
			if err != nil {
				return nil, err
			}
		} else {
			source = clone(source)
		}
		return set(root, op.Path, source, true)
	default:
		return nil, fmt.Errorf("unsupported JSON Patch operation %q", op.Op)
	}
}
func set(root *yaml.Node, path string, v *yaml.Node, add bool) (*yaml.Node, error) {
	if path == "" {
		return v, nil
	}
	parent, key, err := parentAt(root, path)
	if err != nil {
		return nil, err
	}
	switch parent.Kind {
	case yaml.MappingNode:
		i, ok := mapIndex(parent, key)
		if ok {
			retainAuthoredChildren(parent.Content[i+1], v)
			v.HeadComment = parent.Content[i+1].HeadComment
			v.LineComment = parent.Content[i+1].LineComment
			v.FootComment = parent.Content[i+1].FootComment
			parent.Content[i+1] = v
		} else {
			if !add {
				return nil, fmt.Errorf("replace member does not exist")
			}
			parent.Content = append(parent.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, v)
		}
	case yaml.SequenceNode:
		i, err := arrayIndex(key, len(parent.Content), add)
		if err != nil {
			return nil, err
		}
		if add {
			parent.Content = append(parent.Content, nil)
			copy(parent.Content[i+1:], parent.Content[i:])
			parent.Content[i] = v
		} else {
			parent.Content[i] = v
		}
	default:
		return nil, fmt.Errorf("target parent is not a collection")
	}
	return root, nil
}
func remove(root *yaml.Node, path string) (*yaml.Node, error) {
	if path == "" {
		return nil, fmt.Errorf("removing the root document is not supported")
	}
	parent, key, err := parentAt(root, path)
	if err != nil {
		return nil, err
	}
	switch parent.Kind {
	case yaml.MappingNode:
		i, ok := mapIndex(parent, key)
		if !ok {
			return nil, fmt.Errorf("remove member does not exist")
		}
		parent.Content = append(parent.Content[:i], parent.Content[i+2:]...)
	case yaml.SequenceNode:
		i, err := arrayIndex(key, len(parent.Content), false)
		if err != nil {
			return nil, err
		}
		parent.Content = append(parent.Content[:i], parent.Content[i+1:]...)
	default:
		return nil, fmt.Errorf("remove parent is not a collection")
	}
	return root, nil
}
func jsonValue(n *yaml.Node) (any, error) {
	switch n.Kind {
	case yaml.DocumentNode:
		if len(n.Content) != 1 {
			return nil, fmt.Errorf("invalid document")
		}
		return jsonValue(n.Content[0])
	case yaml.MappingNode:
		result := make(map[string]any, len(n.Content)/2)
		for i := 0; i < len(n.Content); i += 2 {
			v, err := jsonValue(n.Content[i+1])
			if err != nil {
				return nil, err
			}
			result[n.Content[i].Value] = v
		}
		return result, nil
	case yaml.SequenceNode:
		result := make([]any, len(n.Content))
		for i, child := range n.Content {
			v, err := jsonValue(child)
			if err != nil {
				return nil, err
			}
			result[i] = v
		}
		return result, nil
	case yaml.ScalarNode:
		if n.Tag == "!!str" {
			return n.Value, nil
		}
		if n.Tag == "!!null" {
			return nil, nil
		}
		if n.Tag == "!!int" || n.Tag == "!!float" {
			spelling := strings.ReplaceAll(n.Value, "_", "")
			if json.Valid([]byte(spelling)) && spelling != "null" {
				return json.Number(spelling), nil
			}
			if n.Tag == "!!int" {
				if integer, ok := new(big.Int).SetString(spelling, 0); ok {
					return json.Number(integer.String()), nil
				}
			}
			return nil, fmt.Errorf("numeric value cannot be represented exactly as JSON")
		}
		if n.Tag == "!!bool" {
			var value bool
			if err := n.Decode(&value); err != nil {
				return nil, err
			}
			return value, nil
		}
	}
	return nil, fmt.Errorf("unsupported YAML value")
}
func equalNodes(a, b *yaml.Node) (bool, error) {
	av, err := jsonValue(a)
	if err != nil {
		return false, err
	}
	bv, err := jsonValue(b)
	if err != nil {
		return false, err
	}
	return equalValue(av, bv), nil
}
func equalValue(a, b any) bool {
	switch x := a.(type) {
	case json.Number:
		y, ok := b.(json.Number)
		if !ok {
			return false
		}
		xr, err := canonicalValue(x)
		if err != nil {
			return false
		}
		yr, err := canonicalValue(y)
		return err == nil && xr == yr
	case map[string]any:
		y, ok := b.(map[string]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for k, v := range x {
			w, ok := y[k]
			if !ok || !equalValue(v, w) {
				return false
			}
		}
		return true
	case []any:
		y, ok := b.([]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for i, v := range x {
			if !equalValue(v, y[i]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}

// canonicalValue tags each type so strings and structured values cannot collide
// with the normalized exact representation of a number.
func canonicalValue(v any) (string, error) {
	switch x := v.(type) {
	case json.Number:
		if len(x) > 4096 {
			return "", fmt.Errorf("number exceeds repair comparison budget")
		}
		if i := strings.IndexAny(string(x), "eE"); i >= 0 {
			exponent, err := strconv.Atoi(string(x)[i+1:])
			if err != nil || exponent < -4096 || exponent > 4096 {
				return "", fmt.Errorf("number exponent exceeds repair comparison budget")
			}
		}
		r, ok := new(big.Rat).SetString(string(x))
		if !ok {
			return "", fmt.Errorf("invalid number")
		}
		return "n:" + r.RatString(), nil
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var b strings.Builder
		b.WriteString("m:")
		for _, k := range keys {
			value, err := canonicalValue(x[k])
			if err != nil {
				return "", err
			}
			fmt.Fprintf(&b, "%d:%s%d:%s", len(k), k, len(value), value)
		}
		return b.String(), nil
	case []any:
		var b strings.Builder
		b.WriteString("a:")
		for _, item := range x {
			value, err := canonicalValue(item)
			if err != nil {
				return "", err
			}
			fmt.Fprintf(&b, "%d:%s", len(value), value)
		}
		return b.String(), nil
	default:
		b, err := json.Marshal(x)
		return "s:" + string(b), err
	}
}
