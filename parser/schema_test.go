package parser

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

var openApiWat = `openapi: 3.2
info:
  title: Test API, valid, but not quite valid 
servers:
  - url: http://quobix.com/api`

func TestCheckSpecIsValidOpenAPI3_Error(t *testing.T) {

	res, err := CheckSpecIsValidOpenAPI([]byte(openApiWat))
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.False(t, res.Valid())

}

func TestCheckSpecIsValidOpenAPI3_Valid(t *testing.T) {

	spec, _ := ioutil.ReadFile("schemas/test_files/tiny.openapi.yaml")
	res, err := CheckSpecIsValidOpenAPI(spec)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.True(t, res.Valid())

}

func TestCheckSpecIsValidOpenAPI2_WrongVersion(t *testing.T) {

	spec, _ := ioutil.ReadFile("schemas/test_files/tiny.swagger-wrong-version.yaml")
	res, err := CheckSpecIsValidOpenAPI(spec)
	assert.Error(t, err)
	assert.Nil(t, res)
}

func TestCheckSpecIsValidOpenAPI2_Validn(t *testing.T) {

	spec, _ := ioutil.ReadFile("schemas/test_files/tiny.swagger.yaml")
	res, err := CheckSpecIsValidOpenAPI(spec)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.True(t, res.Valid())
}

func TestCheckSpecIsValidOpenAPI3_JSON(t *testing.T) {

	spec, _ := ioutil.ReadFile("schemas/test_files/tiny.openapi.json")
	res, err := CheckSpecIsValidOpenAPI(spec)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.True(t, res.Valid())

}

func TestCheckSpecIsValidOpenAPI3_Empty(t *testing.T) {
	res, err := CheckSpecIsValidOpenAPI(make([]byte, 0))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
	assert.Nil(t, res)
}

func TestCheckSpecIsValidOpenAPI3_Nonsense(t *testing.T) {
	res, err := CheckSpecIsValidOpenAPI([]byte("potatoes are:  our: friends"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "spec is neither")
	assert.Nil(t, res)
}

func TestCheckSpecIsValidOpenAPI3_NoVersion(t *testing.T) {
	res, err := CheckSpecIsValidOpenAPI([]byte("potatoes: are our friends"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "spec type not supported")
	assert.Nil(t, res)
}
