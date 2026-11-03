package openapi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/pb33f/testify/assert"
)

func TestTypedEnum_GetSchema(t *testing.T) {
	def := TypedEnum{}
	assert.Equal(t, "typedEnum", def.GetSchema().Name)
}

func TestTypedEnum_RunRule(t *testing.T) {
	def := TypedEnum{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestTypedEnum_RunRule_SuccessCheck(t *testing.T) {
	yml := `paths:
  /pizza/:
    parameters:
      - in: query
        name: party
        schema:
          type: string
          enum: [big, small]
  /cake/:
    parameters:
      - in: query
        name: icecream
        schema:
          type: string
          enum: [lots, little]        
components:
  schemas:
    YesNo:
      type: string
      enum: [yes, no]`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "typed_enum", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := TypedEnum{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestTypedEnum_RunRule_31NullableEnum_SuccessCheck(t *testing.T) {
	yml := `paths:
  /pizza/:
    parameters:
      - in: query
        name: party
        schema:
          type: [string, null]
          enum: [big, small, null]`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "typed_enum", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := TypedEnum{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestTypedEnums_RunRule_ThreeValue_WrongType(t *testing.T) {
	yml := `paths:
  /pizza/:
    parameters:
      - in: query
        name: party
        schema:
          type: string
          enum: [big, 1]
  /cake/:
    parameters:
      - in: query
        name: icecream
        schema:
          type: string
          enum: [0.2, little]        
components:
  schemas:
    YesNo:
      type: string
      enum: [yes, true]
    TooManyTypes:
      type: [string, integer, null]
      enum: [hi, 1, null]`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "typed_enum", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := TypedEnum{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 5)
}

// enums that live in a file reached through a relative $ref sit in the rolodex index for that
// file, not in the root index, so they still need to be checked.
func TestTypedEnum_RunRule_MultiFile(t *testing.T) {
	tmpDir := t.TempDir()

	mainSpec := `openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /items:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: './Item.yaml'
`

	itemSpec := `type: object
properties:
  kind:
    type: string
    enum: [a, 1]
`

	assert.NoError(t, os.WriteFile(filepath.Join(tmpDir, "main.yaml"), []byte(mainSpec), 0644))
	assert.NoError(t, os.WriteFile(filepath.Join(tmpDir, "Item.yaml"), []byte(itemSpec), 0644))

	config := datamodel.NewDocumentConfiguration()
	config.BasePath = tmpDir
	config.AllowFileReferences = true

	document, err := libopenapi.NewDocumentWithConfiguration([]byte(mainSpec), config)
	assert.NoError(t, err)

	m, errs := document.BuildV3Model()
	assert.NoError(t, errs)

	rule := buildOpenApiTestRuleAction("$", "typed_enum", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.Rule = &rule
	ctx.Index = m.Index

	nodes, _ := utils.FindNodes([]byte(mainSpec), "$")

	def := TypedEnum{}
	res := def.RunRule(nodes, ctx)

	if assert.Len(t, res, 1) {
		assert.Contains(t, res[0].Message, "enum type mismatch")
		if assert.NotNil(t, res[0].Origin) {
			assert.True(t, strings.HasSuffix(res[0].Origin.AbsoluteLocation, "Item.yaml"))
		}
	}
}
