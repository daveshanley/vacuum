package openapi

import (
    "fmt"
    "testing"

    "github.com/daveshanley/vacuum/model"
    drModel "github.com/pb33f/doctor/model"
    "github.com/pb33f/libopenapi"
    "github.com/stretchr/testify/assert"
)

func TestCamelCaseProperties_AllGood(t *testing.T) {
    yml := `openapi: 3.1.0
components:
  schemas:
    Good:
      type: object
      properties:
        userName:
          type: string
        id:
          type: string
        userID:
          type: string
        x-vendor:
          type: string
`
    document, err := libopenapi.NewDocument([]byte(yml))
    if err != nil {
        panic(fmt.Sprintf("cannot create new document: %e", err))
    }
    m, _ := document.BuildV3Model()
    drDocument := drModel.NewDrDocument(m)

    rule := buildOpenApiTestRuleAction("$", "camelCaseProperties", "", nil)
    ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
    ctx.Document = document
    ctx.DrDocument = drDocument
    ctx.Rule = &rule

    def := CamelCaseProperties{}
    res := def.RunRule(nil, ctx)

    assert.Len(t, res, 0)
}

func TestCamelCaseProperties_FindsViolations(t *testing.T) {
    yml := `openapi: 3.1.0
components:
  schemas:
    Bad:
      type: object
      properties:
        UserName:
          type: string
        user_name:
          type: string
        user-name:
          type: string
`
    document, err := libopenapi.NewDocument([]byte(yml))
    if err != nil {
        panic(fmt.Sprintf("cannot create new document: %e", err))
    }
    m, _ := document.BuildV3Model()
    drDocument := drModel.NewDrDocument(m)

    rule := buildOpenApiTestRuleAction("$", "camelCaseProperties", "", nil)
    ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
    ctx.Document = document
    ctx.DrDocument = drDocument
    ctx.Rule = &rule

    def := CamelCaseProperties{}
    res := def.RunRule(nil, ctx)

    // Expect 3 violations
    assert.Len(t, res, 3)
    // Ensure messages include detected case type
    assert.Contains(t, res[0].Message, "use camelCase")
    assert.Contains(t, res[1].Message, "use camelCase")
    assert.Contains(t, res[2].Message, "use camelCase")
}

func TestCamelCaseProperties_IgnoresVendorExtensions(t *testing.T) {
    yml := `openapi: 3.1.0
components:
  schemas:
    Mixed:
      type: object
      properties:
        x-custom:
          type: string
        X-Upper:
          type: string
        properCamel:
          type: string
`
    document, err := libopenapi.NewDocument([]byte(yml))
    if err != nil {
        panic(fmt.Sprintf("cannot create new document: %e", err))
    }
    m, _ := document.BuildV3Model()
    drDocument := drModel.NewDrDocument(m)

    rule := buildOpenApiTestRuleAction("$", "camelCaseProperties", "", nil)
    ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
    ctx.Document = document
    ctx.DrDocument = drDocument
    ctx.Rule = &rule

    def := CamelCaseProperties{}
    res := def.RunRule(nil, ctx)

    // Only properCamel remains and is valid, vendor extensions are ignored
    assert.Len(t, res, 0)
}



