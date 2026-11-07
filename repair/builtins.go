// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package repair

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/pb33f/go-yaml"
)

// Finding locates a diagnostic in the canonical source document.
// Pointer must be mapped from the parser's source node, not an English message.
type Finding struct {
	Code        string `json:"code,omitempty"`
	Index       int    `json:"-"`
	Unavailable string `json:"unavailable,omitempty"`
	RuleID      string `json:"ruleId"`
	Pointer     string `json:"pointer"`
}

// BuiltinPlan proposes guarded edits for reported built-in findings. Unhandled
// or ambiguous findings remain explicit declines; no rule is disabled or ignored.
func BuiltinPlan(source string, data []byte, findings []Finding) (Plan, error) {
	findings = orderedFindings(findings)
	plan := NewPlan(source, data)
	parsed, err := ParseDocument(data)
	if err != nil {
		return plan, err
	}
	doc := *parsed
	if len(doc.Content) != 1 {
		return plan, fmt.Errorf("repair requires one document")
	}
	root := doc.Content[0]
	// Duplicate paths must be addressed before the JSON pointer model is usable.
	for _, f := range findings {
		if f.RuleID == "duplicate-paths" && f.Unavailable == "" {
			paths := field(root, "paths")
			if paths == nil || paths.Kind != yaml.MappingNode {
				continue
			}
			for j := len(paths.Content) - 2; j >= 2; j -= 2 {
				for i := 0; i < j; i += 2 {
					if paths.Content[i].Value != paths.Content[j].Value || paths.Content[i].Tag != paths.Content[j].Tag {
						continue
					}
					same, err := equalNodes(paths.Content[i+1], paths.Content[j+1])
					if err == nil && same && uncommented(paths.Content[j]) && uncommented(paths.Content[j+1]) {
						value, err := raw(paths.Content[j+1])
						if err != nil {
							return plan, err
						}
						plan.KeyRemovals = append(plan.KeyRemovals, KeyRemoval{Parent: "/paths", Index: j / 2, Key: paths.Content[j].Value, Value: value})
						break
					}
				}
			}
			if len(plan.KeyRemovals) > 0 {
				plan.Targets = append(plan.Targets, Target{RuleID: f.RuleID, Code: "path-duplicate-identical", Pointer: f.Pointer})
				for _, other := range findings {
					if other.RuleID != f.RuleID || other.Pointer != f.Pointer {
						plan.Declined = append(plan.Declined, Declined{Target: Target{RuleID: other.RuleID, Pointer: other.Pointer}, Reason: "duplicate mapping keys must be repaired first"})
					}
				}
				return plan, nil
			}
		}
	}
	if err := validateTree(root, make(map[*yaml.Node]bool)); err != nil {
		for _, f := range findings {
			plan.Declined = append(plan.Declined, Declined{Target: Target{RuleID: f.RuleID, Pointer: f.Pointer}, Reason: err.Error()})
		}
		return plan, nil
	}
	nodes := sourceNodes(root)
	conflicts := &editIndex{}
	references := hasRef(root)
	seen := make(map[string]bool)
	for _, f := range findings {
		t := Target{RuleID: f.RuleID, Pointer: f.Pointer}
		if f.Unavailable != "" {
			plan.Declined = append(plan.Declined, Declined{Target: t, Reason: f.Unavailable})
			continue
		}
		ops, keys, code, reason := builtin(root, f, references, nodes)
		t.Code = code
		if len(ops) == 0 && len(keys) == 0 {
			if reason == "" {
				reason = "no deterministic repair for this diagnostic"
			}
			plan.Declined = append(plan.Declined, Declined{Target: t, Reason: reason})
			continue
		}
		encoded, _ := json.Marshal(struct {
			Ops  []Operation
			Keys []KeyTagEdit
		}{ops, keys})
		key := string(encoded)
		plan.Targets = append(plan.Targets, t)
		if seen[key] {
			continue
		}
		// Providers work against the same snapshot. Refuse overlapping edit groups;
		// each group retains RFC operation order and is never automatically merged.
		conflict := false
		var footprints []string
		for _, op := range ops {
			footprints = append(footprints, operationFootprint(op, nodes))
			if op.Op == "move" || op.Op == "copy" {
				footprints = append(footprints, sourceFootprint(op, nodes))
			}
		}
		for _, edit := range keys {
			footprints = append(footprints, edit.Parent+"/"+EscapeToken(edit.Key))
		}
		for _, footprint := range footprints {
			if conflicts.conflicts(footprint) {
				conflict = true
				break
			}
		}

		if conflict {
			plan.Targets = plan.Targets[:len(plan.Targets)-1]
			plan.Declined = append(plan.Declined, Declined{Target: t, Reason: "repair overlaps another proposed edit"})
			continue
		}
		for _, footprint := range footprints {
			conflicts.add(footprint)
		}
		seen[key] = true
		plan.Operations = append(plan.Operations, ops...)
		plan.KeyTags = append(plan.KeyTags, keys...)
	}
	return plan, nil
}
func overlaps(a, b string) bool {
	return a == b || strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/") || a == "" || b == ""
}
func field(n *yaml.Node, name string) *yaml.Node {
	if n == nil || n.Kind != yaml.MappingNode {
		return nil
	}
	i, ok := mapIndex(n, name)
	if !ok {
		return nil
	}
	return n.Content[i+1]
}
func ancestorField(nodes map[string]*yaml.Node, path, name string) (*yaml.Node, string) {
	for {
		parts, err := tokens(path)
		if err != nil {
			return nil, ""
		}
		if len(parts) > 0 && parts[len(parts)-1] == name {
			n := nodes[path]
			if n != nil {
				return n, path
			}
		}
		if path == "" {
			break
		}
		path = path[:strings.LastIndex(path, "/")]
	}
	return nil, ""
}
func raw(n *yaml.Node) (json.RawMessage, error) {
	v, err := jsonValue(n)
	if err != nil {
		return nil, err
	}
	return json.Marshal(v)
}
func uncommented(n *yaml.Node) bool {
	if n.Anchor != "" || n.Kind == yaml.AliasNode || n.HeadComment != "" || n.LineComment != "" || n.FootComment != "" {
		return false
	}
	for _, child := range n.Content {
		if !uncommented(child) {
			return false
		}
	}
	return true
}
func hasRef(n *yaml.Node) bool {
	if n.Kind == yaml.MappingNode {
		for i := 0; i < len(n.Content); i += 2 {
			switch n.Content[i].Value {
			case "$ref", "$dynamicRef", "$recursiveRef", "$id", "$anchor", "$dynamicAnchor", "$recursiveAnchor":
				return true
			}
		}
	}
	for _, child := range n.Content {
		if hasRef(child) {
			return true
		}
	}
	return false
}
func builtin(root *yaml.Node, f Finding, references bool, nodes map[string]*yaml.Node) ([]Operation, []KeyTagEdit, string, string) {
	var sequence *yaml.Node
	path := f.Pointer
	code := ""
	switch f.RuleID {
	case "duplicated-entry-in-enum", "json-schema-enum-values-compatible":
		sequence, path = ancestorField(nodes, path, "enum")
		code = "enum-duplicate"
	case "oas2-api-schemes":
		sequence = field(root, "schemes")
		path = "/schemes"
		code = "scheme-duplicate"
	case "openapi-tags":
		sequence = field(root, "tags")
		path = "/tags"
		code = "tag-duplicate"
	case "asyncapi-3-tags-uniqueness":
		sequence, path = ancestorField(nodes, path, "tags")
		code = "tag-duplicate"
	case "operation-parameters", "arazzo-parameter":
		sequence, path = ancestorField(nodes, path, "parameters")
		code = "parameter-duplicate"
	case "path-params":
		n, p := parameterAt(nodes, path)
		if n != nil && field(n, "in") != nil && field(n, "in").Value == "path" {
			req := field(n, "required")
			if req == nil || (req.Tag == "!!bool" && req.Value == "false") {
				op := "replace"
				if req == nil {
					op = "add"
				}
				return []Operation{{Op: op, Path: p + "/required", Value: json.RawMessage("true")}}, nil, "path-required", ""
			}
		}
		sequence, path = ancestorField(nodes, path, "parameters")
		code = "parameter-duplicate"
	case "openapi-tags-alphabetical", "asyncapi-3-tags-alphabetical":
		n, p := ancestorField(nodes, path, "tags")
		if n == nil || n.Kind != yaml.SequenceNode {
			return nil, nil, "", "tag array is unavailable"
		}
		if references {
			return nil, nil, "", "reordering a referenced document requires review"
		}
		if len(n.Content) > 1000 {
			return nil, nil, "", "sorting more than 1000 tags requires review"
		}
		indices := make([]int, len(n.Content))
		for i, tag := range n.Content {
			if name := field(tag, "name"); name == nil || name.Tag != "!!str" {
				return nil, nil, "", "tag names must be strings"
			}
			indices[i] = i
		}
		sort.SliceStable(indices, func(i, j int) bool {
			return field(n.Content[indices[i]], "name").Value < field(n.Content[indices[j]], "name").Value
		})
		current := make([]int, len(indices))
		for i := range current {
			current[i] = i
		}
		var ops []Operation
		for target, wanted := range indices {
			pos := target
			for current[pos] != wanted {
				pos++
			}
			if pos == target {
				continue
			}
			ops = append(ops, Operation{Op: "move", From: p + "/" + strconv.Itoa(pos), Path: p + "/" + strconv.Itoa(target)})
			copy(current[target+1:pos+1], current[target:pos])
			current[target] = wanted
		}
		return ops, nil, "tags-order", ""
	case "operation-success-response":
		n, p := ancestorField(nodes, path, "responses")
		if n == nil || n.Kind != yaml.MappingNode {
			return nil, nil, "", "response mapping is unavailable"
		}
		var edits []KeyTagEdit
		for i := 0; i < len(n.Content); i += 2 {
			k := n.Content[i]
			number, err := strconv.Atoi(k.Value)
			if k.Tag == "!!int" && err == nil && number >= 100 && number <= 599 && strconv.Itoa(number) == k.Value {
				edits = append(edits, KeyTagEdit{Parent: p, Index: i / 2, Key: k.Value, Tag: k.Tag})
			}
		}
		return nil, edits, "response-code-string", ""
	case "json-schema-dependent-required-defined", "oas-schema-check":
		n, p := ancestorField(nodes, path, "dependentRequired")
		if n != nil && n.Kind == yaml.MappingNode {
			var ops []Operation
			for i := 0; i < len(n.Content); i += 2 {
				name := n.Content[i].Value
				values := n.Content[i+1]
				if values.Kind != yaml.SequenceNode {
					continue
				}
				for j := len(values.Content) - 1; j >= 0; j-- {
					v := values.Content[j]
					if v.Tag == "!!str" && v.Value == name && uncommented(v) {
						ops = append(ops, Operation{Op: "remove", Path: p + "/" + EscapeToken(name) + "/" + strconv.Itoa(j)})
					}
				}
			}
			return ops, nil, "dependent-self", ""
		}
		if f.RuleID == "oas-schema-check" {
			n, p = ancestorField(nodes, path, "enum")
			if n != nil && n.Kind == yaml.SequenceNode {
				owner := nodes[p[:strings.LastIndex(p, "/")]]
				c := field(owner, "const")
				version := field(root, "openapi")
				if c != nil && version != nil && (strings.HasPrefix(version.Value, "3.1.") || strings.HasPrefix(version.Value, "3.2.")) && uncommented(n) && mappingKeyUncommented(owner, "enum") {
					for _, v := range n.Content {
						eq, err := equalNodes(v, c)
						if err == nil && eq {
							return []Operation{{Op: "remove", Path: p}}, nil, "enum-redundant-const", ""
						}
					}
				}
			}
		}
		return nil, nil, "", "constraint conflict requires an intended-contract decision"
	case "no-unnecessary-combinator":
		for _, name := range []string{"allOf", "anyOf", "oneOf"} {
			n, p := ancestorField(nodes, path, name)
			if n == nil || n.Kind != yaml.SequenceNode || len(n.Content) != 1 {
				continue
			}
			owner := nodes[p[:strings.LastIndex(p, "/")]]
			if owner == nil || owner.Kind != yaml.MappingNode || len(owner.Content) != 2 || references || !uncommented(owner) {
				return nil, nil, "", "schema scope, siblings, references or comments require review"
			}
			return []Operation{{Op: "move", From: p + "/0", Path: p[:strings.LastIndex(p, "/")]}}, nil, "singleton-combinator", ""
		}
		return nil, nil, "", "no identity combinator found"
	default:
		return nil, nil, "", "requires an explicit policy or reviewed edit"
	}
	if sequence == nil || sequence.Kind != yaml.SequenceNode {
		return nil, nil, code, "repairable sequence is unavailable"
	}
	if references {
		return nil, nil, code, "removing entries in a referenced document requires review"
	}
	// Canonical typed values group duplicates in linear traversal order. Verify
	// equality inside each bucket so encoding mistakes cannot remove data.
	seenValues := make(map[string][]int)
	var duplicates []int
	for j, value := range sequence.Content {
		v, err := jsonValue(value)
		if err != nil {
			return nil, nil, code, err.Error()
		}
		key, err := canonicalValue(v)
		if err != nil {
			return nil, nil, code, err.Error()
		}
		for _, i := range seenValues[key] {
			eq, err := equalNodes(sequence.Content[i], value)
			if err != nil {
				return nil, nil, code, err.Error()
			}
			if eq && uncommented(value) {
				duplicates = append(duplicates, j)
				break
			}
		}
		if len(seenValues[key]) == 0 {
			seenValues[key] = []int{j}
		}
	}
	var ops []Operation
	for i := len(duplicates) - 1; i >= 0; i-- {
		ops = append(ops, Operation{Op: "remove", Path: path + "/" + strconv.Itoa(duplicates[i])})
	}

	return ops, nil, code, ""
}
func parameterAt(nodes map[string]*yaml.Node, path string) (*yaml.Node, string) {
	for {
		n := nodes[path]
		if n != nil && n.Kind == yaml.MappingNode && field(n, "in") != nil {
			return n, path
		}
		if path == "" {
			return nil, ""
		}
		path = path[:strings.LastIndex(path, "/")]
	}
}

func mappingKeyUncommented(owner *yaml.Node, name string) bool {
	i, ok := mapIndex(owner, name)
	return ok && uncommented(owner.Content[i])
}

func orderedFindings(findings []Finding) []Finding {
	ordered := append([]Finding(nil), findings...)
	sort.SliceStable(ordered, func(i, j int) bool {
		a, b := ordered[i], ordered[j]
		if a.Pointer != b.Pointer {
			return a.Pointer < b.Pointer
		}
		if a.RuleID != b.RuleID {
			return a.RuleID < b.RuleID
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		return a.Unavailable < b.Unavailable
	})
	return ordered
}
