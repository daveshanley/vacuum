// Copyright 2024 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package openapi

import (
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/base"
	"github.com/pb33f/libopenapi/datamodel/high"
	"github.com/pb33f/libopenapi/datamodel/low"
	"gopkg.in/yaml.v3"
	"slices"
)

// ExamplesMissing will check anything that can have an example, has one.
type ExamplesMissing struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the ComponentDescription rule.
func (em ExamplesMissing) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "examples_missing"}
}

// RunRule will execute the ComponentDescription rule, based on supplied context and a supplied []*yaml.Node slice.
func (em ExamplesMissing) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	buildResult := func(message, path string, node *yaml.Node, component base.AcceptsRuleResults) model.RuleFunctionResult {
		result := model.RuleFunctionResult{
			Message:   message,
			StartNode: node,
			EndNode:   vacuumUtils.BuildEndNode(node),
			Path:      path,
			Rule:      context.Rule,
		}
		component.AddRuleFunctionResult(base.ConvertRuleResult(&result))
		return result
	}

	isExampleNodeNull := func(nodes []*yaml.Node) bool {
		if len(nodes) <= 0 {
			return true
		}
		for i := range nodes {
			if nodes[i] == nil || nodes[i].Tag == "!!null" {
				return true
			}
		}
		return false
	}

	seen := make(map[[32]byte]bool)

	if context.DrDocument.Parameters != nil {
	paramClear:
		for i := range context.DrDocument.Parameters {
			p := context.DrDocument.Parameters[i]
			if p.SchemaProxy != nil && isSchemaBoolean(p.SchemaProxy.Schema) {
				continue
			}
			if p.SchemaProxy != nil && (p.SchemaProxy.Schema.Value.Examples != nil || p.SchemaProxy.Schema.Value.Example != nil) {
				continue
			}

			// check if the parameter has any content defined with examples
			if p.Content.Len() > 0 {
				for con := p.Content.First(); con != nil; con = con.Next() {
					v := con.Value()
					if v.Examples != nil && p.Examples.Len() >= 0 {
						// add to seen elements, so when checking schemas we can mark them as good.
						h := p.Value.GoLow().Hash()
						if _, ok := seen[h]; !ok {
							seen[h] = true
						}
						break paramClear
					}
				}
			}

			if p.Value.Examples.Len() <= 0 && isExampleNodeNull([]*yaml.Node{p.Value.Example}) {
				n := p.Value.GoLow().RootNode
				if p.Value.GoLow().KeyNode != nil {
					if p.Value.GoLow().KeyNode.Line == n.Line-1 {
						n = p.Value.GoLow().KeyNode
					}
				}
				results = append(results,
					buildResult(vacuumUtils.SuppliedOrDefault(context.Rule.Message, "parameter is missing `examples` or `example`"),
						p.GenerateJSONPath(),
						n, p))
			} else {
				// add to seen elements, so when checking schemas we can mark them as good.
				h := p.Value.GoLow().Hash()
				if _, ok := seen[h]; !ok {
					seen[h] = true
				}
			}
		}
	}

	if context.DrDocument.Headers != nil {
		for i := range context.DrDocument.Headers {
			h := context.DrDocument.Headers[i]
			if h.SchemaProxy != nil && isSchemaBoolean(h.SchemaProxy.Schema) {
				continue
			}
			if h.SchemaProxy != nil && (h.SchemaProxy.Schema.Value.Examples != nil || h.SchemaProxy.Schema.Value.Example != nil) {
				continue
			}
			if h.Value.Examples.Len() <= 0 && isExampleNodeNull([]*yaml.Node{h.Value.Example}) {
				n := h.Value.GoLow().RootNode
				if h.Value.GoLow().KeyNode != nil {
					if h.Value.GoLow().KeyNode.Line == n.Line-1 {
						n = h.Value.GoLow().KeyNode
					}
				}
				results = append(results,
					buildResult(vacuumUtils.SuppliedOrDefault(context.Rule.Message, "header is missing `examples` or `example`"),
						h.GenerateJSONPath(),
						n, h))
			} else {
				// add to seen elements, so when checking schemas we can mark them as good.
				hs := h.Value.GoLow().Hash()
				if _, ok := seen[hs]; !ok {
					seen[hs] = true
				}
			}
		}
	}

	if context.DrDocument.MediaTypes != nil {
		for i := range context.DrDocument.MediaTypes {
			mt := context.DrDocument.MediaTypes[i]
			if mt.Value.Examples.Len() <= 0 && isExampleNodeNull([]*yaml.Node{mt.Value.Example}) {

				n := mt.Value.GoLow().RootNode
				if mt.Value.GoLow().KeyNode != nil {
					if mt.Value.GoLow().KeyNode.Line == n.Line-1 {
						n = mt.Value.GoLow().KeyNode
					}
				}
				results = append(results,
					buildResult(vacuumUtils.SuppliedOrDefault(context.Rule.Message, "media type is missing `examples` or `example`"),
						mt.GenerateJSONPath(),
						n, mt))
			} else {
				// add to seen elements, so when checking schemas we can mark them as good.
				h := mt.Value.GoLow().Hash()
				if _, ok := seen[h]; !ok {
					seen[h] = true
				}
			}
		}
	}

	if context.DrDocument.Schemas != nil {
		for i := range context.DrDocument.Schemas {
			s := context.DrDocument.Schemas[i]
			if isSchemaBoolean(s) {
				continue
			}
			parentHash := extractHash(s)
			if _, ok := seen[parentHash]; ok {
				continue
			}
			if isExampleNodeNull(s.Value.Examples) && isExampleNodeNull([]*yaml.Node{s.Value.Example}) {
				results = append(results,
					buildResult(vacuumUtils.SuppliedOrDefault(context.Rule.Message, "schema is missing `examples` or `example`"),
						s.GenerateJSONPath(),
						s.Value.ParentProxy.GetSchemaKeyNode(), s))

			}
		}
	}
	return results
}

func extractHash(s *base.Schema) [32]byte {
	if s != nil && s.Parent != nil {
		if p := s.Parent.(base.Foundational).GetParent(); p != nil {
			if v := p.(base.HasValue).GetValue(); v != nil {
				if g, ok := v.(high.GoesLowUntyped); ok {
					return g.GoLowUntyped().(low.Hashable).Hash()
				}
			}
		}
	}
	return [32]byte{}
}

func isSchemaBoolean(schema *base.Schema) bool {
	if slices.Contains(schema.Value.Type, "boolean") {
		return true
	}
	return false
}
