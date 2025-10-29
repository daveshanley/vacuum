package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPathParameters_GetSchema(t *testing.T) {
	def := PathParameters{}
	assert.Equal(t, "oasPathParam", def.GetSchema().Name)
}

func TestPathParameters_RunRule(t *testing.T) {
	def := PathParameters{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestPathParameters_RunRule_AllParamsInTop(t *testing.T) {

	yml := `openapi: 3.1
paths:
  /pizza/{type}/{topping}:
    parameters:
      - name: type
        in: path
    get:
      operationId: get_pizza`

	path := "$"

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "path-params", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := PathParameters{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "parameter named `topping` must be defined as part of the path `/pizza/{type}/{topping}` definition, or in the `GET` operation(s)", res[0].Message)
}

func TestPathParameters_RunRule_VerbsWithDifferentParams(t *testing.T) {

	yml := `openapi: 3.1
paths:
  /pizza/{type}/{topping}:
    parameters:
      - name: type
        in: path
    get:
      parameters:
        - name: topping
          in: path
      operationId: get_pizza
    post:
      operationId: make_pizza
`

	path := "$"

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "path-params", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := PathParameters{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "parameter named `topping` must be defined as part of the path `/pizza/{type}/{topping}` definition, or in the `POST` operation(s)", res[0].Message)
}

func TestPathParameters_RunRule_DuplicatePathCheck(t *testing.T) {

	yml := `openapi: 3.1
paths:
  /pizza/{cake}/{limes}:
    parameters:
      - in: path
        name: cake
    get:
      parameters:
        - in: path
          name: limes
  /pizza/{minty}/{tape}:
    parameters:
      - in: path
        name: minty
    get:
      parameters:
        - in: path
          name: tape`

	path := "$"

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "path-params", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := PathParameters{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "GET: paths `/pizza/{cake}/{limes}` and `/pizza/{minty}/{tape}` must not be equivalent, paths must be unique for the same HTTP method", res[0].Message)

}

func TestPathParameters_RunRule_DuplicatePathParamCheck_MissingParam(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
  /pizza/{cake}/{cake}:
    parameters:
      - in: path
        name: cake
    get:
      parameters:
        - in: path
          name: limes
  /pizza/{minty}:
    parameters:
      - in: path
        name: minty
    get:`

	path := "$"

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "path-params", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := PathParameters{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 2)
	assert.Equal(t, "path `/pizza/{cake}/{cake}` must not use the parameter `cake` multiple times", res[0].Message)
	assert.Equal(t, "`GET` parameter named `limes` does not exist in path `/pizza/{cake}/{cake}`", res[1].Message)

}

func TestPathParameters_RunRule_MissingParam_PeriodInParam(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
  /pizza/{cake}/{cake.id}:
    parameters:
      - in: path
        name: cake
    get:
      parameters:
        - in: path
          name: cake.id
          required: true`

	path := "$"

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "path-params", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := PathParameters{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)

}

func TestPathParameters_RunRule_TopParameterCheck_RequiredShouldBeTrue(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
 /musical/{melody}/{pizza}:
   parameters:
       - in: path
         name: melody
         required: false
   get:
     parameters:
       - in: path
         name: pizza
         required: true`

	path := "$"

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "path-params", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := PathParameters{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "path parameter named `melody` at `/musical/{melody}/{pizza}` must have `required` set to `true`", res[0].Message)
}

func TestPathParameters_RunRule_TopParameterCheck_MultipleDefinitionsOfParam(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
 /musical/{melody}/{pizza}:
   parameters:
       - in: path
         name: melody
       - in: path
         name: melody
   get:
     parameters:
       - in: path
         name: pizza
       - in: path
         name: pizza`

	path := "$"

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "path-params", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := PathParameters{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 2)
	assert.Equal(t, "path parameter named `melody` at `/musical/{melody}/{pizza}` is a duplicate of another parameter with the same name", res[0].Message)
	assert.Equal(t, "`GET` parameter named `pizza` at `/musical/{melody}/{pizza}` is a duplicate of another parameter with the same name", res[1].Message)

}

func TestPathParameters_RunRule_TopParameterCheck(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
 /musical/{melody}/{pizza}:
   parameters:
       - in: path
         name: melody
   get:
     parameters:
       - in: path
         name: pizza`

	path := "$"

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "path-params", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := PathParameters{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}

func TestPathParameters_RunRule_TopParameterCheck_MissingParamDefInOp(t *testing.T) {

	yml := `openapi: 3.1"
paths:
 /musical/{melody}/{pizza}/{cake}:
   parameters:
       - in: path
         name: melody
         required: true
   get:
     parameters:
       - in: path
         name: pizza
         required: true`

	path := "$"

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "path-params", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := PathParameters{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "parameter named `cake` must be defined as part of the path `/musical/{melody}/{pizza}/{cake}` definition, or in the `GET` operation(s)",
		res[0].Message)
}

func TestPathParameters_RunRule_MultiplePaths_TopAndVerbParams(t *testing.T) {

	yml := `openapi: 3.1.0
components: 
  parameters:
    chicken:
      in: path
      required: true
      name: chicken
paths:
 /musical/{melody}/{pizza}/{cake}:
   parameters:
     - in: path
       name: melody
       required: true
   get:
     parameters:
       - in: path
         name: pizza
         required: true
 /dogs/{chicken}/{ember}:
   get:
     parameters:
       - in: path
         name: ember
         required: true
       - $ref: '#/components/parameters/chicken'
   post:
     parameters:
       - in: path
         name: ember
       - $ref: '#/components/parameters/chicken'`

	path := "$"

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "path-params", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := PathParameters{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "parameter named `cake` must be defined as part of the path `/musical/{melody}/{pizza}/{cake}` definition, or in the `GET` operation(s)",
		res[0].Message)
}

func TestPathParameters_RunRule_NoParamsDefined(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
  /update/{somethings}:
    post:
      operationId: postSomething
      summary: Post something
      tags:
        - tag1
      responses:
        '200':
          description: Post OK
    get:
      operationId: getSomething
      summary: Get something
      tags:
        - tag1
      responses:
        '200':
          description: Get OK
components:
  securitySchemes:
    basicAuth:
      type: http
      scheme: basic`

	path := "$"

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "path-params", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := PathParameters{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 2)
	assert.Equal(t, "parameter named `somethings` must be defined as part of the path `/update/{somethings}` definition, or in the `POST` operation(s)",
		res[0].Message)
	assert.Equal(t, "parameter named `somethings` must be defined as part of the path `/update/{somethings}` definition, or in the `GET` operation(s)",
		res[1].Message)
}

func TestPathParameters_RunRule_NoParamsDefined_TopExists(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
  /update/{somethings}:
    parameters:
      - in: path
        name: somethings
        schema:
          type: string
          example: something nice
          description: this is something nice.
        required: true
    post:
      operationId: postSomething
      summary: Post something
      tags:
        - tag1
      responses:
        '200':
          description: Post OK
    get:
      operationId: getSomething
      summary: Get something
      tags:
        - tag1
      responses:
        '200':
          description: Get OK
components:
  securitySchemes:
    basicAuth:
      type: http
      scheme: basic`

	path := "$"

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "path-params", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := PathParameters{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}

func TestPathParameters_RunRule_CheckOpHasParam(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
  /test/two/{cakes}:
    parameters:
      - in: header
        name: yeah
    get:
      parameters:
        - in: path
          description:  hey
          schema:
            type: string
          name: cakes`

	path := "$"

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "path-params", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := PathParameters{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}

func TestPathParameters_RunRule_CheckForPathParamInPost(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
    /test/two/{cakes}:
        parameters:
              - in: header
                schema:
                    type: string
                name: yeah
                example: fish
                description: fingers
        post:
          description: minty fresh
          summary: crispy fresh
          parameters:
            - name: cakes
              in: path`

	path := "$"

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "path-params", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := PathParameters{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}

func TestPathParameters_RunRule_Issue740_ThreeDifferentMethods(t *testing.T) {
	// Test that three paths with same normalized pattern but different methods don't conflict
	yml := `openapi: 3.1.0
paths:
  /api/users/{userId}/items/{itemId}:
    get:
      summary: Get user item
      parameters:
        - name: userId
          in: path
          required: true
        - name: itemId
          in: path
          required: true
  /api/users/{customerId}/items/{productId}:
    post:
      summary: Create user item
      parameters:
        - name: customerId
          in: path
          required: true
        - name: productId
          in: path
          required: true
  /api/users/{accountId}/items/{objectId}:
    delete:
      summary: Delete user item
      parameters:
        - name: accountId
          in: path
          required: true
        - name: objectId
          in: path
          required: true`

	path := "$"

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "path-params", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := PathParameters{}
	res := def.RunRule(nil, ctx)

	// Should be NO errors - all different methods
	assert.Len(t, res, 0)
}

func TestPathParameters_RunRule_Issue740_SameMethodConflict(t *testing.T) {
	// Test that two paths with same normalized pattern and SAME method DO conflict
	yml := `openapi: 3.1.0
paths:
  /api/resources/{id}:
    get:
      summary: Get resource by ID
      parameters:
        - name: id
          in: path
          required: true
    post:
      summary: Create resource
      parameters:
        - name: id
          in: path
          required: true
  /api/resources/{resourceId}:
    get:  # This should conflict with the GET above
      summary: Get resource by resource ID
      parameters:
        - name: resourceId
          in: path
          required: true
    put:  # This should NOT conflict (different method)
      summary: Update resource
      parameters:
        - name: resourceId
          in: path
          required: true`

	path := "$"

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "path-params", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := PathParameters{}
	res := def.RunRule(nil, ctx)

	// Should have exactly 1 error for the duplicate GET methods
	assert.Len(t, res, 1)
	assert.Contains(t, res[0].Message, "GET: paths")
	assert.Contains(t, res[0].Message, "must not be equivalent")
	assert.Contains(t, res[0].Message, "/api/resources/{id}")
	assert.Contains(t, res[0].Message, "/api/resources/{resourceId}")
}
