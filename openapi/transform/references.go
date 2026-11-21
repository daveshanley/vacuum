// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package transform

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/pb33f/go-yaml"
	libutils "github.com/pb33f/libopenapi/utils"
)

func isDocumentReferenceKey(key string) bool {
	switch key {
	case "$ref", "$dynamicRef", "$recursiveRef", "operationRef":
		return true
	}
	return false
}

// isDocumentReference accepts scalar reference keywords, limits operationRef to
// Link Objects, and permits external reference-like metadata within extensions.
func isDocumentReference(key string, value *yaml.Node, path []string, inExtension bool, version string) bool {
	return isDocumentReferenceKey(key) && value.Kind == yaml.ScalarNode &&
		(key != "operationRef" || (!inExtension && isLinkObjectPath(path, version))) &&
		!(inExtension && isExternalReference(value.Value))
}

func externalReferenceError(path []string, value string) error {
	return fmt.Errorf("inclusive filtering and unused-component pruning require a bundled OpenAPI document; external reference found at %s: %s", jsonPath(path), value)
}

// ValidateBundled rejects document references whose URI portion is external.
// Runtime URLs such as Example Object externalValue values are ignored.
func ValidateBundled(root *yaml.Node) error {
	_, err := prepareRoot(root, false)
	return err
}

func walkExternalReferences(node *yaml.Node, path []string) error {
	if isArbitraryData(node, path) || isWithinExtension(path) {
		return nil
	}
	switch node.Kind {
	case yaml.AliasNode:
		if node.Alias != nil {
			return walkExternalReferences(node.Alias, path)
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			key, value := node.Content[i].Value, node.Content[i+1]
			path = append(path, key)
			next := path
			if isDocumentReferenceKey(key) && value.Kind == yaml.ScalarNode && isExternalReference(value.Value) {
				return externalReferenceError(next, value.Value)
			}
			if key == "discriminator" {
				if err := validateDiscriminatorReferences(value, next); err != nil {
					return err
				}
			}
			if err := walkExternalReferences(value, next); err != nil {
				return err
			}
			path = path[:len(path)-1]
		}
	case yaml.SequenceNode:
		for i, child := range node.Content {
			path = append(path, fmt.Sprintf("%d", i))
			if err := walkExternalReferences(child, path); err != nil {
				return err
			}
			path = path[:len(path)-1]
		}
	}
	return nil
}

func validateDiscriminatorReferences(node *yaml.Node, path []string) error {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key, value := node.Content[i].Value, node.Content[i+1]
		switch key {
		case "mapping":
			if value.Kind != yaml.MappingNode {
				continue
			}
			for j := 0; j+1 < len(value.Content); j += 2 {
				target := value.Content[j+1]
				if target.Kind == yaml.ScalarNode && isExternalDiscriminatorReference(target.Value) {
					p := appendPath(appendPath(path, key), value.Content[j].Value)
					return externalReferenceError(p, target.Value)
				}
			}
		case "defaultMapping":
			if value.Kind == yaml.ScalarNode && isExternalDiscriminatorReference(value.Value) {
				return externalReferenceError(appendPath(path, key), value.Value)
			}
		}
	}
	return nil
}

func isExternalReference(value string) bool { return libutils.IsExternalRef(value) }

func isExternalDiscriminatorReference(value string) bool {
	if value == "" || strings.HasPrefix(value, "#") {
		return false
	}
	// A bare value is a schema component name.
	lower := strings.ToLower(value)
	return strings.ContainsAny(value, "/:\\") ||
		strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".json")
}

func appendPath(path []string, part string) []string {
	next := make([]string, len(path)+1)
	copy(next, path)
	next[len(path)] = part
	return next
}

// OperationRefLookup indexes the mappings traversed by local Link operationRefs during one lint run.
type OperationRefLookup map[*yaml.Node]map[string]*yaml.Node

// OperationRefTarget is the object a local Link operationRef leads to.
type OperationRefTarget struct {
	// Node is the object at the pointer, or nil when the pointer is invalid or the target is not an object.
	Node *yaml.Node
	// Direct is an equivalent pointer that passes through no $ref. It is set only when the walk
	// followed a local reference, which strict JSON Pointer evaluation does not do.
	Direct string
	// External is set when the walk reached a reference to another document. That target is not checked.
	External bool
}

// maxOperationRefHops bounds the local references followed for one operationRef, so cycles stop.
const maxOperationRefHops = 32

var pointerTokenEscaper = strings.NewReplacer("~", "~0", "/", "~1")

// Target resolves a fragment-only JSON Pointer against the document that owns the link.
// Authors expect a pointer such as '#/paths/~1pets/get' to reach an operation in a referenced
// Path Item, so local references met on the way are followed and the strict pointer is recorded.
func (lookup OperationRefLookup) Target(root *yaml.Node, ref string) OperationRefTarget {
	node, direct, followed, external := lookup.target(root, ref, 0)
	target := OperationRefTarget{Node: node, External: external}
	if followed && node != nil {
		target.Direct = direct
	}
	return target
}

func (lookup OperationRefLookup) target(root *yaml.Node, ref string, hops int) (node *yaml.Node, direct string, followed, external bool) {
	if ref == "" || ref == "#" {
		return documentRoot(root), "#", false, false
	}
	fragment, err := url.PathUnescape(strings.TrimPrefix(ref, "#"))
	if err != nil || !strings.HasPrefix(ref, "#") || !strings.HasPrefix(fragment, "/") {
		return nil, "", false, false
	}
	node = libutils.NodeAlias(documentRoot(root))
	direct = "#"
	for part := range strings.SplitSeq(fragment[1:], "/") {
		token, err := decodePointerToken(part)
		if err != nil || node == nil {
			return nil, "", followed, false
		}
		switch node.Kind {
		case yaml.MappingNode:
			members := lookup.members(node)
			// Descend through a reference, unless the token names a sibling of $ref.
			for {
				if _, local := members[token]; local {
					break
				}
				refNode := libutils.NodeAlias(members["$ref"])
				if refNode == nil || refNode.Kind != yaml.ScalarNode {
					break
				}
				if !strings.HasPrefix(refNode.Value, "#") {
					return nil, "", followed, true
				}
				if hops++; hops > maxOperationRefHops {
					return nil, "", followed, false
				}
				// The reference's own target is reached by a strict pointer, which replaces the prefix.
				if node, direct, _, external = lookup.target(root, refNode.Value, hops); node == nil {
					return nil, "", true, external
				}
				followed = true
				members = lookup.members(node)
			}
			node = members[token]
		case yaml.SequenceNode:
			index, err := strconv.Atoi(token)
			if err != nil || index < 0 || index >= len(node.Content) || strconv.Itoa(index) != token {
				return nil, "", followed, false
			}
			node = node.Content[index]
		default:
			return nil, "", followed, false
		}
		direct += "/" + pointerTokenEscaper.Replace(token)
		node = libutils.NodeAlias(node)
	}
	if node == nil || node.Kind != yaml.MappingNode {
		return nil, "", followed, false
	}
	return node, direct, followed, false
}

func (lookup OperationRefLookup) members(node *yaml.Node) map[string]*yaml.Node {
	if members, ok := lookup[node]; ok {
		return members
	}
	content := libutils.ResolveMergeNodes(node).Content
	members := make(map[string]*yaml.Node, len(content)/2)
	for i := 0; i+1 < len(content); i += 2 {
		members[libutils.NodeAlias(content[i]).Value] = content[i+1]
	}
	lookup[node] = members
	return members
}

func localPointerTokens(value string) ([]string, error) {
	if !strings.HasPrefix(value, "#") {
		return nil, fmt.Errorf("reference is not local: %s", value)
	}
	fragment, err := url.PathUnescape(strings.TrimPrefix(value, "#"))
	if err != nil {
		return nil, fmt.Errorf("invalid URI fragment in reference %q: %w", value, err)
	}
	if fragment == "" {
		return nil, nil
	}
	if fragment[0] != '/' {
		return []string{fragment}, nil
	}
	raw := strings.Split(fragment[1:], "/")
	tokens := make([]string, len(raw))
	for i, token := range raw {
		decoded, err := decodePointerToken(token)
		if err != nil {
			return nil, fmt.Errorf("invalid JSON Pointer in reference %q: %w", value, err)
		}
		tokens[i] = decoded
	}
	return tokens, nil
}

func decodePointerToken(token string) (string, error) {
	var b strings.Builder
	for i := 0; i < len(token); i++ {
		if token[i] != '~' {
			b.WriteByte(token[i])
			continue
		}
		if i+1 >= len(token) || (token[i+1] != '0' && token[i+1] != '1') {
			return "", fmt.Errorf("invalid escape in token %q", token)
		}
		i++
		if token[i] == '0' {
			b.WriteByte('~')
		} else {
			b.WriteByte('/')
		}
	}
	return b.String(), nil
}
