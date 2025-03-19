// Copyright 2024 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package openapi

import (
	"context"
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/base"
	"github.com/pb33f/libopenapi/datamodel/high"
	"github.com/pb33f/libopenapi/index"
	"gopkg.in/yaml.v3"
	"slices"
	"strings"
)

// ExamplesMissing will check anything that can have an example, has one.
type ExamplesMissing struct {
}

// GetCategory returns the category of the ExamplesMissing rule.
func (em ExamplesMissing) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the ComponentDescription rule.
func (em ExamplesMissing) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "oasExampleMissing"}
}

// RunRule will execute the ComponentDescription rule, based on supplied context and a supplied []*yaml.Node slice.
func (em ExamplesMissing) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	// create a string buffer for caching seen schemas
	var buf strings.Builder

	buildResult := func(message, path string, node, valueNode *yaml.Node, component base.AcceptsRuleResults) model.RuleFunctionResult {

		origin := context.Document.GetRolodex().FindNodeOriginWithValue(node, valueNode, nil, "")
		if origin == nil {
			origin = context.Document.GetRolodex().FindNodeOrigin(valueNode)
		}
		result := model.RuleFunctionResult{
			Message:   message,
			StartNode: node,
			EndNode:   vacuumUtils.BuildEndNode(node),
			Path:      path,
			Rule:      context.Rule,
			Origin:    origin,
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

	seen := make(map[string]bool)

	if context.DrDocument.Parameters != nil {
	paramClear:
		for i := range context.DrDocument.Parameters {
			p := context.DrDocument.Parameters[i]
			if p.SchemaProxy != nil && isSchemaBoolean(p.SchemaProxy.Schema) {
				continue
			}
			if p.SchemaProxy != nil && p.SchemaProxy.Schema != nil && p.SchemaProxy.Schema.Value != nil && (p.SchemaProxy.Schema.Value.Examples != nil || p.SchemaProxy.Schema.Value.Example != nil) {
				continue
			}
			if p.SchemaProxy != nil && (isSchemaBoolean(p.SchemaProxy.Schema) ||
				isSchemaEnum(p.SchemaProxy.Schema) || isSchemaNumber(p.SchemaProxy.Schema) || isSchemaString(p.SchemaProxy.Schema)) {
				continue
			}

			if (p.SchemaProxy != nil && p.SchemaProxy.Schema != nil && p.SchemaProxy.Schema.Value != nil) &&
				(p.SchemaProxy.Schema.Value.Const != nil || p.SchemaProxy.Schema.Value.Default != nil) {
				continue
			}

			if p.SchemaProxy != nil {
				if len(p.SchemaProxy.Schema.Value.Type) <= 0 {
					continue
				}

				if p.SchemaProxy.Schema.Value.Items != nil && p.SchemaProxy.Schema.Value.Items.IsA() && p.SchemaProxy.Schema.Value.Items.A != nil {
					if len(p.SchemaProxy.Schema.Value.Items.A.Schema().Enum) > 0 {
						continue
					}
				}
			}

			// check if the parameter has any content defined with examples
			if p.Content != nil && p.Content.Len() > 0 {
				for con := p.Content.First(); con != nil; con = con.Next() {
					v := con.Value()
					if v.Examples != nil && (p.Examples == nil || p.Examples.Len() >= 0) {
						// add to seen elements, so when checking schemas we can mark them as good.
						buf.WriteString(fmt.Sprintf("%s:%d:%d", p.Value.GoLow().GetIndex().GetSpecAbsolutePath(),
							p.Value.GoLow().KeyNode.Line, p.Value.GoLow().KeyNode.Column))
						if _, ok := seen[buf.String()]; !ok {
							seen[buf.String()] = true
						}
						buf.Reset()
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
						n, p.GetValueNode(), p))
			} else {
				// add to seen elements, so when checking schemas we can mark them as good.
				buf.WriteString(fmt.Sprintf("%s:%d:%d", p.Value.GoLow().GetIndex().GetSpecAbsolutePath(),
					p.Value.GoLow().KeyNode.Line, p.Value.GoLow().KeyNode.Column))
				if _, ok := seen[buf.String()]; !ok {
					seen[buf.String()] = true
				}
				buf.Reset()
			}
		}
	}

	if context.DrDocument.Headers != nil {
		for i := range context.DrDocument.Headers {
			h := context.DrDocument.Headers[i]
			if h == nil || h.Schema == nil {
				continue
			}
			if h.Schema.Schema != nil && isSchemaBoolean(h.Schema.Schema) {
				continue
			}
			if h.Schema != nil && (h.Schema.Schema.Value.Examples != nil || h.Schema.Schema.Value.Example != nil) {
				continue
			}

			if h.Schema != nil && isSchemaBoolean(h.Schema.Schema) ||
				isSchemaEnum(h.Schema.Schema) || isSchemaNumber(h.Schema.Schema) || isSchemaString(h.Schema.Schema) {
				continue
			}

			if (h.Schema != nil && h.Schema.Schema != nil && h.Schema.Schema.Value != nil) &&
				(h.Schema.Schema.Value.Const != nil || h.Schema.Schema.Value.Default != nil) {
				continue
			}

			if h.Schema != nil {
				if len(h.Schema.Schema.Value.Type) <= 0 {
					continue
				}

				if h.Schema.Schema.Value.Items != nil && h.Schema.Schema.Value.Items.IsA() && h.Schema.Schema.Value.Items.A != nil {
					if len(h.Schema.Schema.Value.Items.A.Schema().Enum) > 0 {
						continue
					}
				}
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
						n, h.GetValueNode(), h))
			} else {
				buf.WriteString(fmt.Sprintf("%s:%d:%d", h.Value.GoLow().GetIndex().GetSpecAbsolutePath(),
					h.Value.GoLow().KeyNode.Line, h.Value.GoLow().KeyNode.Column))
				if _, ok := seen[buf.String()]; !ok {
					seen[buf.String()] = true
				}
				buf.Reset()
			}
		}
	}

	if context.DrDocument.MediaTypes != nil {
		for i := range context.DrDocument.MediaTypes {
			mt := context.DrDocument.MediaTypes[i]

			if mt.SchemaProxy != nil && isSchemaBoolean(mt.SchemaProxy.Schema) {
				continue
			}
			if mt.SchemaProxy != nil &&
				mt.SchemaProxy.Schema != nil &&
				mt.SchemaProxy.Schema.Value != nil &&
				(mt.SchemaProxy.Schema.Value.Examples != nil || mt.SchemaProxy.Schema.Value.Example != nil) {
				continue
			}

			if mt.SchemaProxy != nil && mt.SchemaProxy.Schema != nil && (isSchemaBoolean(mt.SchemaProxy.Schema) ||
				isSchemaEnum(mt.SchemaProxy.Schema) || isSchemaNumber(mt.SchemaProxy.Schema) || isSchemaString(mt.SchemaProxy.Schema)) {
				continue
			}

			if (mt.SchemaProxy != nil && mt.SchemaProxy.Schema != nil && mt.SchemaProxy.Schema.Value != nil) &&
				(mt.SchemaProxy.Schema.Value.Const != nil || mt.SchemaProxy.Schema.Value.Default != nil) {
				continue
			}

			propErr := false
			hasProps := false
			if mt.SchemaProxy != nil &&
				mt.SchemaProxy.Schema != nil &&
				mt.SchemaProxy.Schema.Properties != nil &&
				mt.SchemaProxy.Schema.Properties.Len() > 0 {

				hasProps = true
				var prop *base.Schema
				var propName string
				for k, v := range mt.SchemaProxy.Schema.Properties.FromOldest() {
					if !checkProps(v.Schema) {
						propErr = true
						prop = v.Schema
						propName = k
						break
					}
				}
				if propErr {
					path := prop.GenerateJSONPath()
					results = append(results,
						buildResult(vacuumUtils.SuppliedOrDefault(context.Rule.Message,
							fmt.Sprintf("media type schema property `%s` is missing `examples` or `example`", propName)),
							path,
							prop.KeyNode, mt.ValueNode, mt))
				}
			}

			buf.WriteString(fmt.Sprintf("%s:%d:%d", mt.Value.GoLow().GetIndex().GetSpecAbsolutePath(),
				mt.Value.GoLow().KeyNode.Line, mt.Value.GoLow().KeyNode.Column))
			if _, ok := seen[buf.String()]; !ok {
				seen[buf.String()] = true
			}
			buf.Reset()

			if hasProps && propErr && mt.Value.Examples.Len() <= 0 && isExampleNodeNull([]*yaml.Node{mt.Value.Example}) {

				n := mt.Value.GoLow().RootNode
				if mt.Value.GoLow().KeyNode != nil {
					if mt.Value.GoLow().KeyNode.Line == n.Line-1 {
						n = mt.Value.GoLow().KeyNode
					}
				}
				path := mt.GenerateJSONPath()
				results = append(results,
					buildResult(vacuumUtils.SuppliedOrDefault(context.Rule.Message, "media type is missing `examples` or `example`"),
						path,
						n, mt.ValueNode, mt))
				continue
			}

			if !hasProps && mt.Value.Examples.Len() <= 0 && isExampleNodeNull([]*yaml.Node{mt.Value.Example}) {

				n := mt.Value.GoLow().RootNode
				if mt.Value.GoLow().KeyNode != nil {
					if mt.Value.GoLow().KeyNode.Line == n.Line-1 {
						n = mt.Value.GoLow().KeyNode
					}
				}
				path := mt.GenerateJSONPath()
				results = append(results,
					buildResult(vacuumUtils.SuppliedOrDefault(context.Rule.Message, "media type is missing `examples` or `example`"),
						path,
						n, mt.ValueNode, mt))
				continue
			}
		}
	}

	if context.DrDocument.Schemas != nil {
		for i := range context.DrDocument.Schemas {
			s := context.DrDocument.Schemas[i]
			if isSchemaBoolean(s) || isSchemaEnum(s) || isSchemaNumber(s) || isSchemaString(s) {
				continue
			}

			if (s.Value != nil) &&
				(s.Value.Const != nil || s.Value.Default != nil) {
				continue
			}

			if len(s.Value.Type) <= 0 {
				continue
			}

			if s.Value.Items != nil && s.Value.Items.IsA() && s.Value.Items.A != nil {
				if s.Value.Items.A.Schema() != nil && len(s.Value.Items.A.Schema().Enum) > 0 {
					continue
				}
			}

			hash := extractHash(s)
			if _, ok := seen[hash]; ok {
				continue

			}
			seen[hash] = true

			// check if this schema has a parent, and if the parent is a schema
			if s.Parent != nil {
				if checkParent(s.Parent, 0) {
					continue
				}
			}

			propErr := false
			hasProps := false
			if s.Properties != nil && s.Properties.Len() > 0 {
				hasProps = true
				var prop *base.Schema
				var propName string
				for k, v := range s.Properties.FromOldest() {
					if !checkProps(v.Schema) {
						propErr = true
						prop = v.Schema
						propName = k
						break
					}
				}
				if propErr {
					path := prop.GenerateJSONPath()
					results = append(results,
						buildResult(vacuumUtils.SuppliedOrDefault(context.Rule.Message,
							fmt.Sprintf("schema property `%s` is missing `examples` or `example`", propName)),
							path,
							prop.KeyNode, s.ValueNode, s))
				}
			}

			if hasProps && propErr && isExampleNodeNull(s.Value.Examples) && isExampleNodeNull([]*yaml.Node{s.Value.Example}) {
				results = append(results,
					buildResult(vacuumUtils.SuppliedOrDefault(context.Rule.Message, "schema is missing `examples` or `example`"),
						s.GenerateJSONPath(),
						s.Value.ParentProxy.GetSchemaKeyNode(), s.Value.ParentProxy.GetValueNode(), s))
			}

			if !hasProps && isExampleNodeNull(s.Value.Examples) && isExampleNodeNull([]*yaml.Node{s.Value.Example}) {
				results = append(results,
					buildResult(vacuumUtils.SuppliedOrDefault(context.Rule.Message, "schema is missing `examples` or `example`"),
						s.GenerateJSONPath(),
						s.Value.ParentProxy.GetSchemaKeyNode(), s.Value.ParentProxy.GetValueNode(), s))
			}
		}
	}
	seen = nil
	buf.Reset()
	return results
}

func checkProps(s *base.Schema) bool {
	if len(s.Value.Examples) > 0 {
		return true
	}
	if s.Value.Example != nil {
		return true
	}
	if s.Value.Properties != nil && s.Value.Properties.Len() > 0 {
		for _, p := range s.Properties.FromOldest() {
			return checkProps(p.Schema)
		}
	}
	return false
}
func checkParent(s any, depth int) bool {
	if depth > 10 {
		return false
	}
	if sp, ok := s.(*base.SchemaProxy); ok {

		// check the parent schema for an example
		if sp.Parent != nil {

			if pp, kk := sp.Parent.(*base.Schema); kk {
				if pp.Value != nil {
					if pp.Value.Example != nil || pp.Value.Examples != nil {
						return true
					}

					if isSchemaBoolean(pp) || isSchemaEnum(pp) || isSchemaNumber(pp) || isSchemaString(pp) {
						return true
					}

					if len(pp.Value.Type) <= 0 {
						return true
					}

					if pp.Value.Items != nil && pp.Value.Items.IsA() && pp.Value.Items.A != nil {
						if len(pp.Value.Items.A.Schema().Enum) > 0 {
							return true
						}
					}
					depth++
					return checkParent(pp, depth)
				}
			}
		}
	}
	return false
}

type contextualPosition interface {
	GetIndex() *index.SpecIndex
	GetContext() context.Context
	GetKeyNode() *yaml.Node
}

func extractHash(s *base.Schema) string {
	if s != nil && s.Parent != nil {
		if p := s.Parent.(base.Foundational).GetParent(); p != nil {
			// check if p implements HasValue
			if hv, ok := p.(base.HasValue); ok {
				// check if hv.Value implements GoesLowUntyped
				if gl, ko := hv.GetValue().(high.GoesLowUntyped); ko {
					// check if gl.GoesLowUntyped() implements contextualPosition
					if cp, kk := gl.GoLowUntyped().(contextualPosition); kk {
						return fmt.Sprintf("%s:%d:%d", cp.GetIndex().GetSpecAbsolutePath(),
							cp.GetKeyNode().Line, cp.GetKeyNode().Column)
					}
				}
			}
		}
	}
	return ""
}

func isSchemaBoolean(schema *base.Schema) bool {
	if schema == nil || schema.Value == nil {
		return false
	}
	if slices.Contains(schema.Value.Type, "boolean") {
		return true
	}
	return false
}

func isSchemaString(schema *base.Schema) bool {
	if schema == nil || schema.Value == nil {
		return false
	}
	if slices.Contains(schema.Value.Type, "string") {
		return true
	}
	return false
}

func isSchemaNumber(schema *base.Schema) bool {
	if schema == nil || schema.Value == nil {
		return false
	}
	if slices.Contains(schema.Value.Type, "number") {
		return true
	}
	if slices.Contains(schema.Value.Type, "integer") {
		return true
	}
	return false
}

func isSchemaEnum(schema *base.Schema) bool {
	if schema == nil || schema.Value == nil {
		return false
	}
	if len(schema.Value.Enum) > 0 {
		return true
	}
	return false
}
