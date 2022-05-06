package rulesets

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBuildDefaultRuleSets(t *testing.T) {

	rs := BuildDefaultRuleSets()
	assert.NotNil(t, rs.GenerateOpenAPIDefaultRuleSet())
	assert.Len(t, rs.GenerateOpenAPIDefaultRuleSet().Rules, 46)

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

func TestRuleSet_GetExtendsValue_Single(t *testing.T) {

	yaml := `extends: spectral:oas
rules:
 fish-cakes:
   description: yummy sea food
   recommended: true
   type: style
   given: "$.some.JSON.PATH"
   then:
     field: nextSteps
     function: cookForTenMins`

	rs, err := CreateRuleSetFromData([]byte(yaml))
	assert.NoError(t, err)
	assert.Len(t, rs.Rules, 1)
	assert.NotNil(t, rs.GetExtendsValue())
	assert.Equal(t, "spectral:oas", rs.GetExtendsValue()["spectral:oas"])

}

func TestRuleSet_GetExtendsValue_Multi(t *testing.T) {

	yaml := `extends:
  -
    - spectral:oas
    - recommended
rules:
 fish-cakes:
   description: yummy sea food
   recommended: true
   type: style
   given: "$.some.JSON.PATH"
   then:
     field: nextSteps
     function: cookForTenMins`

	rs, err := CreateRuleSetFromData([]byte(yaml))
	assert.NoError(t, err)
	assert.Len(t, rs.Rules, 1)
	assert.NotNil(t, rs.GetExtendsValue())
	assert.Equal(t, "recommended", rs.GetExtendsValue()["spectral:oas"])

}

func TestRuleSet_GetExtendsValue_Multi_Noflag(t *testing.T) {

	yaml := `extends:
  - spectral:oas
rules:
 fish-cakes:
   description: yummy sea food
   recommended: true
   type: style
   given: "$.some.JSON.PATH"
   then:
     field: nextSteps
     function: cookForTenMins`

	rs, err := CreateRuleSetFromData([]byte(yaml))
	assert.NoError(t, err)
	assert.Len(t, rs.Rules, 1)
	assert.NotNil(t, rs.GetExtendsValue())
	assert.Equal(t, "spectral:oas", rs.GetExtendsValue()["spectral:oas"])
	assert.Equal(t, "spectral:oas", rs.GetExtendsValue()["spectral:oas"]) // idempotence state check.

}

func TestRuleSet_GetConfiguredRules_All(t *testing.T) {

	// read spec and parse to dashboard.
	rs := BuildDefaultRuleSets()
	ruleSet := rs.GenerateOpenAPIDefaultRuleSet()
	assert.Len(t, ruleSet.Rules, 46)

	ruleSet = rs.GenerateOpenAPIRecommendedRuleSet()
	assert.Len(t, ruleSet.Rules, 36)

}

func TestRuleSetsModel_GenerateRuleSetFromConfig_Rec_OverrideNotFound(t *testing.T) {

	yaml := `extends:
  -
    - spectral:oas
    - recommended
rules:
 soda-pop: "off"`

	def := BuildDefaultRuleSets()
	rs, _ := CreateRuleSetFromData([]byte(yaml))
	override := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, override.Rules, 36)
	assert.Len(t, override.RuleDefinitions, 1)
}

func TestRuleSetsModel_GenerateRuleSetFromConfig_Off_OverrideNotFound(t *testing.T) {

	yaml := `extends:
  -
    - spectral:oas
    - off
rules:
 soda-pop: "warn"`

	def := BuildDefaultRuleSets()
	rs, _ := CreateRuleSetFromData([]byte(yaml))
	override := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, override.Rules, 0)
	assert.Len(t, override.RuleDefinitions, 1)
}

func TestRuleSetsModel_GenerateRuleSetFromConfig_All_OverrideNotFound(t *testing.T) {

	yaml := `extends:
  -
    - spectral:oas
    - all
rules:
 soda-pop: "warn"`

	def := BuildDefaultRuleSets()
	rs, _ := CreateRuleSetFromData([]byte(yaml))
	override := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, override.Rules, 46)
	assert.Len(t, override.RuleDefinitions, 1)
}

func TestRuleSetsModel_GenerateRuleSetFromConfig_Rec_RemoveRule(t *testing.T) {

	yaml := `extends:
  -
    - spectral:oas
    - recommended
rules:
 operation-success-response: "off"`

	def := BuildDefaultRuleSets()
	rs, _ := CreateRuleSetFromData([]byte(yaml))
	override := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, override.Rules, 35)
	assert.Len(t, override.RuleDefinitions, 1)
}

func TestRuleSetsModel_GenerateRuleSetFromConfig_Rec_SeverityInfo(t *testing.T) {

	yaml := `extends:
  -
    - spectral:oas
    - recommended
rules:
 operation-success-response: "hint"`

	def := BuildDefaultRuleSets()
	rs, _ := CreateRuleSetFromData([]byte(yaml))
	override := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, override.Rules, 36)
	assert.Equal(t, "hint", override.Rules["operation-success-response"].Severity)
}
