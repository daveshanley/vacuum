package model

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var goodJSON = `{"name":"kitty", "noises":["meow","purrrr","gggrrraaaaaooooww"]}`
var badJSON = `{"name":"kitty, "noises":[{"meow","purrrr","gggrrraaaaaooooww"]}}`
var goodYAML = `name: kitty
noises:
- meow
- purrr
- gggggrrraaaaaaaaaooooooowwwwwww
`

var badYAML = `name: kitty
  noises:
   - meow
    - purrr
    - gggggrrraaaaaaaaaooooooowwwwwww
`

var OpenApiWat = `openapi: 3.2
info:
  title: Test API, valid, but not quite valid 
servers:
  - url: http://eve.vmware.com/api`

var OpenApiFalse = `openapi: false
info:
  title: Test API verison is a bool?
servers:
  - url: http://eve.vmware.com/api`

var OpenApi3Spec = `openapi: 3.0.1
info:
  title: Test API
tags:
  - name: "Test"
  - name: "Test 2"
servers:
  - url: http://eve.vmware.com/api`

var OpenApi2Spec = `swagger: 2.0.1
info:
  title: Test API
tags:
  - name: "Test"
servers:
  - url: http://eve.vmware.com/api`

var AsyncAPISpec = `asyncapi: 2.0.0
info:
  title: Hello world application
  version: '0.1.0'
channels:
  hello:
    publish:
      message:
        payload:
          type: string
          pattern: '^hello .+$'`

func TestExtractSpecInfo_ValidJSON(t *testing.T) {
	_, e := ExtractSpecInfo([]byte(goodJSON))
	assert.NotNil(t, e)
}

func TestExtractSpecInfo_InvalidJSON(t *testing.T) {
	_, e := ExtractSpecInfo([]byte(badJSON))
	assert.Error(t, e)
}

func TestExtractSpecInfo_ValidYAML(t *testing.T) {
	_, e := ExtractSpecInfo([]byte(goodYAML))
	assert.Error(t, e)
}

func TestExtractSpecInfo_InvalidYAML(t *testing.T) {
	_, e := ExtractSpecInfo([]byte(badYAML))
	assert.Error(t, e)
}

func TestExtractSpecInfo_OpenAPI3(t *testing.T) {

	r, e := ExtractSpecInfo([]byte(OpenApi3Spec))
	assert.Nil(t, e)
	assert.Equal(t, OpenApi3, r.SpecType)
	assert.Equal(t, "3.0.1", r.Version)
}

func TestExtractSpecInfo_OpenAPIWat(t *testing.T) {

	r, e := ExtractSpecInfo([]byte(OpenApiWat))
	assert.Nil(t, e)
	assert.Equal(t, OpenApi3, r.SpecType)
	assert.Equal(t, "3.20", r.Version)
}

func TestExtractSpecInfo_OpenAPIFalse(t *testing.T) {

	_, e := ExtractSpecInfo([]byte(OpenApiFalse))
	assert.Error(t, e)
}

func TestExtractSpecInfo_OpenAPI2(t *testing.T) {

	r, e := ExtractSpecInfo([]byte(OpenApi2Spec))
	assert.Nil(t, e)
	assert.Equal(t, OpenApi2, r.SpecType)
	assert.Equal(t, "2.0.1", r.Version)
}

func TestExtractSpecInfo_AsyncAPI(t *testing.T) {

	r, e := ExtractSpecInfo([]byte(AsyncAPISpec))
	assert.Nil(t, e)
	assert.Equal(t, AsyncApi, r.SpecType)
	assert.Equal(t, "2.0.0", r.Version)
}
