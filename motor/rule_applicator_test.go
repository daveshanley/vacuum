package motor

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestApplyRules(t *testing.T) {

	json := `{
  "documentationUrl": "quobix.com",
  "rules": {
    "hello-test": {
      "description": "this is a test for checking basic mechanics",
      "recommended": true,
      "type": "style",
      "given": "$.info",
      "then": {
        "function": "hello"
      }
    }
  }
}
`
	rc := CreateRuleComposer()
	rs, _ := rc.ComposeRuleSet([]byte(json))
	burgershop, _ := ioutil.ReadFile("../model/test_files/burgershop.openapi.yaml")

	results, err := ApplyRules(rs, burgershop)
	assert.NoError(t, err)
	assert.Len(t, results, 1)

}
