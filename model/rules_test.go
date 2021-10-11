package model

import (
	"github.com/stretchr/testify/assert"
	"github.com/xeipuuv/gojsonschema"
	"io/ioutil"
	"testing"
)

func TestRuleSchema(t *testing.T) {

	schemaMain, err := ioutil.ReadFile("schemas/ruleset.schema.json")
	assert.NoError(t, err)

	goodRules, err := ioutil.ReadFile("test_files/rules.json")
	assert.NoError(t, err)

	schemaLoader := gojsonschema.NewStringLoader(string(schemaMain))
	ruleLoader := gojsonschema.NewStringLoader(string(goodRules))

	result, err := gojsonschema.Validate(schemaLoader, ruleLoader)
	assert.NoError(t, err)
	assert.True(t, result.Valid())
	assert.Len(t, result.Errors(), 0)

}

func TestCreateRuleSetUsingJSON_Fail(t *testing.T) {

	// this is not going to work.
	json := `{ "pizza" : "cake" }`

	_, err := CreateRuleSetUsingJSON([]byte(json))
	assert.Error(t, err)

}

func TestCreateRuleSetUsingJSON_Success(t *testing.T) {

	// this should work.
	json := `{
  "documentationUrl": "quobix.com",
  "rules": {
    "fish-cakes": {
      "description": "yummy sea food",
      "recommended": true,
      "type": "style",
      "given": "$.some.JSON.PATH",
      "then": {
        "field": "nextSteps",
        "function": "cookForTenMins"
      }
    }
  }
}
`
	rs, err := CreateRuleSetUsingJSON([]byte(json))
	assert.NoError(t, err)
	assert.Len(t, rs.Rules, 1)

}
