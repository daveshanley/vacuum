// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package jsonschema

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestIsDynamicScopeResolvingError(t *testing.T) {
	assert.True(t, IsDynamicScopeResolvingError(&index.ResolvingError{Path: "$.$dynamicRef"}))
	assert.True(t, IsDynamicScopeResolvingError(&index.ResolvingError{
		Node: &yaml.Node{Value: "$recursiveRef"},
	}))
	assert.False(t, IsDynamicScopeResolvingError(&index.ResolvingError{Path: "$.$ref"}))
	assert.False(t, IsDynamicScopeResolvingError(nil))
}

func TestIsDynamicScopeIndexingError(t *testing.T) {
	assert.True(t, IsDynamicScopeIndexingError(&index.IndexingError{Path: "$.$dynamicRef"}))
	assert.True(t, IsDynamicScopeIndexingError(&index.IndexingError{
		Node: &yaml.Node{Value: "$recursiveRef"},
	}))
	assert.False(t, IsDynamicScopeIndexingError(&index.IndexingError{Path: "$.$ref"}))
	assert.False(t, IsDynamicScopeIndexingError(nil))
}

func TestNewReferenceValidationRule(t *testing.T) {
	rule := NewReferenceValidationRule()
	assert.Equal(t, "json-schema-ref-valid", rule.Id)
	assert.Contains(t, rule.HowToFix, "$dynamicRef")
	action, ok := rule.Then.(model.RuleAction)
	assert.True(t, ok)
	assert.Equal(t, "blank", action.Function)
}
