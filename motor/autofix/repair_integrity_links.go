// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package autofix

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/daveshanley/vacuum/repair"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
)

// repairLinkDefects follows typed OpenAPI model owners. Example properties named
// operationId or operationRef are data and must not be treated as graph edges.
func repairLinkDefects(document libopenapi.Document, root *yaml.Node) []repairDefect {
	if document.GetSpecInfo().VersionNumeric < 3 {
		return nil
	}
	built, _ := document.BuildV3Model()
	if built == nil {
		return []repairDefect{{"reference graph", "", "OpenAPI model is unavailable"}}
	}
	locations := repair.IndexLocations(root)
	ids := make(map[string]bool)
	type operationOwner struct {
		id    string
		index *index.SpecIndex
		site  string
	}
	operations := make(map[*yaml.Node]operationOwner)
	links := make(map[*yaml.Node]*v3.Link)
	visited := make(map[*yaml.Node]bool)
	var walkPaths func(*orderedmap.Map[string, *v3.PathItem], string)
	var walkCallbacks func(*orderedmap.Map[string, *v3.Callback], string)
	addLinks := func(values *orderedmap.Map[string, *v3.Link]) {
		for _, link := range values.FromOldest() {
			if link != nil && link.GoLow() != nil {
				links[link.GoLow().RootNode] = link
			}
		}
	}
	response := func(r *v3.Response) {
		if r != nil {
			addLinks(r.Links)
		}
	}
	walkCallbacks = func(callbacks *orderedmap.Map[string, *v3.Callback], prefix string) {
		for name, cb := range callbacks.FromOldest() {
			if cb != nil {
				walkPaths(cb.Expression, prefix+"/"+repair.EscapeToken(name))
			}
		}
	}
	walkPaths = func(paths *orderedmap.Map[string, *v3.PathItem], prefix string) {
		for path, p := range paths.FromOldest() {
			if p == nil || p.GoLow() == nil || visited[p.GoLow().RootNode] {
				continue
			}
			visited[p.GoLow().RootNode] = true
			for method, op := range p.GetOperations().FromOldest() {
				if op == nil || op.GoLow() == nil {
					continue
				}
				operations[op.GoLow().RootNode] = operationOwner{op.OperationId, op.GoLow().GetIndex(), prefix + "/" + repair.EscapeToken(path) + "/" + repair.EscapeToken(method)}
				if op.OperationId != "" {
					ids[op.OperationId] = true
				}
				if op.Responses != nil {
					response(op.Responses.Default)
					for _, r := range op.Responses.Codes.FromOldest() {
						response(r)
					}
				}
				walkCallbacks(op.Callbacks, prefix+"/"+repair.EscapeToken(path)+"/"+repair.EscapeToken(method)+"/callbacks")
			}
		}
	}
	d := &built.Model
	if d.Paths != nil {
		walkPaths(d.Paths.PathItems, "/paths")
	}
	walkPaths(d.Webhooks, "/webhooks")
	if d.Components != nil {
		walkPaths(d.Components.PathItems, "/components/pathItems")
		walkCallbacks(d.Components.Callbacks, "/components/callbacks")
		addLinks(d.Components.Links)
		for _, r := range d.Components.Responses.FromOldest() {
			response(r)
		}
	}
	// Cross-parse low nodes need canonical position mapping before pointer lookup.
	operationPaths := make(map[string]bool)
	idOwners := make(map[string]map[string]repairDefect)
	rootSource := ""
	if rolo := document.GetRolodex(); rolo != nil && rolo.GetRootIndex() != nil {
		rootSource = rolo.GetRootIndex().GetSpecAbsolutePath()
	}
	for n, owner := range operations {
		id := owner.id
		idx := owner.index
		source := ""
		if idx != nil {
			source = idx.GetSpecAbsolutePath()
		}
		if rolo := document.GetRolodex(); rolo != nil {
			if origin := rolo.FindNodeOrigin(n); origin != nil && origin.AbsoluteLocation != "" {
				source = origin.AbsoluteLocation
			}
		}
		pointer, local := locations.Find(n)
		if authored := fixNodeAt(root, owner.site); authored == nil || fixScalar(authored, "$ref") != "" {
			local = false
		}
		if source != "" && rootSource != "" && source != rootSource {
			local = false
		}
		identity := pointer
		defect := repairDefect{"duplicate operationId", pointer, "duplicate operationId: " + id}
		if local {
			operationPaths[pointer] = true
		} else {
			// Keep external operations in the graph. Frozen-index replay can
			// change their index ownership; the authored reference site is stable.
			site := owner.site
			for site != "" && fixNodeAt(root, site) == nil {
				site = site[:strings.LastIndex(site, "/")]
			}
			identity = "graph:" + owner.site
			defect.pointer = site
			if ref := fixScalar(fixNodeAt(root, site), "$ref"); ref != "" {
				defect.detail += " through " + ref
			}

		}
		if id != "" {
			if idOwners[id] == nil {
				idOwners[id] = make(map[string]repairDefect)
			}
			idOwners[id][identity] = defect
		}
	}
	var defects []repairDefect
	identifiers := make([]string, 0, len(idOwners))
	for id := range idOwners {
		identifiers = append(identifiers, id)
	}
	sort.Strings(identifiers)
	for _, id := range identifiers {
		owners := idOwners[id]
		if len(owners) < 2 {
			continue
		}
		paths := make([]string, 0, len(owners))
		for p := range owners {
			paths = append(paths, p)
		}
		sort.Strings(paths)
		for _, p := range paths {
			defects = append(defects, owners[p])
		}
	}
	for node, link := range links {
		pointer, _ := locations.Find(node)
		if link.OperationId != "" && !ids[link.OperationId] {
			defects = append(defects, repairDefect{"operation link", pointer, "operationId target does not exist: " + link.OperationId})
		}
		if link.OperationRef == "" {
			continue
		}
		ref, err := url.Parse(link.OperationRef)
		if err != nil || ref.Scheme != "" || ref.Host != "" || ref.Path != "" || ref.RawQuery != "" {
			defects = append(defects, repairDefect{"operation link", pointer, "external operationRef cannot be verified: " + link.OperationRef})
			continue
		}
		target := ref.Fragment
		if !strings.HasPrefix(target, "/") || !operationPaths[target] {
			defects = append(defects, repairDefect{"operation link", pointer, fmt.Sprintf("operationRef target is not an operation: %s", link.OperationRef)})
		}
	}
	return defects
}
