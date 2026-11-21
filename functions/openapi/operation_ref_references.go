// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

// OperationRefReferences warns when a Link operationRef reaches its operation only by passing
// through a $ref. Strict JSON Pointer evaluation does not follow references, so other tools may
// not resolve the link.
type OperationRefReferences struct{}

// GetSchema identifies the operationRef reference rule function.
func (OperationRefReferences) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "oasOperationRefReferences"}
}

// GetCategory returns the OpenAPI function category.
func (OperationRefReferences) GetCategory() string { return model.FunctionCategoryOpenAPI }

// RunRule reports each resolving operationRef that passes through a local reference, with the direct pointer.
func (OperationRefReferences) RunRule(nodes []*yaml.Node, ctx model.RuleFunctionContext) []model.RuleFunctionResult {
	return runDocumentCheck(nodes, ctx, func(doc *v3.Document, ctx model.RuleFunctionContext, add reportRelationship) {
		for _, ref := range localOperationRefs(doc, ctx.SpecInfo.RootNode) {
			if ref.resolved && ref.target.Direct != "" {
				add(ref.field, ref.pointer, fmt.Sprintf("link operationRef %q passes through a $ref, which JSON Pointer does not follow; use %q", ref.field.Value, ref.target.Direct))
			}
		}
	})
}
