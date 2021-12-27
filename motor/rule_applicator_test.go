package motor

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

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
	assert.Equal(t, "'description' must be truthy", results[0].Message)

}

func TestApplyRules_TruthyTest_MultipleElements_Fail(t *testing.T) {

	json := `{
  "documentationUrl": "quobix.com",
  "rules": {
    "truthy-test": {
      "description": "this is a test for checking truthy",
      "recommended": true,
      "type": "style",
      "given": "$.info.contact",
      "severity": "error",
      "then": [
		{
        	"function": "truthy",
			"field": "name"
		},
		{
        	"function": "truthy",
			"field": "url"
		},
		{
        	"function": "truthy",
			"field": "email"
		}]

    }
  }
}
`
	rc := CreateRuleComposer()
	rs, _ := rc.ComposeRuleSet([]byte(json))
	burgershop, _ := ioutil.ReadFile("../model/test_files/burgershop.openapi.yaml")

	results, err := ApplyRules(rs, burgershop)
	assert.NoError(t, err)
	assert.Len(t, results, 2)

}

func TestApplyRules_LengthTestFail(t *testing.T) {

	json := `{
  "documentationUrl": "quobix.com",
  "rules": {
    "length-test": {
      "description": "this is a test for checking the length function",
      "recommended": true,
      "type": "style",
      "given": "$.paths./burgers.post.requestBody.content.application/json",
      "severity": "error",
      "then": {
        "function": "length",
		"field": "examples",
		"functionOptions" : { 
			"max" : "1"
		}
      }
    }
  }
}
`
	rc := CreateRuleComposer()
	rs, err := rc.ComposeRuleSet([]byte(json))
	assert.NoError(t, err)

	burgershop, _ := ioutil.ReadFile("../model/test_files/burgershop.openapi.yaml")

	results, err := ApplyRules(rs, burgershop)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "'examples' must not be longer/greater than '1'", results[0].Message)

}

func TestApplyRules_LengthTestSuccess(t *testing.T) {

	json := `{
  "documentationUrl": "quobix.com",
  "rules": {
    "length-test": {
      "description": "this is a test for checking the length function",
      "recommended": true,
      "type": "style",
      "given": "$.paths./burgers.post.requestBody.content.application/json",
      "severity": "error",
      "then": {
        "function": "length",
		"field": "examples",
		"functionOptions" : { 
			"min" : "2",
			"max" : "4"
		}
      }
    }
  }
}
`
	rc := CreateRuleComposer()
	rs, err := rc.ComposeRuleSet([]byte(json))
	assert.NoError(t, err)

	burgershop, _ := ioutil.ReadFile("../model/test_files/burgershop.openapi.yaml")

	results, err := ApplyRules(rs, burgershop)
	assert.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestApplyRules_PatternTestSuccess_NotMatch(t *testing.T) {

	json := `{
  "documentationUrl": "quobix.com",
  "rules": {
    "pattern-test-description": {
      "description": "this is a test for checking the pattern function",
      "recommended": true,
      "type": "style",
      "given": "$..description",
      "severity": "error",
      "then": {
        "function": "pattern",
		"functionOptions" : { 
			"notMatch" : "eval\\("
		}
      }
    }
  }
}
`
	rc := CreateRuleComposer()
	rs, err := rc.ComposeRuleSet([]byte(json))
	assert.NoError(t, err)

	burgershop, _ := ioutil.ReadFile("../model/test_files/burgershop.openapi.yaml")

	results, err := ApplyRules(rs, burgershop)
	assert.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestApplyRules_AlphabeticalTestFail_Tags(t *testing.T) {

	json := `{
  "documentationUrl": "quobix.com",
  "rules": {
    "pattern-test-description": {
      "description": "this is a test for checking the alphabetical function",
      "recommended": true,
      "type": "style",
      "given": "$.tags",
      "severity": "error",
      "then": {
        "function": "alphabetical",
		"functionOptions" : { 
			"keyedBy" : "name"
		}
      }
    }
  }
}
`
	rc := CreateRuleComposer()
	rs, err := rc.ComposeRuleSet([]byte(json))
	assert.NoError(t, err)

	burgershop, _ := ioutil.ReadFile("../model/test_files/burgershop.openapi.yaml")

	results, err := ApplyRules(rs, burgershop)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestApplyRules_LengthFail_Tags(t *testing.T) {

	json := `{
  "documentationUrl": "quobix.com",
  "rules": {
    "pattern-test-description": {
      "description": "this is a test for checking the length function",
      "recommended": true,
      "type": "style",
      "given": "$.tags",
      "severity": "error",
      "then": {
        "function": "length",
		"functionOptions" : { 
			"max" : "1"
		}
      }
    }
  }
}
`
	rc := CreateRuleComposer()
	rs, err := rc.ComposeRuleSet([]byte(json))
	assert.NoError(t, err)

	burgershop, _ := ioutil.ReadFile("../model/test_files/burgershop.openapi.yaml")

	results, err := ApplyRules(rs, burgershop)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestApplyRules_LengthSuccess_Description(t *testing.T) {

	json := `{
  "documentationUrl": "quobix.com",
  "rules": {
    "pattern-test-description": {
      "description": "this is a test for checking the length function",
      "recommended": true,
      "type": "style",
      "given": "$.components.schemas.Burger",
      "severity": "error",
      "then": {
        "function": "length",
		"field": "required",
		"functionOptions" : { 
			"max" : "2"
		}
      }
    }
  }
}
`
	rc := CreateRuleComposer()
	rs, err := rc.ComposeRuleSet([]byte(json))
	assert.NoError(t, err)

	burgershop, _ := ioutil.ReadFile("../model/test_files/burgershop.openapi.yaml")

	results, err := ApplyRules(rs, burgershop)
	assert.NoError(t, err)
	assert.Len(t, results, 0)
}
