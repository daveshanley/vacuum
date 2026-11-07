// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/testify/require"
)

func TestMissingType_BooleanSchemas(t *testing.T) {
	for _, version := range []string{"3.1.0", "3.2.0"} {
		t.Run(version, func(t *testing.T) {
			source := fmt.Sprintf(`openapi: %s
info: {title: Rule function options, version: '1'}
paths: {}
components:
  schemas:
    Allow: true
    Deny: false
    AllowRef:
      $ref: '#/components/schemas/Allow'
    DenyRef:
      $ref: '#/components/schemas/Deny'
    Untyped: {}
    Action:
      type: object
      properties:
        allow: true
        deny: false
        allowRef:
          $ref: '#/components/schemas/Allow'
        denyRef:
          $ref: '#/components/schemas/Deny'
        allowWithDescription:
          $ref: '#/components/schemas/Allow'
          description: Any function options are accepted.
        denyWithDescription:
          $ref: '#/components/schemas/Deny'
          description: No function options are accepted.
        needsType: {}
`, version)
			document, err := libopenapi.NewDocument([]byte(source))
			require.NoError(t, err)
			defer document.Release()
			built, err := document.BuildV3Model()
			require.NoError(t, err)
			dr := drModel.NewDrDocument(built)
			results := (MissingType{}).RunRule(nil, model.RuleFunctionContext{Rule: &model.Rule{Name: "missing-type"}, Document: document, DrDocument: dr})
			require.Len(t, results, 3, "only ordinary object schemas without a type must be reported")
			foundComponent, foundProperty := false, false
			for _, result := range results {
				component := strings.Contains(result.Path, "schemas['Untyped']")
				property := strings.Contains(result.Path, "properties['needsType']")
				require.True(t, component || property, "unexpected missing type at %s", result.Path)
				foundComponent = foundComponent || component
				foundProperty = foundProperty || property
			}
			require.True(t, foundComponent)
			require.True(t, foundProperty)
		})
	}
}
