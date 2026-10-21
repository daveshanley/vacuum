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

func TestMissingType_ReferenceSiblings(t *testing.T) {
	for _, sibling := range []string{"description: identifier", "deprecated: true", "default: example", "maxLength: 20"} {
		for _, target := range []struct {
			name, schema string
			missing      bool
		}{
			{"typed", "type: string", false},
			{"implied", "properties: {id: {type: string}}", false},
			{"enum", "enum: [one, two]", false},
			{"untyped", "description: missing type", true},
			{"reference chain", "$ref: '#/components/schemas/End'\n      description: intermediate\n    End:\n      type: string", false},
		} {
			t.Run(sibling+"/"+target.name, func(t *testing.T) {
				source := fmt.Sprintf(`openapi: 3.1.0
info: {title: Reference siblings, version: '1'}
paths: {}
components:
  schemas:
    Target:
      %s
    Alias:
      $ref: '#/components/schemas/Target'
      %s
    Container:
      type: object
      properties:
        value:
          $ref: '#/components/schemas/Target'
          %s
`, target.schema, sibling, sibling)
				document, err := libopenapi.NewDocument([]byte(source))
				require.NoError(t, err)
				m, err := document.BuildV3Model()
				require.NoError(t, err)
				dr := drModel.NewDrDocument(m)
				results := (MissingType{}).RunRule(nil, model.RuleFunctionContext{
					Rule: &model.Rule{Name: "missing-type"}, Document: document, DrDocument: dr,
				})
				if !target.missing {
					require.Empty(t, results)
					return
				}
				require.NotEmpty(t, results)
				foundProperty, foundTarget := false, false
				for _, result := range results {
					require.NotNil(t, result.StartNode)
					require.NotEmpty(t, result.Paths)
					foundProperty = foundProperty || strings.Contains(result.Path, "properties['value']")
					foundTarget = foundTarget || strings.Contains(result.Path, "schemas['Target']")
				}
				require.True(t, foundProperty, "untyped referenced property must still be diagnosed")
				require.True(t, foundTarget, "untyped reference target must still be diagnosed")
			})
		}
	}
}

func TestMissingType_UnresolvedReferenceSiblings(t *testing.T) {
	document, err := libopenapi.NewDocument([]byte(`openapi: 3.1.0
info: {title: Broken reference, version: '1'}
paths: {}
components:
  schemas:
    Container:
      type: object
      properties:
        value:
          $ref: '#/components/schemas/Missing'
          description: unresolved
`))
	require.NoError(t, err)
	_, err = document.BuildV3Model()
	require.Error(t, err)
	require.Contains(t, err.Error(), "Missing")
}
