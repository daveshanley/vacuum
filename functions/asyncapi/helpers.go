// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package asyncapi

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/go-yaml"
)

var templateExpression = regexp.MustCompile(`\{([^{}]+)\}`)

func rootNode(context model.RuleFunctionContext) *yaml.Node {
	if context.AsyncAPI == nil {
		return nil
	}
	root := context.AsyncAPI.Root()
	if root != nil && root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		return root.Content[0]
	}
	return root
}

func nodePath(context model.RuleFunctionContext, node *yaml.Node, fallback string) string {
	if context.AsyncAPI != nil {
		if path, ok := context.AsyncAPI.NodePath(node); ok {
			return path
		}
	}
	if context.Index != nil {
		if path, ok := vacuumUtils.NodePathIndexForContext(context, context.Index.GetRootNode()).Lookup(node); ok {
			return path
		}
	}
	if fallback != "" {
		return fallback
	}
	return "$"
}

func result(context model.RuleFunctionContext, node *yaml.Node, path, message string) model.RuleFunctionResult {
	if node == nil {
		node = &yaml.Node{Line: 1, Column: 1}
	}
	if path == "" {
		path = nodePath(context, node, "$")
	}
	return model.RuleFunctionResult{
		Message:   message,
		StartNode: node,
		EndNode:   vacuumUtils.BuildEndNode(node),
		Path:      path,
		Rule:      context.Rule,
	}
}

func mappingValue(node *yaml.Node, key string) (*yaml.Node, *yaml.Node) {
	if node == nil {
		return nil, nil
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}
	if node.Kind != yaml.MappingNode {
		return nil, nil
	}
	for i := 0; i < len(node.Content)-1; i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i], node.Content[i+1]
		}
	}
	return nil, nil
}

func mappingKeys(node *yaml.Node) map[string]*yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return map[string]*yaml.Node{}
	}
	keys := make(map[string]*yaml.Node, len(node.Content)/2)
	for i := 0; i < len(node.Content)-1; i += 2 {
		keys[node.Content[i].Value] = node.Content[i]
	}
	return keys
}

func mappingEntries(node *yaml.Node) [][2]*yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	entries := make([][2]*yaml.Node, 0, len(node.Content)/2)
	for i := 0; i < len(node.Content)-1; i += 2 {
		entries = append(entries, [2]*yaml.Node{node.Content[i], node.Content[i+1]})
	}
	return entries
}

func scalarValue(node *yaml.Node) string {
	if node == nil {
		return ""
	}
	return node.Value
}

func refValue(node *yaml.Node) string {
	_, value := mappingValue(node, "$ref")
	if value != nil && value.Kind == yaml.ScalarNode && value.Tag == "!!str" {
		return value.Value
	}
	return ""
}

// resolveObject uses the document index and retains the target's reference scope.
func resolveObject(node *yaml.Node, ctx model.RuleFunctionContext) (*yaml.Node, model.RuleFunctionContext) {
	seen := make(map[*yaml.Node]bool)
	for node != nil {
		_, ref := mappingValue(node, "$ref")
		if ref == nil {
			return node, ctx
		}
		if seen[node] || refValue(node) == "" || ctx.Index == nil {
			return nil, ctx
		}
		seen[node] = true
		target, owner, _ := ctx.Index.SearchIndexForReferenceWithContext(context.Background(), ref.Value)
		if target == nil {
			return nil, ctx
		}
		node = target.Node
		if owner != nil {
			ctx.Index = owner
		}
	}
	return nil, ctx
}

// referenceTokens decodes the URI fragment before splitting JSON Pointer tokens.
func referenceTokens(ref string) (string, []string, bool) {
	u, err := url.Parse(ref)
	if err != nil || ref == "" || (u.Fragment != "" && !strings.HasPrefix(u.Fragment, "/")) {
		return "", nil, false
	}
	var parts []string
	if u.Fragment != "" {
		parts = strings.Split(u.Fragment[1:], "/")
	}
	for j, part := range parts {
		for i := 0; i < len(part); i++ {
			if part[i] == '~' {
				if i+1 == len(part) || (part[i+1] != '0' && part[i+1] != '1') {
					return "", nil, false
				}
				i++
			}
		}
		parts[j] = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
	}
	u.Fragment, u.RawFragment = "", ""
	if u.Path != "" {
		u.Path, u.RawPath = path.Clean(u.Path), ""
	}
	return u.String(), parts, true
}

func referenceDocument(ctx model.RuleFunctionContext, base string) string {
	if ctx.Index == nil {
		return base
	}
	source := ctx.Index.GetSpecAbsolutePath()
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		origin, _ := url.Parse(source)
		relative, err := url.Parse(base)
		if err == nil {
			return origin.ResolveReference(relative).String()
		}
	}
	if base == "" {
		return source
	}
	if address, err := url.Parse(base); err == nil && address.IsAbs() {
		if address.Scheme != "file" {
			return address.String()
		}
		base = filepath.FromSlash(address.Path)
		if len(base) > 2 && base[0] == filepath.Separator && filepath.VolumeName(base[1:]) != "" {
			base = base[1:]
		}
	}
	if filepath.IsAbs(base) {
		return filepath.Clean(base)
	}
	return filepath.Join(filepath.Dir(source), filepath.FromSlash(base))
}

func componentRefName(ref, location string) string {
	prefix := "#/components/" + location + "/"
	if !strings.HasPrefix(ref, prefix) {
		return ""
	}
	return strings.TrimPrefix(ref, prefix)
}

func rootRefName(ref, location string) string {
	prefix := "#/" + location + "/"
	if !strings.HasPrefix(ref, prefix) {
		return ""
	}
	return strings.TrimPrefix(ref, prefix)
}

func collectRefs(node *yaml.Node, refs map[string]bool) {
	if node == nil {
		return
	}
	if node.Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content)-1; i += 2 {
			key := node.Content[i]
			value := node.Content[i+1]
			if key.Value == "$ref" && value.Value != "" {
				refs[value.Value] = true
			}
			collectRefs(value, refs)
		}
		return
	}
	for _, child := range node.Content {
		collectRefs(child, refs)
	}
}

func templateVariables(value string) []string {
	matches := templateExpression.FindAllStringSubmatch(value, -1)
	var variables []string
	if len(matches) > 0 {
		variables = make([]string, 0, len(matches))
	}
	for _, match := range matches {
		if len(match) > 1 && match[1] != "" {
			variables = append(variables, match[1])
		}
	}
	sort.Strings(variables)
	return variables
}

func validateTemplateVariables(
	context model.RuleFunctionContext,
	ownerNode *yaml.Node,
	value string,
	variablesNode *yaml.Node,
	location string,
) []model.RuleFunctionResult {
	declared := mappingKeys(variablesNode)
	used := templateVariables(value)
	usedSet := make(map[string]bool, len(used))
	var results []model.RuleFunctionResult
	for _, name := range used {
		usedSet[name] = true
		if declared[name] == nil {
			results = append(results, result(
				context,
				ownerNode,
				nodePath(context, ownerNode, ""),
				fmt.Sprintf("%s variable `%s` is used but not defined.", location, name),
			))
		}
	}

	declaredNames := make([]string, 0, len(declared))
	for name := range declared {
		declaredNames = append(declaredNames, name)
	}
	sort.Strings(declaredNames)
	for _, name := range declaredNames {
		keyNode := declared[name]
		if !usedSet[name] {
			results = append(results, result(
				context,
				keyNode,
				nodePath(context, keyNode, ""),
				fmt.Sprintf("%s variable `%s` is defined but not used.", location, name),
			))
		}
	}
	return results
}

func componentMap(root *yaml.Node, location string) *yaml.Node {
	_, components := mappingValue(root, "components")
	if components == nil {
		return nil
	}
	_, node := mappingValue(components, location)
	return node
}
