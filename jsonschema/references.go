// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package jsonschema

import (
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
)

const (
	dynamicRefKeyword   = "$dynamicRef"
	recursiveRefKeyword = "$recursiveRef"
)

// NewReferenceValidationRule builds the synthetic JSON Schema reference validation rule used by the motor.
func NewReferenceValidationRule() *model.Rule {
	return &model.Rule{
		Name:         "Check JSON Schema references can be resolved correctly",
		Id:           "json-schema-ref-valid",
		Description:  "$ref values must be resolvable and locatable within a local or remote JSON Schema document.",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategorySchemas],
		Type:         "validation",
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "blank",
		},
		HowToFix: "Ensure that all ordinary $ref values are resolvable and locatable within a local or remote JSON Schema document. " +
			"$dynamicRef and $recursiveRef are dynamic-scope keywords and are not resolved as ordinary references.",
	}
}

// IsDynamicScopeResolvingError reports whether a libopenapi resolving error came from a dynamic-scope reference keyword.
func IsDynamicScopeResolvingError(err *index.ResolvingError) bool {
	if err == nil {
		return false
	}
	if hasDynamicScopeKeyword(err.Path) {
		return true
	}
	if err.Node != nil {
		return hasDynamicScopeKeyword(err.Node.Value)
	}
	return false
}

// IsDynamicScopeIndexingError reports whether a libopenapi indexing error came from a dynamic-scope reference keyword.
func IsDynamicScopeIndexingError(err *index.IndexingError) bool {
	if err == nil {
		return false
	}
	if hasDynamicScopeKeyword(err.Path) {
		return true
	}
	if err.Node != nil {
		return hasDynamicScopeKeyword(err.Node.Value)
	}
	return false
}

func hasDynamicScopeKeyword(value string) bool {
	return strings.Contains(value, dynamicRefKeyword) || strings.Contains(value, recursiveRefKeyword)
}
