// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package transform

import (
	"fmt"
	"net/url"
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
