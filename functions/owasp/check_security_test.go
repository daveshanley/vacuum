package owasp

import (
	"fmt"
	"testing"

	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"

	"github.com/daveshanley/vacuum/model"
	"github.com/stretchr/testify/assert"
)

func TestCheckSecurity_GetSchema(t *testing.T) {
	def := CheckSecurity{}
	assert.Equal(t, "check_security", def.GetSchema().Name)
}

func TestCheckSecurity_RunRule(t *testing.T) {
	def := CheckSecurity{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestCheckSecurity_SecurityMissing(t *testing.T) {

	yml := `openapi: 3.0.1
info:
  version: "1.2.3"
  title: "securitySchemes"
paths:
  /security-global-ok-put:
    put:
      responses: {}
  /security-ok-put:
    put:
      responses: {}
components:
  securitySchemes:
    BasicAuth:
      type: http
      scheme: basic`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "check_security", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), map[string]interface{}{
		"methods": []string{"put"},
	})

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	def := CheckSecurity{}
	res := def.RunRule(nil, ctx)

	assert.Len(t, res, 2)
	assert.Equal(t, "`security` was not defined for path `/security-global-ok-put` in method `put`", res[0].Message)
	assert.Equal(t, "`security` was not defined for path `/security-ok-put` in method `put`", res[1].Message)
	assert.Equal(t, "$.paths['/security-global-ok-put'].put", res[0].Path)
	assert.Equal(t, "$.paths['/security-ok-put'].put", res[1].Path)

}

func TestCheckSecurity_SecurityMissingOnOneOperation(t *testing.T) {
	yml := `openapi: 3.0.1
info:
  version: "1.2.3"
  title: "securitySchemes"
security:
  - BasicAuth: []
paths:
  /insecure:
    put:
      responses: {}
      security: []
  /secure:
    put:
      responses: {}
components:
  securitySchemes:
    BasicAuth:
      type: http
      scheme: basic`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "check_security", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), map[string]interface{}{
		"methods": []string{"put"},
	})

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := CheckSecurity{}.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "`security` is empty for path `/insecure` in method `put`", res[0].Message)
	assert.Equal(t, "$.paths['/insecure'].put", res[0].Path)
}

func TestCheckSecurity_SecurityGlobalDefined(t *testing.T) {
	yml := `openapi: 3.0.1
info:
  version: "1.2.3"
  title: "securitySchemes"
security:
  - BasicAuth: []
paths:
  /insecure:
    put:
      responses: {}
  /secure:
    put:
      responses: {}
components:
  securitySchemes:
    BasicAuth:
      type: http
      scheme: basic`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "check_security", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), map[string]interface{}{
		"methods": []string{"put"},
	})

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := CheckSecurity{}.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}

func TestCheckSecurity_SecurityLocalSecurityEmpty(t *testing.T) {
	yml := `openapi: 3.0.1
info:
  version: "1.2.3"
  title: "securitySchemes"
paths:
  /secure:
    put:
      responses: {}
      security:
        - {}`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "check_security", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), map[string]interface{}{
		"methods": []string{"put"},
	})

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := CheckSecurity{}.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "`security` has null elements for path `/secure` in method `put`", res[0].Message)
	assert.Equal(t, "$.paths['/secure'].put.security[0]", res[0].Path)
}

func TestCheckSecurity_SecurityLocalSecurityEmpty_AllowNull(t *testing.T) {
	yml := `openapi: 3.0.1
info:
  version: "1.2.3"
  title: "securitySchemes"
paths:
  /secure:
    put:
      responses: {}
      security:
        - {}`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "check_security", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), map[string]interface{}{
		"methods":  []string{"put"},
		"nullable": true,
	})

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := CheckSecurity{}.RunRule(nil, ctx)

	assert.Len(t, res, 0)
}

func TestCheckSecurity_SecurityGlobalDefined_Empty(t *testing.T) {
	yml := `openapi: 3.0.1
info:
  version: "1.2.3"
  title: "securitySchemes"
security: []
paths:
  /insecure:
    put:
      responses: {}
  /secure:
    put:
      responses: {}
      security:
        - BasicAuth: []
components:
  securitySchemes:
    BasicAuth:
      type: http
      scheme: basic`

	document, err := libopenapi.NewDocument([]byte(yml))
	if err != nil {
		panic(fmt.Sprintf("cannot create new document: %e", err))
	}

	m, _ := document.BuildV3Model()
	path := "$"

	drDocument := drModel.NewDrDocument(m)

	rule := buildOpenApiTestRuleAction(path, "check_security", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), map[string]interface{}{
		"methods": []string{"put"},
	})

	ctx.Document = document
	ctx.DrDocument = drDocument
	ctx.Rule = &rule

	res := CheckSecurity{}.RunRule(nil, ctx)

	assert.Len(t, res, 1)
	assert.Equal(t, "`security` was not defined for path `/insecure` in method `put`", res[0].Message)
	assert.Equal(t, "$.paths['/insecure'].put", res[0].Path)
}
