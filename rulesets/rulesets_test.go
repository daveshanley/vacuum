package rulesets

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var totalRules = 53
var totalRecommendedRules = 42

func TestBuildDefaultRuleSets(t *testing.T) {

	rs := BuildDefaultRuleSets()
	assert.NotNil(t, rs.GenerateOpenAPIDefaultRuleSet())
	assert.Len(t, rs.GenerateOpenAPIDefaultRuleSet().Rules, totalRules)

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
	assert.Len(t, ruleSet.Rules, totalRules)

	ruleSet = rs.GenerateOpenAPIRecommendedRuleSet()
	assert.Len(t, ruleSet.Rules, totalRecommendedRules)

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
	assert.Len(t, override.Rules, totalRecommendedRules)
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
	rs, err := CreateRuleSetFromData([]byte(yaml))
	assert.NoError(t, err)
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
	assert.Len(t, override.Rules, totalRules)
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
	assert.Len(t, override.Rules, totalRecommendedRules-1)
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
	assert.Len(t, override.Rules, totalRecommendedRules)
	assert.Equal(t, "hint", override.Rules["operation-success-response"].Severity)
}

func TestRuleSetsModel_GenerateRuleSetFromConfig_Off_EnableRules(t *testing.T) {

	yaml := `extends:
  -
    - spectral:oas
    - off
rules:
 operation-success-response: true
 info-contact: true
 `

	def := BuildDefaultRuleSets()
	rs, _ := CreateRuleSetFromData([]byte(yaml))
	override := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, override.Rules, 2)
	assert.Equal(t, "warn", override.Rules["operation-success-response"].Severity)
	assert.Equal(t, "warn", override.Rules["info-contact"].Severity)
}

func TestRuleSetsModel_GenerateRuleSetFromConfig_Off_EnableRulesNotFound(t *testing.T) {

	yaml := `extends:
  -
    - spectral:oas
    - off
rules:
 chewy-dinner: true
 burpy-baby: true
 `

	def := BuildDefaultRuleSets()
	rs, _ := CreateRuleSetFromData([]byte(yaml))
	override := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, override.Rules, 0)

}

func TestRuleSetsModel_GenerateRuleSetFromConfig_All_NewRule(t *testing.T) {

	yaml := `extends:
  -
    - spectral:oas
    - all
rules:
 fish-cakes:
   description: yummy sea food
   recommended: true
   type: style
   given: "$.some.JSON.PATH"
   then:
     field: nextSteps
     function: cookForTenMin`

	def := BuildDefaultRuleSets()
	rs, _ := CreateRuleSetFromData([]byte(yaml))
	newrs := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, newrs.Rules, totalRules+1)
	assert.Equal(t, true, newrs.Rules["fish-cakes"].Recommended)
	assert.Equal(t, "yummy sea food", newrs.Rules["fish-cakes"].Description)

}

func TestRuleSetsModel_GenerateRuleSetFromConfig_All_NewRuleReplace(t *testing.T) {

	yaml := `extends:
  -
    - spectral:oas
    - all
rules:
 info-contact:
   description: yummy sea food
   recommended: true
   type: style
   given: "$.some.JSON.PATH"
   then:
     field: nextSteps
     function: cookForTenMin`

	def := BuildDefaultRuleSets()
	rs, _ := CreateRuleSetFromData([]byte(yaml))
	repl := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, repl.Rules, totalRules)
	assert.Equal(t, true, repl.Rules["info-contact"].Recommended)
	assert.Equal(t, "yummy sea food", repl.Rules["info-contact"].Description)

}

func TestRuleSetsModel_GenerateRuleSetFromConfig_Off_CustomRule(t *testing.T) {

	yaml := `extends:
  -
    - spectral:oas
    - all
rules:
 info-contact:
   description: yummy sea food
   recommended: true
   type: style
   given: "$.some.JSON.PATH"
   then:
     field: nextSteps
     function: cookForTenMin`

	def := BuildDefaultRuleSets()
	rs, _ := CreateRuleSetFromData([]byte(yaml))
	repl := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, repl.Rules, totalRules)
	assert.Equal(t, true, repl.Rules["info-contact"].Recommended)
	assert.Equal(t, "yummy sea food", repl.Rules["info-contact"].Description)

}

func TestRuleSetsModel_GenerateRuleSetFromConfig_Off_RuleCategory(t *testing.T) {

	yaml := `extends: [[spectral:oas, off]]
rules:
  check-title-is-exactly-this:
    description: Check the title of the spec is exactly, 'this specific thing'
    severity: error
    recommended: true
    formats: [oas2, oas3]
    given: $.info.title
    then:
      field: title
      function: pattern
      functionOptions:
        match: 'this specific thing'
    howToFix: Make sure the title matches 'this specific thing'
    category:
      id: schemas`

	def := BuildDefaultRuleSets()
	rs, _ := CreateRuleSetFromData([]byte(yaml))
	repl := def.GenerateRuleSetFromSuppliedRuleSet(rs)
	assert.Len(t, repl.Rules, 1)
	assert.Equal(t, "schemas", repl.Rules["check-title-is-exactly-this"].RuleCategory.Id)

}
