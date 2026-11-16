// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

// Package openapi contains shared traversal of the typed OpenAPI document.
package openapi

import (
	"strconv"

	"github.com/daveshanley/vacuum/repair"
	"github.com/pb33f/go-yaml"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
)

// OperationVisitor receives typed graph objects and the JSON Pointer to their use site.
// Nil callbacks are skipped. Referenced path items are visited once.
type OperationVisitor struct {
	Operation func(*v3.Operation, string)
	Link      func(*v3.Link, string)
	Server    func(*v3.Server, string)
}

func pointer(base, key string) string {
	return base + "/" + repair.EscapeToken(key)
}

// Walk visits objects reachable from paths, webhooks, callbacks and components.
func (v OperationVisitor) Walk(d *v3.Document) {
	if d == nil {
		return
	}
	servers := func(items []*v3.Server, owner string) {
		if v.Server == nil {
			return
		}
		for i, s := range items {
			if s != nil {
				v.Server(s, owner+"/servers/"+strconv.Itoa(i))
			}
		}
	}
	links := func(items *orderedmap.Map[string, *v3.Link], owner string) {
		for name, l := range items.FromOldest() {
			if l == nil || l.GoLow() == nil {
				continue
			}
			at := pointer(owner+"/links", name)
			if v.Link != nil {
				v.Link(l, at)
			}
			if l.Server != nil && v.Server != nil {
				v.Server(l.Server, at+"/server")
			}
		}
	}
	response := func(r *v3.Response, at string) {
		if r != nil {
			links(r.Links, at)
		}
	}
	var paths func(*orderedmap.Map[string, *v3.PathItem], string)
	callbacks := func(items *orderedmap.Map[string, *v3.Callback], owner string) {
		for name, c := range items.FromOldest() {
			if c != nil {
				paths(c.Expression, pointer(owner+"/callbacks", name))
			}
		}
	}
	visited := make(map[*yaml.Node]bool)
	paths = func(items *orderedmap.Map[string, *v3.PathItem], owner string) {
		for name, p := range items.FromOldest() {
			if p == nil || p.GoLow() == nil || visited[p.GoLow().RootNode] {
				continue
			}
			visited[p.GoLow().RootNode] = true
			at := pointer(owner, name)
			servers(p.Servers, at)
			for method, op := range p.GetOperations().FromOldest() {
				if op == nil || op.GoLow() == nil {
					continue
				}
				base := at
				if p.AdditionalOperations != nil && p.AdditionalOperations.GetOrZero(method) == op {
					base += "/additionalOperations"
				}
				site := pointer(base, method)
				if v.Operation != nil {
					v.Operation(op, site)
				}
				servers(op.Servers, site)
				if op.Responses != nil {
					response(op.Responses.Default, site+"/responses/default")
					for code, r := range op.Responses.Codes.FromOldest() {
						response(r, pointer(site+"/responses", code))
					}
				}
				callbacks(op.Callbacks, site)
			}
		}
	}
	servers(d.Servers, "")
	if d.Paths != nil {
		paths(d.Paths.PathItems, "/paths")
	}
	paths(d.Webhooks, "/webhooks")
	if c := d.Components; c != nil {
		paths(c.PathItems, "/components/pathItems")
		callbacks(c.Callbacks, "/components")
		links(c.Links, "/components")
		for name, r := range c.Responses.FromOldest() {
			response(r, pointer("/components/responses", name))
		}
	}
}
