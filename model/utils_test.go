package model

import (
	"github.com/daveshanley/vacuum/utils"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

const (
	// OpenApi3 is used by all OpenAPI 3+ docs
	OpenApi3 = "openapi"

	// OpenApi2 is used by all OpenAPI 2 docs, formerly known as swagger.
	OpenApi2 = "swagger"

	// AsyncApi is used by akk AsyncAPI docs, all versions.
	AsyncApi = "asyncapi"
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
  title: Test API version is a bool?
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
	assert.Equal(t, utils.OpenApi3, r.SpecType)
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

func TestValidateRuleFunctionContextAgainstSchema_Success(t *testing.T) {

	opts := make(map[string]string)
	opts["type"] = "snake"
	rf := dummyFunc{}
	ctx := RuleFunctionContext{
		RuleAction: &RuleAction{
			Field:           "none",
			Function:        "casing",
			FunctionOptions: opts,
		},
		Options: opts,
	}
	res, errs := ValidateRuleFunctionContextAgainstSchema(rf, ctx)

	assert.True(t, res)
	assert.Len(t, errs, 0)
}

func TestValidateRuleFunctionContextAgainstSchema_Success_SimulateYAML(t *testing.T) {

	opts := make(map[string]interface{})
	opts["type"] = "snake"
	rf := dummyFunc{}
	ctx := RuleFunctionContext{
		RuleAction: &RuleAction{
			Field:           "none",
			Function:        "casing",
			FunctionOptions: opts,
		},
		Options: opts,
	}
	res, errs := ValidateRuleFunctionContextAgainstSchema(rf, ctx)

	assert.True(t, res)
	assert.Len(t, errs, 0)
}

func TestValidateRuleFunctionContextAgainstSchema_Fail(t *testing.T) {

	opts := make(map[string]string)
	rf := dummyFunc{}
	ctx := RuleFunctionContext{
		RuleAction: &RuleAction{
			Field:           "none",
			Function:        "casing",
			FunctionOptions: opts,
		},
		Options: opts,
	}
	res, errs := ValidateRuleFunctionContextAgainstSchema(rf, ctx)

	assert.False(t, res)
	assert.Len(t, errs, 1)
}

func TestValidateRuleFunctionContextAgainstSchema_MinMax_FailMin(t *testing.T) {

	opts := make(map[string]string)
	rf := dummyFuncMinMax{}
	ctx := RuleFunctionContext{
		RuleAction: &RuleAction{
			Field:           "none",
			Function:        "casing",
			FunctionOptions: opts,
		},
		Options: opts,
	}
	res, errs := ValidateRuleFunctionContextAgainstSchema(rf, ctx)

	assert.False(t, res)
	assert.Len(t, errs, 2)
}

func TestValidateRuleFunctionContextAgainstSchema_MinMax_FailMax(t *testing.T) {

	opts := make(map[string]string)
	opts["beer"] = "shoes"
	opts["lime"] = "kitty"
	opts["carrot"] = "cake"
	rf := dummyFuncMinMax{}
	ctx := RuleFunctionContext{
		RuleAction: &RuleAction{
			Field:           "none",
			Function:        "casing",
			FunctionOptions: opts,
		},
		Options: opts,
	}
	res, errs := ValidateRuleFunctionContextAgainstSchema(rf, ctx)

	assert.False(t, res)
	assert.Len(t, errs, 2)
}

func TestBuildFunctionResult(t *testing.T) {
	fr := BuildFunctionResult("pizza", "party", "tonight")
	assert.Equal(t, "'pizza' party 'tonight'", fr.Message)
}

func TestCastToRuleAction(t *testing.T) {
	ra := &RuleAction{
		Field: "choco",
	}
	assert.Equal(t, "choco", CastToRuleAction(ra).Field)
}

func TestCastToRuleAction_Fail_WrongType(t *testing.T) {
	ra := "not a rule action"
	assert.Nil(t, CastToRuleAction(ra))
}

func TestCastToRuleAction_Fail_Nil(t *testing.T) {
	var ra interface{}
	assert.Nil(t, CastToRuleAction(ra))
}

func TestAreValuesCorrectlyTyped(t *testing.T) {

	assert.Len(t, AreValuesCorrectlyTyped("string", []interface{}{"hi"}), 0)
	assert.Len(t, AreValuesCorrectlyTyped("string", []interface{}{1}), 1)
	assert.Len(t, AreValuesCorrectlyTyped("string", []interface{}{"nice", 123, int64(12345)}), 2)
	assert.Len(t, AreValuesCorrectlyTyped("string", []interface{}{1.2, "burgers"}), 1)
	assert.Len(t, AreValuesCorrectlyTyped("string", []interface{}{true, false, "what"}), 2)

	assert.Len(t, AreValuesCorrectlyTyped("integer", []interface{}{1, 2, 3, 4}), 0)
	assert.Len(t, AreValuesCorrectlyTyped("integer", []interface{}{"no way!"}), 1)
	assert.Len(t, AreValuesCorrectlyTyped("integer", []interface{}{"nice", 123, int64(12345)}), 1)
	assert.Len(t, AreValuesCorrectlyTyped("integer", []interface{}{999, 1.2, "burgers"}), 2)
	assert.Len(t, AreValuesCorrectlyTyped("integer", []interface{}{true, false, "what"}), 3)

	assert.Len(t, AreValuesCorrectlyTyped("number", []interface{}{1.2345}), 0)
	assert.Len(t, AreValuesCorrectlyTyped("number", []interface{}{"no way!"}), 1)
	assert.Len(t, AreValuesCorrectlyTyped("number", []interface{}{"nice", 123, 2.353}), 1)
	assert.Len(t, AreValuesCorrectlyTyped("number", []interface{}{999, 1.2, "burgers"}), 1)
	assert.Len(t, AreValuesCorrectlyTyped("number", []interface{}{true, false, "what"}), 3)

	assert.Len(t, AreValuesCorrectlyTyped("boolean", []interface{}{true, false, true}), 0)
	assert.Len(t, AreValuesCorrectlyTyped("boolean", []interface{}{"no way!"}), 1)
	assert.Len(t, AreValuesCorrectlyTyped("boolean", []interface{}{"nice", 123, 2.353, true}), 3)
	assert.Len(t, AreValuesCorrectlyTyped("boolean", []interface{}{true, true, "burgers"}), 1)
	assert.Len(t, AreValuesCorrectlyTyped("boolean", []interface{}{true, false, "what", 1.2, 4}), 3)

}

type dummyFunc struct {
}

func (df dummyFunc) GetSchema() RuleFunctionSchema {
	return RuleFunctionSchema{
		Required: []string{"type"},
		Properties: []RuleFunctionProperty{
			{
				Name:        "type",
				Description: "a type",
			},
		},
		ErrorMessage: "missing the type my friend.",
	}
}

func (df dummyFunc) RunRule(nodes []*yaml.Node, context RuleFunctionContext) []RuleFunctionResult {
	return nil
}

type dummyFuncMinMax struct {
}

func (df dummyFuncMinMax) GetSchema() RuleFunctionSchema {
	return RuleFunctionSchema{
		Required: []string{"type"},
		Properties: []RuleFunctionProperty{
			{
				Name:        "type",
				Description: "a type",
			},
		},
		MinProperties: 1,
		MaxProperties: 2,
		ErrorMessage:  "missing the type my friend.",
	}
}

func (df dummyFuncMinMax) RunRule(nodes []*yaml.Node, context RuleFunctionContext) []RuleFunctionResult {
	return nil
}
