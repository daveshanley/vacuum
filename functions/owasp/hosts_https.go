// Copyright 2023 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/go-yaml"
	"net/url"
	"slices"
	"strings"
)

// HostsHttps checks declared server URLs for insecure transport.
type HostsHttps struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the HostsHttps rule.
func (hh HostsHttps) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "owaspHostsHttps"}
}

// GetCategory returns the category of the HostsHttps rule.
func (hh HostsHttps) GetCategory() string {
	return model.FunctionCategoryOWASP
}

// RunRule checks root, path, and operation server URLs, including variable values.
func (hh HostsHttps) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil || context.DrDocument.V3Document == nil {
		return results
	}

	doc := context.DrDocument.V3Document
	servers := slices.Clone(doc.Servers)
	var paths []*v3.PathItem
	if doc.Paths != nil {
		for pair := doc.Paths.PathItems.First(); pair != nil; pair = pair.Next() {
			paths = append(paths, pair.Value())
		}
	}
	for pair := doc.Webhooks.First(); pair != nil; pair = pair.Next() {
		paths = append(paths, pair.Value())
	}
	seenPaths := make(map[*v3.PathItem]bool)
	for len(paths) > 0 {
		path := paths[len(paths)-1]
		paths = paths[:len(paths)-1]
		if path == nil || seenPaths[path] {
			continue
		}
		seenPaths[path] = true
		servers = append(servers, path.Servers...)
		for operation := path.GetOperations().First(); operation != nil; operation = operation.Next() {
			servers = append(servers, operation.Value().Servers...)
			for callback := operation.Value().Callbacks.First(); callback != nil; callback = callback.Next() {
				for expression := callback.Value().Expression.First(); expression != nil; expression = expression.Next() {
					paths = append(paths, expression.Value())
				}
			}
		}
	}
	for _, server := range servers {
		rawURL := server.Value.URL
		var replacements []string
		for pair := server.Value.Variables.First(); pair != nil; pair = pair.Next() {
			replacements = append(replacements, "{"+pair.Key()+"}", pair.Value().Default)
		}
		candidates := []string{rawURL}
		if len(replacements) > 0 {
			defaults := strings.NewReplacer(replacements...)
			candidates[0] = defaults.Replace(rawURL)
			// Vary one declared value at a time; do not expand the Cartesian product.
			for pair := server.Value.Variables.First(); pair != nil; pair = pair.Next() {
				placeholder := "{" + pair.Key() + "}"
				if !strings.Contains(rawURL, placeholder) {
					continue
				}
				for _, value := range pair.Value().Enum {
					candidates = append(candidates, defaults.Replace(strings.ReplaceAll(rawURL, placeholder, value)))
				}
			}
		}
		for _, candidate := range candidates {
			parsed, err := url.Parse(candidate)
			// Relative URLs inherit transport from their deployment location.
			if err != nil || parsed.Scheme == "" || strings.EqualFold(parsed.Scheme, "https") {
				continue
			}

			node := server.Value.GoLow().URL.KeyNode
			result := model.RuleFunctionResult{
				Message:   vacuumUtils.SuppliedOrDefault(context.Rule.Message, "server URLs should use TLS (https)"),
				StartNode: node,
				EndNode:   vacuumUtils.BuildEndNode(node),
				Path:      fmt.Sprintf("%s.url", server.GenerateJSONPath()),
				Rule:      context.Rule,
			}
			server.AddRuleFunctionResult(v3.ConvertRuleResult(&result))
			results = append(results, result)
			break
		}
	}
	return results
}
