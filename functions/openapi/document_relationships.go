// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/openapi"
	"github.com/daveshanley/vacuum/openapi/transform"
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
	return runDocumentCheck(nodes, ctx, checkOpenAPIRelationships)
}

type reportRelationship func(*yaml.Node, string, string)

func runDocumentCheck(nodes []*yaml.Node, ctx model.RuleFunctionContext, check func(*v3.Document, model.RuleFunctionContext, reportRelationship)) []model.RuleFunctionResult {
	if len(nodes) == 0 || ctx.Document == nil || ctx.SpecInfo == nil {
		return nil
	}
	built, _ := ctx.Document.BuildV3Model()
	if built == nil {
		return nil
	}
	root := ctx.SpecInfo.RootNode
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
	check(&built.Model, ctx, add)
	return results
}

func checkOpenAPIRelationships(doc *v3.Document, ctx model.RuleFunctionContext, add reportRelationship) {
	root := ctx.SpecInfo.RootNode
	version := ctx.SpecInfo.VersionNumeric
	checkTagRelationships(doc.Tags, version, add)
	ids := make(map[string]bool)
	type target struct {
		id      string
		node    *yaml.Node
		pointer string
	}
	var targets []target
	seen := make(map[*yaml.Node]bool)
	type localLink struct {
		field, target *yaml.Node
		pointer       string
		owner         *yaml.Node
	}
	var refs []localLink
	lookup := make(transform.OperationRefLookup)
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
			field := link.GoLow().OperationRef.ValueNode
			if field == nil || seen[field] || (link.OperationRef != "" && !strings.HasPrefix(link.OperationRef, "#")) {
				return
			}
			seen[field] = true
			owner := root
			if idx := link.GoLow().GetIndex(); idx != nil {
				owner = idx.GetRootNode()
			}
			refs = append(refs, localLink{field, lookup.Target(owner, link.OperationRef), at + "/operationRef", owner})
		},
	}
	if version >= 3.1 {
		v.Server = checkServerDefaults("must", add)
	}
	v.Walk(doc)
	if len(refs) > 0 {
		// A merged Operation Object can be a shallow copy of the authored node.
		type location struct {
			owner        *yaml.Node
			line, column int
		}
		operations := make(map[location]bool)
		// A referenced Path Item can live in a fragment with no OpenAPI header.
		openapi.OperationVisitor{Operation: func(op *v3.Operation, _ string) {
			low := op.GoLow()
			if idx := low.GetIndex(); idx != nil {
				operations[location{idx.GetRootNode(), low.RootNode.Line, low.RootNode.Column}] = true
			}
		}}.Walk(doc)
		checked := make(map[*yaml.Node]bool)
		for _, ref := range refs {
			var target location
			if ref.target != nil {
				target = location{ref.owner, ref.target.Line, ref.target.Column}
			}
			if ref.target != nil && !operations[target] && !checked[ref.owner] {
				// A linked operation may be outside the part used by the entry document.
				transform.VisitDocumentOperations(ref.owner, func(node *yaml.Node) { operations[location{ref.owner, node.Line, node.Column}] = true })
				checked[ref.owner] = true
			}
			if !operations[target] {
				add(ref.field, ref.pointer, fmt.Sprintf("link operationRef %q does not resolve to an operation", ref.field.Value))
			}
		}
	}
	// A Link can name an operation elsewhere in an already parsed document.
	rolodex := ctx.Document.GetRolodex()
	for _, t := range targets {
		if !ids[t.id] && rolodex != nil {
			for _, idx := range rolodex.GetIndexes() {
				if idx != nil && idx != rolodex.GetRootIndex() {
					transform.VisitOperationIDs(idx.GetRootNode(), func(id string) { ids[id] = true })
				}
			}
			rolodex = nil // All loaded documents now share the ID table.
		}
		if !ids[t.id] {
			add(t.node, t.pointer, fmt.Sprintf("link operationId %q does not resolve to an operation", t.id))
		}
	}
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
