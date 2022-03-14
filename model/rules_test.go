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

func TestRuleFunctionSchema_GetPropertyDescription(t *testing.T) {
	df := dummyFunc{}
	assert.Equal(t, "a type", df.GetSchema().GetPropertyDescription("type"))
}

func TestRuleFunctionSchema_GetPropertyDescription_Fail(t *testing.T) {
	df := dummyFunc{}
	assert.Empty(t, df.GetSchema().GetPropertyDescription("pizza"))
}

func TestRule_ToJSON(t *testing.T) {
	r := Rule{}
	assert.NotEmpty(t, r.ToJSON())

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

func TestNewRuleResultSet(t *testing.T) {

	r1 := RuleFunctionResult{
		Message: "pip",
		Rule: &Rule{
			Severity: severityError,
		},
	}
	results := NewRuleResultSet([]RuleFunctionResult{r1})

	assert.Equal(t, r1, *results.Results[0])

}

func TestRuleResults_GetErrorCount(t *testing.T) {

	r1 := &RuleFunctionResult{Rule: &Rule{
		Severity: severityError,
	}}
	r2 := &RuleFunctionResult{Rule: &Rule{
		Severity: severityError,
	}}
	r3 := &RuleFunctionResult{Rule: &Rule{
		Severity: severityWarn,
	}}

	results := &RuleResultSet{Results: []*RuleFunctionResult{r1, r2, r3}}

	assert.Equal(t, 2, results.GetErrorCount())
	assert.Equal(t, 2, results.GetErrorCount())

}

func TestRuleResults_GetWarnCount(t *testing.T) {

	r1 := &RuleFunctionResult{Rule: &Rule{
		Severity: severityInfo,
	}}
	r2 := &RuleFunctionResult{Rule: &Rule{
		Severity: severityError,
	}}
	r3 := &RuleFunctionResult{Rule: &Rule{
		Severity: severityWarn,
	}}

	results := &RuleResultSet{Results: []*RuleFunctionResult{r1, r2, r3}}

	assert.Equal(t, 1, results.GetErrorCount())
	assert.Equal(t, 1, results.GetErrorCount())

}

func TestRuleResults_GetInfoCount(t *testing.T) {

	r1 := &RuleFunctionResult{Rule: &Rule{
		Severity: severityInfo,
	}}
	r2 := &RuleFunctionResult{Rule: &Rule{
		Severity: severityInfo,
	}}
	r3 := &RuleFunctionResult{Rule: &Rule{
		Severity: severityWarn,
	}}

	results := &RuleResultSet{Results: []*RuleFunctionResult{r1, r2, r3}}

	assert.Equal(t, 2, results.GetInfoCount())
	assert.Equal(t, 2, results.GetInfoCount())

}
