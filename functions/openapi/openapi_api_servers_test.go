package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestAPIServers_GetSchema(t *testing.T) {
	def := APIServers{}
	assert.Equal(t, "oasAPIServers", def.GetSchema().Name)
}

func TestAPIServers_RunRule(t *testing.T) {
	def := APIServers{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestAPIServers_RunRule_Fail(t *testing.T) {

	yml := `nope: no"`

	path := "$"

	specInfo, _ := datamodel.ExtractSpecInfo([]byte(yml))

	rule := buildOpenApiTestRuleAction(path, "api_servers", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(specInfo.RootNode, config)
	ctx.SpecInfo = specInfo

	def := APIServers{}
	res := def.RunRule([]*yaml.Node{specInfo.RootNode}, ctx)

	assert.Len(t, res, 1)
}

func TestAPIServers_RunRule_Fail_EmptyServers(t *testing.T) {

	yml := `servers:
paths:
  /nice/cake:`

	path := "$"

	specInfo, _ := datamodel.ExtractSpecInfo([]byte(yml))

	rule := buildOpenApiTestRuleAction(path, "api_servers", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(specInfo.RootNode, config)
	ctx.SpecInfo = specInfo

	def := APIServers{}
	res := def.RunRule([]*yaml.Node{specInfo.RootNode}, ctx)

	assert.Len(t, res, 1)
}

func TestAPIServers_RunRule_Success(t *testing.T) {

	yml := `servers:
  - url: https://quobix.com
paths:
  /nice/cake:`

	path := "$"

	specInfo, _ := datamodel.ExtractSpecInfo([]byte(yml))

	rule := buildOpenApiTestRuleAction(path, "api_servers", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(specInfo.RootNode, config)
	ctx.SpecInfo = specInfo

	def := APIServers{}
	res := def.RunRule([]*yaml.Node{specInfo.RootNode}, ctx)

	assert.Len(t, res, 0)
}

func TestAPIServers_RunRule_Success_With_Path(t *testing.T) {

	yml := `servers:
  - url: https://quobix.com/vacuum/docs
paths:
  /nice/cake:`

	path := "$"

	specInfo, _ := datamodel.ExtractSpecInfo([]byte(yml))

	rule := buildOpenApiTestRuleAction(path, "api_servers", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(specInfo.RootNode, config)
	ctx.SpecInfo = specInfo

	def := APIServers{}
	res := def.RunRule([]*yaml.Node{specInfo.RootNode}, ctx)

	assert.Len(t, res, 0)
}

func TestAPIServers_RunRule_Fail_With_Path(t *testing.T) {

	yml := `servers:
  - url: https://quobix.com/vacuum/docs/
paths:
  /nice/cake:`

	path := "$"

	specInfo, _ := datamodel.ExtractSpecInfo([]byte(yml))

	rule := buildOpenApiTestRuleAction(path, "api_servers", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(specInfo.RootNode, config)
	ctx.SpecInfo = specInfo

	def := APIServers{}
	res := def.RunRule([]*yaml.Node{specInfo.RootNode}, ctx)

	assert.Len(t, res, 1)
}

func TestAPIServers_RunRule_Fail_EmptyURL(t *testing.T) {

	yml := `servers:
  - pizza: https://quobix.com
paths:
  /nice/cake:`

	path := "$"

	specInfo, _ := datamodel.ExtractSpecInfo([]byte(yml))

	rule := buildOpenApiTestRuleAction(path, "api_servers", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(specInfo.RootNode, config)
	ctx.SpecInfo = specInfo

	def := APIServers{}
	res := def.RunRule([]*yaml.Node{specInfo.RootNode}, ctx)

	assert.Len(t, res, 1)
}

func TestAPIServers_RunRule_Fail_BadURL(t *testing.T) {
	// https://stackoverflow.com/questions/25747580/ensure-a-uri-is-valid/25747925#25747925
	yml := `servers:
  - url: http.\:::/not.valid/a//a??a?b=&&c#hi
paths:
  /nice/cake:`

	path := "$"

	specInfo, _ := datamodel.ExtractSpecInfo([]byte(yml))

	rule := buildOpenApiTestRuleAction(path, "api_servers", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(specInfo.RootNode, config)
	ctx.SpecInfo = specInfo

	def := APIServers{}
	res := def.RunRule([]*yaml.Node{specInfo.RootNode}, ctx)

	assert.Len(t, res, 1)
}

func TestAPIServers_RunRule_Fail_ReallyBadURL(t *testing.T) {

	yml := `servers:
  - url: http::no::///no
paths:
  /nice/cake:`

	path := "$"

	specInfo, _ := datamodel.ExtractSpecInfo([]byte(yml))

	rule := buildOpenApiTestRuleAction(path, "api_servers", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(specInfo.RootNode, config)
	ctx.SpecInfo = specInfo

	def := APIServers{}
	res := def.RunRule([]*yaml.Node{specInfo.RootNode}, ctx)

	assert.Len(t, res, 1)
}

func TestAPIServers_RunRule_Fail_EndsWithSlash(t *testing.T) {

	yml := `servers:
  - url: https://quobix.com/
paths:
  /nice/cake:`

	path := "$"

	specInfo, _ := datamodel.ExtractSpecInfo([]byte(yml))

	rule := buildOpenApiTestRuleAction(path, "api_servers", "", nil)
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), nil)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(specInfo.RootNode, config)
	ctx.SpecInfo = specInfo

	def := APIServers{}
	res := def.RunRule([]*yaml.Node{specInfo.RootNode}, ctx)

	assert.Len(t, res, 1)
}
