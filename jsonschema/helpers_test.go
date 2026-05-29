// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package jsonschema

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestDetectDialectDefaultsTo202012(t *testing.T) {
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("type: object\n"), &node))

	dialect := DetectDialect(&node)
	assert.Equal(t, model.JSONSchemaDraft2020, dialect.Format)
	assert.Equal(t, SchemaURL2020, dialect.URL)
}

func TestValidateAgainstMetaschemaMapsPointerToYAMLNode(t *testing.T) {
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(`$schema: https://json-schema.org/draft/2020-12/schema
type: nope
`), &node))

	issues, err := ValidateAgainstMetaschema(&node)
	require.NoError(t, err)
	require.NotEmpty(t, issues)

	var typeIssue *ValidationIssue
	for i := range issues {
		if issues[i].Path == "$.type" {
			typeIssue = &issues[i]
			break
		}
	}
	require.NotNil(t, typeIssue)
	assert.Equal(t, "#/type", typeIssue.Pointer)
	assert.Equal(t, 2, typeIssue.Node.Line)
	assert.Equal(t, 7, typeIssue.Node.Column)
}
