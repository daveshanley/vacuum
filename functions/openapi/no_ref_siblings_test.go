package openapi

import (
	"net/url"
	"testing"

	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestNoRefSiblings_GetSchema(t *testing.T) {
	def := NoRefSiblings{}
	assert.Equal(t, "refSiblings", def.GetSchema().Name)
}

func TestNoRefSiblings_RunRule(t *testing.T) {
	def := NoRefSiblings{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestNoRefSiblings_RunRule_Fail(t *testing.T) {

	yml := `paths:
  /nice/{rice}:
    requestBody:
      content:
        application/json:
          schema:
            description: this is the wrong place this this buddy.
            $ref: '#/components/schemas/Rice'
  /hot/{dog}:
    requestBody:
      content:
        application/json:
          schema:
            description: still the wrong place for this.
            $ref: '#/components/schemas/Dog'`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)
	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "no_ref_siblings", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := NoRefSiblings{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 2)

}

// TestNoRefSiblings_RunRule_Components tests the NoRefSiblings rule
// This applies to OAS 2.0 and OAS 3.0.x (not 3.1+)
// For OAS 3.1+, use OASNoRefSiblings which always returns empty
func TestNoRefSiblings_RunRule_Components(t *testing.T) {

	// Note: No openapi/swagger version specified - this tests the raw function behavior
	// In production, this would only apply to specs < OAS 3.1 due to format filtering
	yml := `components:
  schemas:
    Beer:
      description: nice
      $ref: '#/components/Yum'
    Bottle:
      type: string
    Cake:
      $ref: '#/components/Sugar'`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "no_ref_siblings", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := NoRefSiblings{}
	res := def.RunRule(nodes, ctx)

	// Beer has description + $ref = 1 error
	// Cake has only $ref = no error
	assert.Len(t, res, 1)

}

// TestNoRefSiblings_RunRule_Components_OAS31 tests that OAS 3.1 allows $ref siblings in Schema Objects
func TestNoRefSiblings_RunRule_Components_OAS31(t *testing.T) {

	yml := `openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Beer:
      description: nice
      $ref: '#/components/schemas/Yum'
    Bottle:
      type: string
    Cake:
      $ref: '#/components/schemas/Sugar'`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "oas_no_ref_siblings", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	// Use OASNoRefSiblings for OAS 3.1
	def := OASNoRefSiblings{}
	res := def.RunRule(nodes, ctx)

	// In OAS 3.1, Schema Objects CAN have $ref with siblings
	// OASNoRefSiblings always returns empty because libopenapi handles this
	assert.Len(t, res, 0)

}

func TestNoRefSiblings_RunRule_Parameters(t *testing.T) {

	yml := `parameters:
  testParam:
    $ref: '#/parameters/oldParam'
  oldParam:
    in: query
    description: old
  wrongParam:
    description: I should not be here
    $ref: '#/parameters/oldParam'  `

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "no_ref_siblings", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := NoRefSiblings{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)

}

func TestNoRefSiblings_RunRule_Definitions(t *testing.T) {

	yml := `definitions:
  test:
    $ref: '#/definitions/old'
  old:
    type: object
    description: old
  wrong:
    description: I should not be here
    $ref: '#/definitions/oldParam'  `

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "no_ref_siblings", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := NoRefSiblings{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)

}

func TestNoRefSiblings_RunRule_Success(t *testing.T) {

	yml := `paths:
  /nice/{rice}:
    requestBody:
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Rice'
  /hot/{dog}:
    requestBody:
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Dog'`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "no_ref_siblings", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := NoRefSiblings{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)

}

func TestNoRefSiblings_RunRule_Fail_Single(t *testing.T) {

	yml := `paths:
  /nice/{rice}:
    requestBody:
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Rice'
  /hot/{dog}:
    requestBody:
      content:
        application/json:
          schema:
            description: still the wrong place for this.
            $ref: '#/components/schemas/Dog'`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "no_ref_siblings", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := NoRefSiblings{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)

}

func TestNoRefSiblings_RunRule_MultiFile(t *testing.T) {
	rootYML := `
openapi: 3.0.0
info: {title: api, version: 1.0.0}
paths:
  /item:
    get:
      responses:
        '200':
          content:
            application/json:
              schema:
                $ref: './child.yaml#/components/schemas/External'
components:
  schemas:
    RootBad:
      description: wrong
      $ref: '#/components/schemas/Ok'
    Ok: {type: object}`

	childYML := `
components:
  schemas:
    ExternalBad:
      description: wrong
      $ref: '#/components/schemas/Ok2'
    External: {type: object}
    Ok2: {type: object}`

	var rootNode, childNode yaml.Node
	_ = yaml.Unmarshal([]byte(rootYML), &rootNode)
	_ = yaml.Unmarshal([]byte(childYML), &childNode)

	childCfg := index.CreateOpenAPIIndexConfig()
	childCfg.BaseURL, _ = url.Parse("child.yaml")
	childCfg.AllowFileLookup = false
	childCfg.AllowRemoteLookup = false
	childIdx := index.NewSpecIndexWithConfig(&childNode, childCfg)

	rolodex := index.NewRolodex(childCfg)
	rolodex.AddIndex(childIdx)

	rootCfg := index.CreateOpenAPIIndexConfig()
	rootCfg.Rolodex = rolodex
	rootCfg.AllowFileLookup = false
	rootCfg.AllowRemoteLookup = false
	rootIdx := index.NewSpecIndexWithConfig(&rootNode, rootCfg)

	rule := buildOpenApiTestRuleAction("$", "no_ref_siblings_multi_file", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Index = rootIdx

	nodes, _ := utils.FindNodes([]byte(rootYML), "$")

	var def NoRefSiblings
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 2)
}

// TestNoRefSiblings_Issue750_OAS31_SchemaObject_Valid tests that Schema Objects
// CAN have $ref with siblings in OAS 3.1 (this is valid per JSON Schema Draft 2020-12)
func TestNoRefSiblings_Issue750_OAS31_SchemaObject_Valid(t *testing.T) {
	// This is the exact scenario from issue #750
	yml := `openapi: 3.1.1
info:
  title: Test for Issue 750
  version: 1.0.0
components:
  schemas:
    Referenced:
      type: object
      properties:
        id:
          type: string
    Example:
      $ref: "#/components/schemas/Referenced"
      readOnly: true`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "no_ref_siblings", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	// Build the document model to get spec info
	doc, err := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, err)

	info := doc.GetSpecInfo()
	assert.Equal(t, "3.1.1", info.Version)

	// Use OASNoRefSiblings for OAS 3.1 (the correct rule for this version)
	def := OASNoRefSiblings{}
	res := def.RunRule(nodes, ctx)

	// In OAS 3.1, OASNoRefSiblings always returns empty because libopenapi handles this
	// Schema Objects CAN have $ref with siblings per JSON Schema Draft 2020-12
	assert.Len(t, res, 0, "OAS 3.1 Schema Objects should allow $ref with siblings like readOnly")
}

// TestNoRefSiblings_Issue750_OAS31_MultipleValidSiblings tests Schema Objects
// with multiple valid sibling properties alongside $ref
func TestNoRefSiblings_Issue750_OAS31_MultipleValidSiblings(t *testing.T) {
	yml := `openapi: 3.1.0
info:
  title: Multiple Siblings Test
  version: 1.0.0
components:
  schemas:
    Base:
      type: object
      properties:
        id:
          type: string
    Extended:
      $ref: "#/components/schemas/Base"
      readOnly: true
      description: Extended schema with multiple constraints
      maxLength: 100
      nullable: true
      deprecated: true`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "no_ref_siblings", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	// Use OASNoRefSiblings for OAS 3.1
	def := OASNoRefSiblings{}
	res := def.RunRule(nodes, ctx)

	// All these siblings are valid JSON Schema keywords in OAS 3.1
	assert.Len(t, res, 0, "OAS 3.1 should allow multiple valid siblings with $ref in Schema Objects")
}

// TestNoRefSiblings_OAS30_vs_OAS31_Comparison tests that the same spec
// is handled differently in OAS 3.0 vs OAS 3.1
func TestNoRefSiblings_OAS30_vs_OAS31_Comparison(t *testing.T) {
	// OAS 3.0 - siblings should be flagged
	yml30 := `openapi: 3.0.3
info:
  title: OAS 3.0 Test
  version: 1.0.0
components:
  schemas:
    Example:
      description: This is invalid in OAS 3.0
      $ref: '#/components/schemas/Base'`

	// OAS 3.1 - same structure should be valid
	yml31 := `openapi: 3.1.0
info:
  title: OAS 3.1 Test
  version: 1.0.0
components:
  schemas:
    Example:
      description: This is valid in OAS 3.1
      $ref: '#/components/schemas/Base'`

	path := "$"

	// Test OAS 3.0
	var rootNode30 yaml.Node
	_ = yaml.Unmarshal([]byte(yml30), &rootNode30)
	nodes30, _ := utils.FindNodes([]byte(yml30), path)
	rule30 := buildOpenApiTestRuleAction(path, "no_ref_siblings_30", "", nil)
	ctx30 := buildOpenApiTestContext(model.CastToRuleAction(rule30.Then), nil)
	config30 := index.CreateOpenAPIIndexConfig()
	ctx30.Index = index.NewSpecIndexWithConfig(&rootNode30, config30)

	def30 := NoRefSiblings{}
	res30 := def30.RunRule(nodes30, ctx30)

	// OAS 3.0 should report error
	assert.Greater(t, len(res30), 0, "OAS 3.0 should flag $ref siblings as errors")

	// Test OAS 3.1 - use OASNoRefSiblings (correct rule for 3.1)
	var rootNode31 yaml.Node
	_ = yaml.Unmarshal([]byte(yml31), &rootNode31)
	nodes31, _ := utils.FindNodes([]byte(yml31), path)
	rule31 := buildOpenApiTestRuleAction(path, "no_ref_siblings_31", "", nil)
	ctx31 := buildOpenApiTestContext(model.CastToRuleAction(rule31.Then), nil)
	config31 := index.CreateOpenAPIIndexConfig()
	ctx31.Index = index.NewSpecIndexWithConfig(&rootNode31, config31)

	def31 := OASNoRefSiblings{}
	res31 := def31.RunRule(nodes31, ctx31)

	// OAS 3.1 should NOT report error
	assert.Len(t, res31, 0, "OAS 3.1 should allow $ref siblings in Schema Objects")
}

// TestOASNoRefSiblings_AlwaysReturnsEmpty tests that the OAS3.1-specific rule
// always returns empty (because libopenapi handles this)
func TestOASNoRefSiblings_AlwaysReturnsEmpty(t *testing.T) {
	yml := `openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Example:
      $ref: "#/components/schemas/Base"
      readOnly: true
      description: Any siblings
      custom: property`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "oas_ref_siblings", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := OASNoRefSiblings{}
	res := def.RunRule(nodes, ctx)

	// Should always return empty because libopenapi handles $ref siblings in OAS 3.1
	assert.Len(t, res, 0, "OASNoRefSiblings should always return empty (libopenapi handles this)")
}

// TestNoRefSiblings_FormatDetection_Matrix tests format detection across
// all supported OpenAPI versions
func TestNoRefSiblings_FormatDetection_Matrix(t *testing.T) {
	testCases := []struct {
		version         string
		shouldHaveError bool
		reason          string
	}{
		{"2.0", true, "Swagger 2.0 doesn't allow $ref siblings"},
		{"3.0.0", true, "OAS 3.0.0 doesn't allow $ref siblings"},
		{"3.0.1", true, "OAS 3.0.1 doesn't allow $ref siblings"},
		{"3.0.2", true, "OAS 3.0.2 doesn't allow $ref siblings"},
		{"3.0.3", true, "OAS 3.0.3 doesn't allow $ref siblings"},
		{"3.1.0", false, "OAS 3.1.0 allows $ref siblings in Schema Objects"},
		{"3.1.1", false, "OAS 3.1.1 allows $ref siblings in Schema Objects"},
	}

	for _, tc := range testCases {
		t.Run("version_"+tc.version, func(t *testing.T) {
			var yml string
			if tc.version == "2.0" {
				yml = `swagger: "2.0"
info:
  title: Test
  version: 1.0.0
definitions:
  Example:
    description: test
    $ref: '#/definitions/Base'`
			} else {
				yml = `openapi: ` + tc.version + `
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    Example:
      description: test
      $ref: '#/components/schemas/Base'`
			}

			var rootNode yaml.Node
			_ = yaml.Unmarshal([]byte(yml), &rootNode)
			nodes, _ := utils.FindNodes([]byte(yml), "$")

			rule := buildOpenApiTestRuleAction("$", "no_ref_siblings_"+tc.version, "", nil)
			ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
			config := index.CreateOpenAPIIndexConfig()
			ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

			// Parse document to verify version detection
			doc, err := libopenapi.NewDocument([]byte(yml))
			assert.NoError(t, err)
			info := doc.GetSpecInfo()
			assert.Equal(t, tc.version, info.Version, "Version detection failed")

			// Use appropriate rule based on version
			var res []model.RuleFunctionResult
			if tc.version == "3.1.0" || tc.version == "3.1.1" {
				// OAS 3.1 uses OASNoRefSiblings
				def := OASNoRefSiblings{}
				res = def.RunRule(nodes, ctx)
			} else {
				// OAS 2.0 and 3.0.x use NoRefSiblings
				def := NoRefSiblings{}
				res = def.RunRule(nodes, ctx)
			}

			if tc.shouldHaveError {
				assert.Greater(t, len(res), 0, tc.reason)
			} else {
				assert.Len(t, res, 0, tc.reason)
			}
		})
	}
}

// TestNoRefSiblings_Issue750_InlineSchema tests $ref siblings in inline schemas
// (e.g., in request/response bodies)
func TestNoRefSiblings_Issue750_InlineSchema(t *testing.T) {
	yml := `openapi: 3.1.0
info:
  title: Inline Schema Test
  version: 1.0.0
paths:
  /test:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Base'
              readOnly: true
              description: Inline schema with siblings
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Response'
                deprecated: true
components:
  schemas:
    Base:
      type: object
    Response:
      type: object`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "no_ref_siblings", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	// Build doctor model for full analysis
	doc, err := libopenapi.NewDocument([]byte(yml))
	assert.NoError(t, err)
	v3Model, modelErrors := doc.BuildV3Model()
	assert.NoError(t, modelErrors)
	drDocument := drModel.NewDrDocument(v3Model)
	ctx.DrDocument = drDocument

	// Use OASNoRefSiblings for OAS 3.1
	def := OASNoRefSiblings{}
	res := def.RunRule(nodes, ctx)

	// Inline schemas in OAS 3.1 should allow $ref with siblings
	assert.Len(t, res, 0, "OAS 3.1 inline schemas should allow $ref with siblings")
}

// TestNoRefSiblings_Issue750_AllOfOneOfAnyOf tests $ref usage within
// composition keywords (allOf, oneOf, anyOf)
func TestNoRefSiblings_Issue750_AllOfOneOfAnyOf(t *testing.T) {
	yml := `openapi: 3.1.0
info:
  title: Composition Test
  version: 1.0.0
components:
  schemas:
    AllOfExample:
      allOf:
        - $ref: '#/components/schemas/Base'
          description: This is allowed
        - type: object
          properties:
            extra:
              type: string
    OneOfExample:
      oneOf:
        - $ref: '#/components/schemas/Option1'
          readOnly: true
        - $ref: '#/components/schemas/Option2'
    Base:
      type: object
    Option1:
      type: object
    Option2:
      type: object`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "no_ref_siblings", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	// Use OASNoRefSiblings for OAS 3.1
	def := OASNoRefSiblings{}
	res := def.RunRule(nodes, ctx)

	// $ref with siblings inside allOf/oneOf/anyOf is valid in OAS 3.1
	assert.Len(t, res, 0, "OAS 3.1 should allow $ref with siblings in composition keywords")
}
