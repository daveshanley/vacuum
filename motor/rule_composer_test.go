package motor

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCreateRuleComposer(t *testing.T) {
	assert.NotNil(t, CreateRuleComposer())
}

func TestRuleComposer_ComposeRuleSet_MissingData(t *testing.T) {
	rc := CreateRuleComposer()
	_, err := rc.ComposeRuleSet([]byte(""))
	assert.Error(t, err)

}

func TestRuleComposer_ComposeRuleSet_NoRules(t *testing.T) {
	// this should not work.
	json := `{
  "documentationUrl": "quobix.com",
  "rules": {
  }
}
`
	rc := CreateRuleComposer()
	_, err := rc.ComposeRuleSet([]byte(json))

	assert.Error(t, err)

}

func TestRuleComposer_ComposeRuleSet(t *testing.T) {
	// this should not work, there is no function called 'cookForTenMins'
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
	rc := CreateRuleComposer()
	_, err := rc.ComposeRuleSet([]byte(json))
	assert.Error(t, err)

}
