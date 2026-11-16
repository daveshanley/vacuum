// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"strconv"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/openapi"
	"github.com/daveshanley/vacuum/repair"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

// DocumentRelationships validates links, tag relationships, and server variable defaults.
type DocumentRelationships struct{}

// GetSchema identifies the document relationship rule function.
func (DocumentRelationships) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "oasDocumentRelationships"}
}

// GetCategory returns the OpenAPI function category.
func (DocumentRelationships) GetCategory() string { return model.FunctionCategoryOpenAPI }

// RunRule checks relationships against the resolved model and reports the authored value.
func (DocumentRelationships) RunRule(nodes []*yaml.Node, ctx model.RuleFunctionContext) []model.RuleFunctionResult {
	if len(nodes) == 0 || ctx.Document == nil || ctx.SpecInfo == nil {
		return nil
	}
	built, _ := ctx.Document.BuildV3Model()
	if built == nil {
		return nil
	}
	return checkOpenAPIRelationships(&built.Model, ctx)
}

func checkOpenAPIRelationships(doc *v3.Document, ctx model.RuleFunctionContext) []model.RuleFunctionResult {
	root := ctx.SpecInfo.RootNode
	version := ctx.SpecInfo.VersionNumeric
	paths := make(map[*yaml.Node]*utils.NodePathIndex)
	var results []model.RuleFunctionResult
	add := func(n *yaml.Node, pointer, message string) {
		owner := root
		finding := model.RuleFunctionResult{Message: message, StartNode: n, EndNode: utils.BuildEndNode(n), Rule: ctx.Rule}
		if rolodex := ctx.Document.GetRolodex(); rolodex != nil {
			finding.Origin = rolodex.FindNodeOrigin(n)
			if finding.Origin != nil && finding.Origin.Index != nil {
				owner = finding.Origin.Index.GetRootNode()
			}
		}
		if _, ok := paths[owner]; !ok {
			paths[owner] = utils.NodePathIndexForContext(ctx, owner)
		}
		path, ok := paths[owner].Lookup(n)
		if !ok {
			path = jsonPointerToJSONPath(pointer, root)
		}
		finding.Path = path
		finding.Paths = []string{path}
		results = append(results, finding)
	}
	checkTagRelationships(doc.Tags, version, add)
	ids := make(map[string]bool)
	type target struct {
		id      string
		node    *yaml.Node
		pointer string
	}
	var targets []target
	seen := make(map[*yaml.Node]bool)
	v := openapi.OperationVisitor{
		Operation: func(op *v3.Operation, _ string) {
			if op.OperationId != "" {
				ids[op.OperationId] = true
			}
		},
		Link: func(link *v3.Link, at string) {
			n := link.GoLow().OperationId.ValueNode
			if n != nil && !seen[n] {
				seen[n] = true
				targets = append(targets, target{link.OperationId, n, at + "/operationId"})
			}
		},
	}
	if version >= 3.1 {
		v.Server = func(s *v3.Server, at string) {
			for name, variable := range s.Variables.FromOldest() {
				if variable == nil || variable.GoLow() == nil {
					continue
				}
				low := variable.GoLow()
				n := low.Default.ValueNode
				if n == nil || seen[n] || len(low.Enum) == 0 {
					continue
				}
				seen[n] = true
				found := false
				for _, entry := range low.Enum {
					if entry.Value == low.Default.Value {
						found = true
						break
					}
				}
				if !found {
					add(n, at+"/variables/"+repair.EscapeToken(name)+"/default", fmt.Sprintf("server variable %q default %q must be a member of its enum", name, low.Default.Value))
				}
			}
		}
	}
	v.Walk(doc)
	for _, t := range targets {
		if !ids[t.id] {
			add(t.node, t.pointer, fmt.Sprintf("link operationId %q does not resolve to an operation", t.id))
		}
	}
	return results
}

func checkTagRelationships(tags []*base.Tag, version float32, add func(*yaml.Node, string, string)) {
	byName := make(map[string]int, len(tags))
	for i, t := range tags {
		if t == nil || t.GoLow() == nil || t.GoLow().Name.ValueNode == nil {
			continue
		}
		if _, exists := byName[t.Name]; exists {
			add(t.GoLow().Name.ValueNode, "/tags/"+strconv.Itoa(i)+"/name", fmt.Sprintf("tag name %q must be unique", t.Name))
			continue
		}
		byName[t.Name] = i
	}
	if version < 3.2 {
		return
	}
	// Each tag has at most one parent. Mark the chain on first visit to keep
	// deep hierarchies linear and avoid recursive traversal of cyclic input.
	visited := make([]int, len(tags))
	for i, t := range tags {
		if t == nil || t.GoLow() == nil || t.GoLow().Parent.ValueNode == nil {
			continue
		}
		if _, exists := byName[t.Parent]; !exists {
			add(t.GoLow().Parent.ValueNode, "/tags/"+strconv.Itoa(i)+"/parent", fmt.Sprintf("tag parent %q does not name a declared tag", t.Parent))
			continue
		}
		for j := i; tags[j].GoLow().Parent.ValueNode != nil; {
			member := tags[j]
			if visited[j] != 0 {
				if visited[j] == i+1 {
					add(member.GoLow().Parent.ValueNode, "/tags/"+strconv.Itoa(j)+"/parent", fmt.Sprintf("tag %q participates in a parent cycle", member.Name))
				}
				break
			}
			visited[j] = i + 1
			next, exists := byName[member.Parent]
			if !exists {
				break
			}
			j = next
		}
	}
}
