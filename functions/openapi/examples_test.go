package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/resolver"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestExamples_GetSchema(t *testing.T) {
	def := Examples{}
	assert.Equal(t, "examples", def.GetSchema().Name)
}

func TestExamples_RunRule(t *testing.T) {
	def := Examples{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestExamples_RunRule_Fail_Schema_No_Examples(t *testing.T) {

	yml := `paths:
  /pizza:
    requestBody:
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Pizza'
  /pasta:
    requestBody:
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Pizza'          
components:
  schemas:
    Pizza:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	rule := buildOpenApiTestRuleAction(path, "examples", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)
	ctx.SpecInfo = &datamodel.SpecInfo{
		SpecFormat: model.OAS3,
	}

	def := Examples{}

	// we need to resolve this
	resolver := resolver.NewResolver(ctx.Index)
	resolver.Resolve()
	res := def.RunRule([]*yaml.Node{&rootNode}, ctx)

	assert.Len(t, res, 2)
	assert.Equal(t, "Missing example for `id` on component `Pizza`", res[0].Message)
	assert.NotNil(t, res[0].Path)

}

func TestExamples_RunRule_Fail_Schema_Examples_Not_Objects(t *testing.T) {

	yml := `paths:
  /cake:
    requestBody:
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Cake'
          examples:
            not: a cake,
            tasty: not today
components:
  schemas:
    Cake:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	rule := buildOpenApiTestRuleAction(path, "examples", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)
	ctx.SpecInfo = &datamodel.SpecInfo{
		SpecFormat: model.OAS3,
	}
	def := Examples{}

	// we need to resolve this
	resolver := resolver.NewResolver(ctx.Index)
	resolver.Resolve()
	res := def.RunRule([]*yaml.Node{&rootNode}, ctx)

	assert.Len(t, res, 2)
	assert.NotNil(t, res[0].Path)

}

func TestExamples_RunRule_Fail_Schema_Examples_Not_Valid(t *testing.T) {

	yml := `paths:
 /fruits:
   post:
    requestBody:
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Citrus'
          examples:
            lemon:
              value:
                id: not-a-number
            lime:
              value:
                id: 2
                name: Limes!
components:
 schemas:
   Citrus:
     type: object
     required: 
      - name
      - id
     properties:
       id:
         type: integer
       name:
         type: string`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	//nodes, _ := utils.FindNodes([]byte(yml), path)
	rule := buildOpenApiTestRuleAction(path, "examples", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)
	def := Examples{}

	ctx.SpecInfo = &datamodel.SpecInfo{
		SpecFormat: model.OAS3,
	}

	// we need to resolve this
	//resolved, _ := model.ResolveOpenAPIDocument(nodes[0])
	//res := def.RunRule([]*yaml.Node{resolved}, ctx)

	resolver := resolver.NewResolver(ctx.Index)
	resolver.Resolve()
	res := def.RunRule([]*yaml.Node{&rootNode}, ctx)

	assert.Len(t, res, 6)
	assert.NotNil(t, res[0].Path)
}

func TestExamples_RunRule_Fail_Inline_Schema_Multi_Examples(t *testing.T) {

	yml := `paths:
  /fruits:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required: 
                - name
                - id
              properties:
                id:
                  type: integer
                name:
                  type: string
            examples:
              lemon:
                value: 
                  id: in
                  invalidProperty: oh dear
              lime:
                value: 
                  id: 2
                  name: Pickles`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)
	rule := buildOpenApiTestRuleAction(path, "examples", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	ctx.SpecInfo = &datamodel.SpecInfo{
		SpecFormat: model.OAS3,
	}

	def := Examples{}

	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 4)
	assert.NotNil(t, res[0].Path)
}

func TestExamples_RunRule_Fail_Inline_Schema_Missing_Summary(t *testing.T) {

	yml := `paths:
  /fruits:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required: 
                - id
              properties:
                id:
                  type: integer
            examples:
              lemon:
                summary: this is an example of a lemon.	
                value:
                  id: 1
              lime:
                value: 
                  id: 2`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)
	rule := buildOpenApiTestRuleAction(path, "examples", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)
	ctx.SpecInfo = &datamodel.SpecInfo{
		SpecFormat: model.OAS3,
	}
	def := Examples{}

	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "Example `lime` (line 19) missing a `summary` - examples need explaining", res[0].Message)
	assert.NotNil(t, res[0].Path)
}

func TestExamples_RunRule_Fail_Single_Example_Not_An_Object(t *testing.T) {

	yml := `paths:
  /fruits:
    put:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required: 
                - id
              properties:
                id:
                  type: integer
            example: apples`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)
	rule := buildOpenApiTestRuleAction(path, "examples", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)
	ctx.SpecInfo = &datamodel.SpecInfo{
		SpecFormat: model.OAS3,
	}
	def := Examples{}

	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "Example for `application/json` is not valid: `Invalid type. Expected: object, given: string`", res[0].Message)
	assert.NotNil(t, res[0].Path)
}

func TestExamples_RunRule_Fail_Single_Example_Invalid_Object(t *testing.T) {

	yml := `paths:
  /fruits:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required: 
                - id
              properties:
                id:
                  type: integer
            example:
              id: cake`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)
	rule := buildOpenApiTestRuleAction(path, "examples", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)
	ctx.SpecInfo = &datamodel.SpecInfo{
		SpecFormat: model.OAS3,
	}
	def := Examples{}

	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "Example for `application/json` is not valid: `Invalid type. Expected: "+
		"integer, given: string`", res[0].Message)
	assert.NotNil(t, res[0].Path)
}

func TestExamples_RunRule_Fail_Single_Example_Invalid_Object_Response(t *testing.T) {

	yml := `paths:
  /fruits:
    get:
      responses:
        '200':
          content:
            application/json:
              schema:
                type: object
                required: 
                  - id
                properties:
                  id:
                    type: integer
              example:
                id: cake`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)
	rule := buildOpenApiTestRuleAction(path, "examples", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)
	def := Examples{}
	ctx.SpecInfo = &datamodel.SpecInfo{
		SpecFormat: model.OAS3,
	}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "Example for `application/json` is not valid: `Invalid type. "+
		"Expected: integer, given: string`", res[0].Message)
	assert.NotNil(t, res[0].Path)
}

//
//func TestExamples_RunRule_Fail_InlineExample_Wrong_Type(t *testing.T) {
//
//	yml := `paths:
//  /fruits:
//    get:
//      responses:
//        '200':
//          content:
//            application/json:
//              schema:
//                type: object
//                required:
//                  - id
//                properties:
//                  id:
//                    type: integer
//                    example: cake
//                  enabled:
//                    type: boolean
//                    example: limes
//                  stock:
//                    type: number
//                    example: fizzbuzz`
//
//	path := "$"
//
//	var rootNode yaml.Node
//	yaml.Unmarshal([]byte(yml), &rootNode)
//
//	nodes, _ := utils.FindNodes([]byte(yml), path)
//	rule := buildOpenApiTestRuleAction(path, "examples", "", nil)
//	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
//	ctx.Index = index.NewSpecIndex(&rootNode)
//	ctx.SpecInfo = &datamodel.SpecInfo{
//		SpecFormat: model.OAS3,
//	}
//	def := Examples{}
//
//	res := def.RunRule(nodes, ctx)
//
//	assert.Len(t, res, 4)
//}

func TestExamples_RunRule_Fail_Single_Example_Param_No_Example(t *testing.T) {

	yml := `paths:
 /chicken:
   get:
     parameters:
       - in: path
         name: nuggets
         schema:
           type: integer`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)
	rule := buildOpenApiTestRuleAction(path, "examples", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)
	ctx.SpecInfo = &datamodel.SpecInfo{
		SpecFormat: model.OAS3,
	}
	def := Examples{}

	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "Schema for `nuggets` does not contain any examples or example data", res[0].Message)
	assert.NotNil(t, res[0].Path)
}

func TestExamples_RunRule_Fail_TopLevel_Param_No_Example(t *testing.T) {

	yml := `components:
  parameters:
    param1:
      in: path
      name: icypop
      schema:
        type: integer`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)
	rule := buildOpenApiTestRuleAction(path, "examples", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)
	ctx.SpecInfo = &datamodel.SpecInfo{
		SpecFormat: model.OAS3,
	}
	def := Examples{}

	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "Schema for `icypop` does not contain any examples or example data", res[0].Message)
	assert.NotNil(t, res[0].Path)
}

func TestExamples_RunRule_Fail_Component_No_Example(t *testing.T) {

	yml := `components:
  schemas:
    Chickens:
      type: object
      required: 
        - id
      properties:
        id:
          type: integer
          `

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)
	rule := buildOpenApiTestRuleAction(path, "examples", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)
	ctx.SpecInfo = &datamodel.SpecInfo{
		SpecFormat: model.OAS3,
	}
	def := Examples{}

	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "Missing example for `id` on component `Chickens`", res[0].Message)
	assert.NotNil(t, res[0].Path)
}

func TestExamples_RunRule_Fail_Component_Invalid_Inline_Example(t *testing.T) {

	yml := `components:
  schemas:
    Chickens:
      type: object
      required: 
        - id
      properties:
        id:
          type: integer
          example: burgers`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)
	rule := buildOpenApiTestRuleAction(path, "examples", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)
	ctx.SpecInfo = &datamodel.SpecInfo{
		SpecFormat: model.OAS3,
	}
	def := Examples{}

	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "Example for property `id` is not valid: `Invalid type. Expected: integer, "+
		"given: string`. Value `burgers` is not compatible", res[0].Message)
	assert.NotNil(t, res[0].Path)
}

func TestExamples_RunRule_Fail_Component_Invalid_ObjectLevel_Example(t *testing.T) {

	yml := `components:
  schemas:
    Lemons:
      type: object
      required: 
        - id
      properties:
        id:
          type: integer
      example:
        id: cake`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)
	rule := buildOpenApiTestRuleAction(path, "examples", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)
	ctx.SpecInfo = &datamodel.SpecInfo{
		SpecFormat: model.OAS3,
	}
	def := Examples{}

	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "Example for component `Lemons` is not valid: `Invalid type. Expected: integer, "+
		"given: string`. Value `cake` is not compatible", res[0].Message)
	assert.NotNil(t, res[0].Path)
}

func TestExamples_RunRule_Fail_Parameters_Invalid_ObjectLevel_Example(t *testing.T) {

	yml := `components:
  schemas:
    Lemons:
      type: object
      required: 
        - id
      properties:
        id:
          type: integer
      example:
        id: cake`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)
	rule := buildOpenApiTestRuleAction(path, "examples", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)
	ctx.SpecInfo = &datamodel.SpecInfo{
		SpecFormat: model.OAS3,
	}
	def := Examples{}

	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "Example for component `Lemons` is not valid: `Invalid type. Expected: integer, "+
		"given: string`. Value `cake` is not compatible", res[0].Message)
	assert.NotNil(t, res[0].Path)

}

func TestExamples_RunRule_Fail_ExternalAndValue(t *testing.T) {

	yml := `paths:
  /fruits:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required: 
                - id
              properties:
                id:
                  type: integer
            examples:
              lemon:
                externalValue: https://quobix.com
                summary: this is an example of a lemon.                  
                value:
                  id: 1
              lime:
                summary: nice chickens
                value: 
                  id: 2`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)
	rule := buildOpenApiTestRuleAction(path, "examples", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)
	ctx.SpecInfo = &datamodel.SpecInfo{
		SpecFormat: model.OAS3,
	}
	def := Examples{}

	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "Example `lemon` is not valid: cannot use both `value` and `externalValue` - choose one or the other", res[0].Message)
	assert.NotNil(t, res[0].Path)
}
