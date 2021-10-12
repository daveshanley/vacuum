package motor

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestApplyRules_Hello(t *testing.T) {

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

func TestApplyRules_PostResponseSuccess(t *testing.T) {

	json := `{
  "documentationUrl": "quobix.com",
  "rules": {
    "hello-test": {
      "description": "this is a test for checking basic mechanics",
      "recommended": true,
      "type": "style",
      "given": "$.paths.*.post.responses",
      "then": {
        "function": "post_response_success",
		"functionOptions" : { 
			"properties": [
				"200", "201", "202", "204"
			]
		}
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
	assert.Len(t, results, 0)
}

func TestApplyRules_PostResponseFailure(t *testing.T) {

	// use a bunch of error codes that won't exist.

	json := `{
  "documentationUrl": "quobix.com",
  "rules": {
    "hello-test": {
      "description": "this is a test for checking basic mechanics",
      "recommended": true,
      "type": "style",
      "given": "$.paths.*.post.responses",
      "then": {
        "function": "post_response_success",
		"functionOptions" : { 
			"properties": [
				"900", "300", "750", "600" 
			]
		}
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
	assert.Equal(t, "operations must define a success response with one of the following codes: 900, 300, 750, 600", results[0].Message)

}

func TestApplyRules_TruthyTest(t *testing.T) {

	json := `{
  "documentationUrl": "quobix.com",
  "rules": {
    "truthy-test": {
      "description": "this is a test for checking truthy",
      "recommended": true,
      "type": "style",
      "given": "$.tags[*]",
      "severity": "error",
      "then": {
        "function": "truthy",
		"field": "description"
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
	assert.Equal(t, "operations must define a success response with one of the following codes: 900, 300, 750, 600", results[0].Message)

}
