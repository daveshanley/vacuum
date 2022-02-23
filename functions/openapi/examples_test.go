package openapi

import (
	"fmt"
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

	assert.Len(t, res, 1)
	assert.Equal(t, "schema for 'application/json' does not contain a sibling 'example' or 'examples', "+
		"examples are *super* important", res[0].Message)

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

	assert.Len(t, res, 2)

}

func TestExamples_RunRule_Fail_Schema_Examples_Not_Valid(t *testing.T) {

	//	yml := `paths:
	//  /pizza:
	//    requestBody:
	//      content:
	//        application/json:
	//          schema:
	//            $ref: '#/components/schemas/Pizza'
	//          examples:
	//            fish:
	//              id: 1
	//              name: cod
	//            cake:
	//              id: 2
	//              name: carrot
	//components:
	//  schemas:
	//    Pizza:
	//      type: object
	//      properties:
	//        id:
	//          type: integer
	//        name:
	//          type: string`

	yml := `paths:
 /fruits:
   requestBody:
     content:
       application/json:
         schema:
           $ref: '#/components/schemas/Citrus'
         examples:
           lemon:
             id: not-a-number
           lime:
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

	nodes, e := utils.FindNodes([]byte(yml), path)
	fmt.Print(e)
	rule := buildOpenApiTestRuleAction(path, "examples", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)

	def := Examples{}

	// we need to resolve this
	resolved, _ := model.ResolveOpenAPIDocument(nodes[0])
	res := def.RunRule([]*yaml.Node{resolved}, ctx)

	assert.Len(t, res, 2)

}
