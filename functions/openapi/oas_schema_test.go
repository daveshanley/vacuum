package openapi

import (
    "github.com/daveshanley/vacuum/model"
    "github.com/pb33f/libopenapi"
    "github.com/pb33f/libopenapi/datamodel"
    "github.com/pb33f/libopenapi/index"
    "github.com/stretchr/testify/assert"
    "gopkg.in/yaml.v3"
    "testing"
)

func TestOAS2Schema_GetSchema(t *testing.T) {
    def := OASSchema{}
    assert.Equal(t, "oas_schema", def.GetSchema().Name)
}

func TestOAS2Schema_RunRule(t *testing.T) {
    def := OASSchema{}
    res := def.RunRule(nil, model.RuleFunctionContext{})
    assert.Len(t, res, 0)
}

func TestOAS2Schema_RunRule_Fail(t *testing.T) {

    yml := `swagger: 2.0
wiff: waff`

    path := "$"

    specInfo, _ := datamodel.ExtractSpecInfo([]byte(yml))

    rule := buildOpenApiTestRuleAction(path, "oas2_schema", "", nil)
    ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
    config := index.CreateOpenAPIIndexConfig()
    ctx.Index = index.NewSpecIndexWithConfig(specInfo.RootNode, config)
    ctx.SpecInfo = specInfo

    // add doc to context
    ctx.Document, _ = libopenapi.NewDocument([]byte(yml))

    def := OASSchema{}
    res := def.RunRule([]*yaml.Node{specInfo.RootNode}, ctx)

    assert.Len(t, res, 3)
}

func TestOAS2Schema_RunRule_JSONSource_Fail_SpecBorked(t *testing.T) {

    yml := `{"swagger":"2.0", hello":"there"}`

    path := "$"

    specInfo, _ := datamodel.ExtractSpecInfo([]byte(yml))

    rule := buildOpenApiTestRuleAction(path, "oas2_schema", "", nil)
    ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
    config := index.CreateOpenAPIIndexConfig()
    ctx.Index = index.NewSpecIndexWithConfig(specInfo.RootNode, config)
    ctx.SpecInfo = specInfo

    // add doc to context
    ctx.Document, _ = libopenapi.NewDocument([]byte(yml))

    def := OASSchema{}
    res := def.RunRule([]*yaml.Node{specInfo.RootNode}, ctx)

    assert.Len(t, res, 1)
}

func TestOAS2Schema_RunRule_JSONSource_Fail(t *testing.T) {

    yml := `{"swagger":"2.0", "hello":"there"}`

    path := "$"

    specInfo, _ := datamodel.ExtractSpecInfo([]byte(yml))

    rule := buildOpenApiTestRuleAction(path, "oas2_schema", "", nil)
    ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
    config := index.CreateOpenAPIIndexConfig()
    ctx.Index = index.NewSpecIndexWithConfig(specInfo.RootNode, config)
    ctx.SpecInfo = specInfo

    // add doc to context
    ctx.Document, _ = libopenapi.NewDocument([]byte(yml))

    def := OASSchema{}
    res := def.RunRule([]*yaml.Node{specInfo.RootNode}, ctx)

    assert.Len(t, res, 2)
}

func TestOAS2Schema_RunRule_JSONSource_Fail_Unknown(t *testing.T) {

    yml := `{"swimmer":"2.0", "hello":"there"}`

    path := "$"

    specInfo, _ := datamodel.ExtractSpecInfo([]byte(yml))

    rule := buildOpenApiTestRuleAction(path, "oas2_schema", "", nil)
    ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
    config := index.CreateOpenAPIIndexConfig()
    ctx.Index = index.NewSpecIndexWithConfig(specInfo.RootNode, config)
    ctx.SpecInfo = specInfo

    // add doc to context
    ctx.Document, _ = libopenapi.NewDocument([]byte(yml))

    def := OASSchema{}
    res := def.RunRule([]*yaml.Node{specInfo.RootNode}, ctx)

    assert.Len(t, res, 0)
}

func TestOAS2Schema_RunRule_AlmostPass(t *testing.T) {

    yml := `swagger: 2.0
info:
  contact:
    name: Hi
    url: https://quobix.com/vacuum
  license:
    name: MIT
  termsOfService: https://quobix.com/vacuum
  title: Quobix 
  version: "1.0"
paths:
  /hi:
    get:
      responses:
        "200":
          description: OK`

    path := "$"

    specInfo, _ := datamodel.ExtractSpecInfo([]byte(yml))

    rule := buildOpenApiTestRuleAction(path, "oas2_schema", "", nil)
    ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
    config := index.CreateOpenAPIIndexConfig()
    ctx.Index = index.NewSpecIndexWithConfig(specInfo.RootNode, config)
    ctx.SpecInfo = specInfo

    // add doc to context
    ctx.Document, _ = libopenapi.NewDocument([]byte(yml))

    def := OASSchema{}
    res := def.RunRule([]*yaml.Node{specInfo.RootNode}, ctx)

    assert.Len(t, res, 1)
}

func TestOAS3Schema_RunRule_Pass(t *testing.T) {

    yml := `openapi: "3.0.0"
info:
  contact:
    name: Hi
    url: https://quobix.com/vacuum
  license:
    name: MIT
  termsOfService: https://quobix.com/vacuum
  title: Quobix 
  version: "1.0"
paths:
  /hi:
    get:
      responses:
        "200":
          description: OK`

    path := "$"

    specInfo, _ := datamodel.ExtractSpecInfo([]byte(yml))

    rule := buildOpenApiTestRuleAction(path, "oas3_schema", "", nil)
    ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
    config := index.CreateOpenAPIIndexConfig()
    ctx.Index = index.NewSpecIndexWithConfig(specInfo.RootNode, config)
    ctx.SpecInfo = specInfo

    // add doc to context
    ctx.Document, _ = libopenapi.NewDocument([]byte(yml))

    def := OASSchema{}
    res := def.RunRule([]*yaml.Node{specInfo.RootNode}, ctx)

    assert.Len(t, res, 0)
}

func TestOAS3Schema_RunRule_Fail(t *testing.T) {

    yml := `openapi: "3.0"`

    path := "$"

    specInfo, _ := datamodel.ExtractSpecInfo([]byte(yml))

    rule := buildOpenApiTestRuleAction(path, "oas3_schema", "", nil)
    ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
    config := index.CreateOpenAPIIndexConfig()
    ctx.Index = index.NewSpecIndexWithConfig(specInfo.RootNode, config)
    ctx.SpecInfo = specInfo

    // add doc to context
    ctx.Document, _ = libopenapi.NewDocument([]byte(yml))

    def := OASSchema{}
    res := def.RunRule([]*yaml.Node{specInfo.RootNode}, ctx)

    assert.Len(t, res, 2)
}

func TestOAS2Schema_RunRule_Success(t *testing.T) {

    yml := `swagger: '2.0'
info:
  contact:
    name: Hi
    url: https://quobix.com/vacuum
  license:
    name: MIT
  termsOfService: https://quobix.com/vacuum
  title: Quobix 
  version: "1.0"
paths:
  /hi:
    get:
      responses:
        "200":
          description: OK`

    path := "$"

    specInfo, _ := datamodel.ExtractSpecInfo([]byte(yml))

    rule := buildOpenApiTestRuleAction(path, "oas2_schema", "", nil)
    ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
    config := index.CreateOpenAPIIndexConfig()
    ctx.Index = index.NewSpecIndexWithConfig(specInfo.RootNode, config)
    ctx.SpecInfo = specInfo

    // add doc to context
    ctx.Document, _ = libopenapi.NewDocument([]byte(yml))

    def := OASSchema{}
    res := def.RunRule([]*yaml.Node{specInfo.RootNode}, ctx)

    assert.Len(t, res, 0)
}
