// Copyright 2023 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
	"net/url"
	"slices"
	"strings"
	"sync"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/index"
)

// LocateSchemaPropertyPaths is a wrapper for the utils version, kept for backwards compatibility
func LocateSchemaPropertyPaths(
	context model.RuleFunctionContext,
	schema *v3.Schema,
	keyNode *yaml.Node,
	valueNode *yaml.Node,
) (primaryPath string, allPaths []string) {
	return vacuumUtils.LocateSchemaPropertyPaths(context, schema, keyNode, valueNode)
}

// explicitInsecureScheme leaves relative and malformed URLs to their owning checks.
func explicitInsecureScheme(raw string) bool {
	parsed, err := url.Parse(raw)
	return err == nil && parsed.Scheme != "" && !strings.EqualFold(parsed.Scheme, "https")
}

func schemaHasFiniteValues(schema *base.Schema, specInfo *datamodel.SpecInfo) bool {
	return (!vacuumUtils.IsOAS30(specInfo) && schema.Const != nil) || len(schema.Enum) > 0
}

func schemaIsObject(schema *base.Schema) bool {
	return slices.Contains(schema.Type, "object") || (len(schema.Type) == 0 &&
		(schema.Properties != nil || schema.AdditionalProperties != nil || schema.UnevaluatedProperties != nil))
}

func schemaObjectNodes(schema *base.Schema) (*yaml.Node, *yaml.Node) {
	low := schema.GoLow()
	if low.Type.KeyNode != nil {
		return low.Type.KeyNode, low.Type.ValueNode
	}
	if low.Properties.KeyNode != nil {
		return low.Properties.KeyNode, low.Properties.ValueNode
	}
	if low.AdditionalProperties.KeyNode != nil {
		return low.AdditionalProperties.KeyNode, low.AdditionalProperties.ValueNode
	}
	return low.UnevaluatedProperties.KeyNode, low.UnevaluatedProperties.ValueNode
}

type composedOnlySchemasKey struct{ index *index.SpecIndex }

// Cache composition-only reference targets for this document's rule execution.
func composedOnlySchemas(context model.RuleFunctionContext) map[*yaml.Node]bool {
	if context.Index == nil {
		return nil
	}
	build := func() map[*yaml.Node]bool {
		composedOnly := make(map[*yaml.Node]bool)
		indexes := []*index.SpecIndex{context.Index}
		references := context.Index.GetAllReferences()
		mapped := context.Index.GetMappedReferences()
		if rolodex := context.Index.GetRolodex(); rolodex != nil {
			indexes = append(indexes, rolodex.GetIndexes()...)
			references = rolodex.GetAllReferences()
			mapped = rolodex.GetAllMappedReferences()
		}
		for _, idx := range indexes {
			for _, ref := range idx.GetPolyReferences() {
				if references[ref.FullDefinition] == nil {
					if target := mapped[ref.FullDefinition]; target != nil {
						composedOnly[target.Node] = true
					}
				}
			}
		}
		return composedOnly
	}
	if context.SchemaPathCache == nil {
		return build()
	}
	key := composedOnlySchemasKey{context.Index}
	if cached, ok := context.SchemaPathCache.Load(key); ok {
		return cached.(func() map[*yaml.Node]bool)()
	}
	cached, _ := context.SchemaPathCache.LoadOrStore(key, sync.OnceValue(build))
	return cached.(func() map[*yaml.Node]bool)()
}
