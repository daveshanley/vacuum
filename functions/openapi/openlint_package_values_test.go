// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package openapi

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/jsonpath/pkg/jsonpath"
	"github.com/pb33f/jsonpath/pkg/jsonpath/config"
	"github.com/pb33f/testify/require"
)

func TestOpenLintPackageValueChecks(t *testing.T) {
	const spec = `openapi: 3.0.3
schema:
  type: integer
  nullable: true
  enum: [1.0, null, bad, [], {}]
tags:
  - name: first
  - name: first
reference:
  $ref: '#/schema'
  description: unused
  summary: unused`
	var doc yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(spec), &doc))
	for _, tt := range []struct {
		name, selector, given string
		paths, messages       []string
	}{
		{"refSiblings", "$..[?(@property === '$ref')]^", "$.reference",
			[]string{"$.reference.description", "$.reference.summary"},
			[]string{"$ref must not be placed next to any other properties", "$ref must not be placed next to any other properties"}},
		{"uniquenessTags", "$.tags", "$.tags",
			[]string{"$.tags[1].name"}, []string{`"tags" object contains duplicate tag name "first".`}},
		{"typedEnum", "$.schema", "$.schema",
			[]string{"$.schema.enum[2]", "$.schema.enum[3]", "$.schema.enum[4]"},
			[]string{`Enum value "bad" must be "integer,null".`, `Enum value Array[] must be "integer,null".`, `Enum value Object{} must be "integer,null".`}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path, err := jsonpath.NewPath(tt.selector, config.WithSpectralCompatibility(), config.WithLazyContextTracking())
			require.NoError(t, err)
			nodes := path.Query(&doc)
			require.Len(t, nodes, 1)
			fn := OpenLintPackageFunction(tt.name)
			actual := fn.RunRule(nodes, model.RuleFunctionContext{Given: tt.given, ExecutionRoot: &doc, Contract: model.OpenLintV1Draft})
			require.Len(t, actual, len(tt.paths))
			for i, finding := range actual {
				require.Equal(t, tt.paths[i], finding.Path)
				require.Equal(t, tt.messages[i], finding.Message)
				require.Positive(t, finding.StartNode.Line)
			}
		})
	}
}
