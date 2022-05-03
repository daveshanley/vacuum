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
        "function": "post-response-success",
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
        "function": "post-response-success",
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
	assert.Equal(t, "operations must define a success response with one of the following codes: "+
		"'900, 300, 750, 600'", results[0].Message)

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
	assert.Equal(t, "this is a test for checking truthy: 'description' must be set", results[0].Message)

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
    "alpha-test-description": {
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
    "length-test-description": {
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
    "length-test-description": {
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

func TestApplyRules_Xor_Success(t *testing.T) {

	json := `{
  "documentationUrl": "quobix.com",
  "rules": {
    "xor-test-description": {
      "description": "this is a test for checking the xor function",
      "recommended": true,
      "type": "style",
      "given": [
        "$.components.examples[*]",
        "$.paths[*][*]..content[*].examples[*]",
        "$.paths[*][*]..parameters[*].examples[*]",
        "$.components.parameters[*].examples[*]",
        "$.paths[*][*]..headers[*].examples[*]",
        "$.components.headers[*].examples[*]"
      ],
      "severity": "error",
      "then": {
        "function": "xor",
		"functionOptions" : { 
			"properties" : ["externalValue", "value"]
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

func TestApplyRules_Xor_Fail(t *testing.T) {

	json := `{
  "documentationUrl": "quobix.com",
  "rules": {
    "xor-test-description": {
      "description": "this is a test for checking the xor function",
      "recommended": true,
      "type": "style",
      "given": [
        "$.components.examples[*]",
        "$.paths[*][*]..content[*].examples[*]",
        "$.paths[*][*]..parameters[*].examples[*]",
        "$.components.parameters[*].examples[*]",
        "$.paths[*][*]..headers[*].examples[*]",
        "$.components.headers[*].examples[*]"
      ],
      "severity": "error",
      "then": {
        "function": "xor",
		"functionOptions" : { 
			"properties" : ["externalValue", "value"]
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

func TestApplyRules_BadData(t *testing.T) {

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

	burgershop := []byte("!@#$%^&*()(*&^%$#%^&*(*)))]")

	_, err = ApplyRules(rs, burgershop)
	assert.Error(t, err)
}

func TestApplyRules_CircularReferences(t *testing.T) {

	burgershop, _ := ioutil.ReadFile("../model/test_files/circular-tests.yaml")

	// circular references can still be extracted, even without a ruleset.

	results, _ := ApplyRules(nil, burgershop)
	assert.Len(t, results, 3)
}

func TestApplyRules_LengthSuccess_Description_Rootnode(t *testing.T) {

	json := `{
  "documentationUrl": "quobix.com",
  "rules": {
    "length-test-description": {
      "description": "this is a test for checking the length function",
      "recommended": true,
      "type": "style",
      "given": "$",
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

func TestApplyRules_Length_Description_BadPath(t *testing.T) {

	json := `{
  "documentationUrl": "quobix.com",
  "rules": {
    "length-test-description": {
      "description": "this is a test for checking the length function",
      "recommended": true,
      "type": "style",
      "given": "I AM NOT A PATH",
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

	_, err = ApplyRules(rs, burgershop)

	// TODO: this doesn't return any errors yet, we need to change the signature of the ApplyRules function
	// to return an array of errors, no a single one.
	assert.NoError(t, err)

}

func TestApplyRules_Length_Description_BadConfig(t *testing.T) {

	json := `{
  "documentationUrl": "quobix.com",
  "rules": {
    "length-test-description": {
      "description": "this is a test for checking the length function",
      "recommended": true,
      "type": "style",
      "given": "$.info",
      "severity": "error",
      "then": {
        "function": "length",
		"field": "required",
		"functionOptions" : {
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

	assert.Len(t, results, 1)
	assert.NoError(t, err)

}

func TestApplyRulesToRuleSet_Length_Description_BadPath(t *testing.T) {

	json := `{
  "documentationUrl": "quobix.com",
  "rules": {
    "length-test-description": {
      "description": "this is a test for checking the length function",
      "recommended": true,
      "type": "style",
      "given": "I AM NOT A PATH",
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

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    burgershop,
	}

	result := ApplyRulesToRuleSet(rse)

	assert.Len(t, result.Errors, 1)
}

func TestApplyRulesToRuleSet_CircularReferences(t *testing.T) {

	json := `{
  "documentationUrl": "quobix.com",
  "rules": {
    "length-test-description": {
      "description": "this is a test for checking the length function",
      "recommended": true,
      "type": "style",
      "given": "$",
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

	burgershop, _ := ioutil.ReadFile("../model/test_files/circular-tests.yaml")

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    burgershop,
	}

	result := ApplyRulesToRuleSet(rse)

	assert.Len(t, result.Results, 3)
	assert.Equal(t, result.Results[0].Rule.Id, "circular-references")
	assert.Equal(t, result.Results[1].Rule.Id, "circular-references")
	assert.Equal(t, result.Results[2].Rule.Id, "circular-references")

}
