package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
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

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "examples", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := Examples{}

	// we need to resolve this
	resolved, _ := model.ResolveOpenAPIDocument(nodes[0])
	res := def.RunRule([]*yaml.Node{resolved}, ctx)

	assert.Len(t, res, 4)
	assert.Equal(t, "schema for 'application/json' does not contain any examples or example data", res[0].Message)

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

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "examples", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := Examples{}

	// we need to resolve this
	resolved, _ := model.ResolveOpenAPIDocument(nodes[0])
	res := def.RunRule([]*yaml.Node{resolved}, ctx)

	assert.Len(t, res, 4)

}

func TestExamples_RunRule_Fail_Schema_Examples_Not_Valid(t *testing.T) {

	yml := `paths:
 /fruits:
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

	nodes, _ := utils.FindNodes([]byte(yml), path)
	rule := buildOpenApiTestRuleAction(path, "examples", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := Examples{}

	// we need to resolve this
	resolved, _ := model.ResolveOpenAPIDocument(nodes[0])
	res := def.RunRule([]*yaml.Node{resolved}, ctx)

	assert.Len(t, res, 6)

}

func TestExamples_RunRule_Fail_Inline_Schema_Multi_Examples(t *testing.T) {

	yml := `paths:
 /fruits:
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

	nodes, _ := utils.FindNodes([]byte(yml), path)
	rule := buildOpenApiTestRuleAction(path, "examples", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := Examples{}

	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 4)

}

func TestExamples_RunRule_Fail_Inline_Schema_Missing_Summary(t *testing.T) {

	yml := `paths:
 /fruits:
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
              value:
                summary: this is an example of a lemon.
                id: 1
            lime:
              value: 
                id: 2`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)
	rule := buildOpenApiTestRuleAction(path, "examples", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := Examples{}

	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "example 'lime' missing a 'summary', examples need explaining", res[0].Message)
}

func TestExamples_RunRule_Fail_Single_Example_Not_An_Object(t *testing.T) {

	yml := `paths:
 /fruits:
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

	nodes, _ := utils.FindNodes([]byte(yml), path)
	rule := buildOpenApiTestRuleAction(path, "examples", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := Examples{}

	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "example for media type 'application/json' is malformed, "+
		"should be object, not 'apples'", res[0].Message)
}

func TestExamples_RunRule_Fail_Single_Example_Invalid_Object(t *testing.T) {

	yml := `paths:
 /fruits:
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

	nodes, _ := utils.FindNodes([]byte(yml), path)
	rule := buildOpenApiTestRuleAction(path, "examples", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := Examples{}

	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "example for 'application/json' is not valid: 'Invalid type. Expected: "+
		"integer, given: string' on field 'id'", res[0].Message)
}

func TestExamples_RunRule_Fail_Single_Example_Invalid_Object_Response(t *testing.T) {

	yml := `paths:
 /fruits:
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

	nodes, _ := utils.FindNodes([]byte(yml), path)
	rule := buildOpenApiTestRuleAction(path, "examples", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := Examples{}

	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "example for 'application/json' is not valid: 'Invalid type. Expected: "+
		"integer, given: string' on field 'id'", res[0].Message)
}

func TestExamples_RunRule_Fail_InlineExample_Wrong_Type(t *testing.T) {

	yml := `paths:
 /fruits:
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
                example: cake
              enabled:
                type: boolean
                example: limes
              stock:
                type: number
                example: fizzbuzz`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)
	rule := buildOpenApiTestRuleAction(path, "examples", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := Examples{}

	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 3)
	assert.Equal(t, "example value 'cake' in 'id' is not a valid integer", res[0].Message)
	assert.Equal(t, "example value 'limes' in 'enabled' is not a valid boolean", res[1].Message)
	assert.Equal(t, "example value 'fizzbuzz' in 'stock' is not a valid number", res[2].Message)
}

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

	nodes, _ := utils.FindNodes([]byte(yml), path)
	rule := buildOpenApiTestRuleAction(path, "examples", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := Examples{}

	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "schema for 'nuggets' does not contain any examples or example data", res[0].Message)
}

func TestExamples_RunRule_Fail_TopLevel_Param_No_Example(t *testing.T) {

	yml := `components:
  parameters:
    - in: path
      name: icypop
      schema:
        type: integer`

	path := "$"

	nodes, _ := utils.FindNodes([]byte(yml), path)
	rule := buildOpenApiTestRuleAction(path, "examples", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := Examples{}

	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "schema for 'icypop' does not contain any examples or example data", res[0].Message)
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

	nodes, _ := utils.FindNodes([]byte(yml), path)
	rule := buildOpenApiTestRuleAction(path, "examples", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := Examples{}

	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "missing example for 'id' on component 'Chickens'", res[0].Message)
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

	nodes, _ := utils.FindNodes([]byte(yml), path)
	rule := buildOpenApiTestRuleAction(path, "examples", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := Examples{}

	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "example for property 'id' is not valid: 'Invalid type. Expected: integer, "+
		"given: string'. Value 'burgers' is not compatible", res[0].Message)
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

	nodes, _ := utils.FindNodes([]byte(yml), path)
	rule := buildOpenApiTestRuleAction(path, "examples", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := Examples{}

	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "example for component 'Lemons' is not valid: 'Invalid type. Expected: integer, "+
		"given: string'. Value 'cake' is not compatible", res[0].Message)
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

	nodes, _ := utils.FindNodes([]byte(yml), path)
	rule := buildOpenApiTestRuleAction(path, "examples", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := Examples{}

	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "example for component 'Lemons' is not valid: 'Invalid type. Expected: integer, "+
		"given: string'. Value 'cake' is not compatible", res[0].Message)

}
