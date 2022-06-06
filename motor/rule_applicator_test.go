package motor

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
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
	assert.Len(t, results, 0)
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

func TestRuleSet_ContactProperties(t *testing.T) {

	yml := `info:
  contact:
    name: pizza
    email: monkey`

	rules := make(map[string]*model.Rule)
	rules["contact-properties"] = rulesets.GetContactPropertiesRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	results, _ := ApplyRules(rs, []byte(yml))
	assert.NotNil(t, results)
	assert.Equal(t, "Contact details are incomplete: 'url' must be set", results[0].Message)

}

func TestRuleSet_InfoContact(t *testing.T) {

	yml := `info:
  title: Terrible API Spec
  description: No operations, no contact, useless.`

	rules := make(map[string]*model.Rule)
	rules["info-contact"] = rulesets.GetInfoContactRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	results, _ := ApplyRules(rs, []byte(yml))
	assert.NotNil(t, results)
	assert.Equal(t, "Info section is missing contact details: 'contact' must be set", results[0].Message)

}

func TestRuleSet_InfoDescription(t *testing.T) {

	yml := `info:
  title: Terrible API Spec
  contact:
    name: rubbish
    email: no@acme.com`

	rules := make(map[string]*model.Rule)
	rules["info-description"] = rulesets.GetInfoDescriptionRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	results, _ := ApplyRules(rs, []byte(yml))
	assert.NotNil(t, results)
	assert.Equal(t, "Info section is missing a description: 'description' must be set", results[0].Message)

}

func TestRuleSet_InfoLicense(t *testing.T) {

	yml := `info:
  title: Terrible API Spec
  description: really crap
  contact:
    name: rubbish
    email: no@acme.com`

	rules := make(map[string]*model.Rule)
	rules["info-license"] = rulesets.GetInfoLicenseRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	results, _ := ApplyRules(rs, []byte(yml))
	assert.NotNil(t, results)
	assert.Equal(t, "Info section should contain a license: 'license' must be set", results[0].Message)

}

func TestRuleSet_InfoLicenseUrl(t *testing.T) {

	yml := `info:
  title: Terrible API Spec
  description: really crap
  contact:
    name: rubbish
    email: no@acme.com
  license:
      name: Cake`

	rules := make(map[string]*model.Rule)
	rules["license-url"] = rulesets.GetInfoLicenseUrlRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	results, _ := ApplyRules(rs, []byte(yml))
	assert.NotNil(t, results)
	assert.Equal(t, "License should contain an url: 'url' must be set", results[0].Message)

}

func TestRuleSet_NoEvalInMarkdown(t *testing.T) {

	yml := `info:
  description: this has no eval('alert(1234') impact in vacuum, but JS tools might suffer.`

	rules := make(map[string]*model.Rule)
	rules["no-eval-in-markdown"] = rulesets.GetNoEvalInMarkdownRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	results, _ := ApplyRules(rs, []byte(yml))
	assert.NotNil(t, results)
	assert.Equal(t, "description contains content with 'eval\\(', forbidden", results[0].Message)

}

func TestRuleSet_NoScriptInMarkdown(t *testing.T) {

	yml := `info:
  description: this has no impact in vacuum, <script>alert('XSS for you')</script>`

	rules := make(map[string]*model.Rule)
	rules["no-script-tags-in-markdown"] = rulesets.GetNoScriptTagsInMarkdownRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	results, _ := ApplyRules(rs, []byte(yml))
	assert.NotNil(t, results)
	assert.Equal(t, "description contains content with '<script', forbidden",
		results[0].Message)

}

func TestRuleSet_TagsAlphabetical(t *testing.T) {

	yml := `tags:
  - name: zebra
  - name: chicken
  - name: puppy`

	rules := make(map[string]*model.Rule)
	rules["openapi-tags-alphabetical"] = rulesets.GetOpenApiTagsAlphabeticalRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	results, _ := ApplyRules(rs, []byte(yml))
	assert.NotNil(t, results)
	assert.Equal(t, "Tags must be in alphabetical order: 'chicken' must be placed before 'zebra' (alphabetical)",
		results[0].Message)

}

func TestRuleSet_TagsMissing(t *testing.T) {

	yml := `info:
  contact:
    name: Duck
paths:
  /hi:
    get:
      description: I love fresh herbs.
components:
  schemas:
    Ducky:
      type: string`

	rules := make(map[string]*model.Rule)
	rules["openapi-tags"] = rulesets.GetOpenApiTagsRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	results, _ := ApplyRules(rs, []byte(yml))
	assert.NotNil(t, results)
	assert.Equal(t, "Top level spec 'tags' must not be empty, and must be an array: 'tags', is missing and is required",
		results[0].Message)

}

func TestRuleSet_TagsNotArray(t *testing.T) {

	yml := `info:
  contact:
    name: Duck
tags: none
paths:
  /hi:
    get:
      description: I love fresh herbs.
components:
  schemas:
    Ducky:
      type: string`

	rules := make(map[string]*model.Rule)
	rules["openapi-tags"] = rulesets.GetOpenApiTagsRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	results, _ := ApplyRules(rs, []byte(yml))
	assert.NotNil(t, results)
	assert.Equal(t, "Top level spec 'tags' must not be empty, and must be an array: Invalid type. Expected: array, given: string",
		results[0].Message)

}

func TestRuleSet_TagsWrongType(t *testing.T) {

	yml := `info:
  contact:
    name: Duck
tags:
  - lemons
paths:
  /hi:
    get:
      description: I love fresh herbs.
components:
  schemas:
    Ducky:
      type: string`

	rules := make(map[string]*model.Rule)
	rules["openapi-tags"] = rulesets.GetOpenApiTagsRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	results, _ := ApplyRules(rs, []byte(yml))
	assert.NotNil(t, results)
	assert.Equal(t, "Top level spec 'tags' must not be empty, and must be an array: Invalid type. Expected: object, given: string",
		results[0].Message)

}

func TestRuleSet_OperationIdInvalidInUrl(t *testing.T) {

	yml := `paths:
  /hi:
    get:
      operationId: nice rice
    post:
      operationId: wow^cool
    delete:
      operationId: this-works`

	rules := make(map[string]*model.Rule)
	rules["operation-operationId-valid-in-url"] = rulesets.GetOperationIdValidInUrlRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	results, _ := ApplyRules(rs, []byte(yml))
	assert.NotNil(t, results)
	assert.Len(t, results, 2)

}

func TestRuleSetGetOperationTagsRule(t *testing.T) {

	yml := `paths:
  /hi:
    get:
      tags:
        - fresh
    post:
      operationId: cool
    delete:
      operationId: jam`

	rules := make(map[string]*model.Rule)
	rules["operation-tags"] = rulesets.GetOperationTagsRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	results, _ := ApplyRules(rs, []byte(yml))
	assert.NotNil(t, results)
	assert.Len(t, results, 2)

}

func TestRuleSetGetOperationTagsMultipleRule(t *testing.T) {

	yml := `paths:
  /hi:
    get:
      tags:
        - fresh
    post:
      operationId: cool
    delete:
      operationId: jam
  /bye:
    get:
      tags:
        - hot
    post:
      operationId: coffee`

	rules := make(map[string]*model.Rule)
	rules["operation-tags"] = rulesets.GetOperationTagsRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	results, _ := ApplyRules(rs, []byte(yml))
	assert.NotNil(t, results)
	assert.Len(t, results, 3)

}

func TestRuleSetGetPathDeclarationsMustExist(t *testing.T) {

	yml := `paths:
  /hi/{there}:
    get:
      operationId: a
  /oh/{}:
    get:
      operationId: b`

	rules := make(map[string]*model.Rule)
	rules["path-declarations-must-exist"] = rulesets.GetPathDeclarationsMustExistRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	results, _ := ApplyRules(rs, []byte(yml))
	assert.NotNil(t, results)
	assert.Len(t, results, 1)
	assert.Equal(t, "Path parameter declarations must not be empty ex. '/api/{}' is invalid:"+
		" matches the expression '{}'", results[0].Message)

}

func TestRuleSetNoPathTrailingSlashTest(t *testing.T) {

	yml := `paths:
  /hi/{there}/:
    get:
      operationId: a
  /oh/no/:
    get:
      operationId: b
  /halp:
    get:
      operationId: b`

	rules := make(map[string]*model.Rule)
	rules["path-keys-no-trailing-slash"] = rulesets.GetPathNoTrailingSlashRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	results, _ := ApplyRules(rs, []byte(yml))
	assert.NotNil(t, results)
	assert.Len(t, results, 2)

}

func TestRuleSetNoPathQueryString(t *testing.T) {

	yml := `paths:
  /hi/{there}?oh=yeah:
    get:
      operationId: a
  /woah/slow/down:
    get:
      operationId: b
  /moving?too=fast:
    get:
      operationId: b`

	rules := make(map[string]*model.Rule)
	rules["path-not-include-query"] = rulesets.GetPathNotIncludeQueryRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	results, _ := ApplyRules(rs, []byte(yml))
	assert.NotNil(t, results)
	assert.Len(t, results, 2)

}

func TestRuleTagDescriptionRequiredRule(t *testing.T) {

	yml := `tags:
  - name: pizza
    description: nice
  - name: cinnamon
  - name: lemons
    description: zing`

	rules := make(map[string]*model.Rule)
	rules["tag-description"] = rulesets.GetTagDescriptionRequiredRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	results, _ := ApplyRules(rs, []byte(yml))
	assert.NotNil(t, results)
	assert.Len(t, results, 1)
	assert.Equal(t, "Tag must have a description defined: 'description' must be set", results[0].Message)

}

func TestRuleOAS2APIHostRule(t *testing.T) {

	yml := `swagger: 2.0
paths:
  /nice:
    get:
      description: fresh fish
definitions: 
  Cake: 
    description: and tea`

	rules := make(map[string]*model.Rule)
	rules["oas2-api-host"] = rulesets.GetOAS2APIHostRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	results, _ := ApplyRules(rs, []byte(yml))
	assert.NotNil(t, results)
	assert.Len(t, results, 1)

}

func TestRuleOAS2APIHostRule_Success(t *testing.T) {

	yml := `swagger: 2.0
host: https://quobix.com
paths:
  /nice:
    get:
      description: fresh fish
definitions: 
  Cake: 
    description: and tea`

	rules := make(map[string]*model.Rule)
	rules["oas2-api-host"] = rulesets.GetOAS2APIHostRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	results, _ := ApplyRules(rs, []byte(yml))
	assert.Nil(t, results)
	assert.Len(t, results, 0)

}

func TestRuleOAS2APISchemesRule(t *testing.T) {

	yml := `swagger: 2.0
no-schemes: [yeah]
paths:
  /nice:
    get:
      description: fresh fish
definitions: 
  Cake: 
    description: and tea`

	rules := make(map[string]*model.Rule)
	rules["oas2-api-schemes"] = rulesets.GetOAS2APISchemesRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	results, _ := ApplyRules(rs, []byte(yml))
	assert.NotNil(t, results)
	assert.Len(t, results, 1)

}

func TestRuleOAS2APISchemesRule_Success(t *testing.T) {

	yml := `swagger: 2.0
schemes: [http, https]
paths:
  /nice:
    get:
      description: fresh fish
definitions: 
  Cake: 
    description: and tea`

	rules := make(map[string]*model.Rule)
	rules["oas2-api-schemes"] = rulesets.GetOAS2APISchemesRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	results, _ := ApplyRules(rs, []byte(yml))
	assert.Nil(t, results)
	assert.Len(t, results, 0)

}

func TestRuleOAS2HostNotExampleRule(t *testing.T) {

	yml := `swagger: 2.0
host: https://quobix.com`

	rules := make(map[string]*model.Rule)
	rules["oas2-host-not-example"] = rulesets.GetOAS2HostNotExampleRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	results, _ := ApplyRules(rs, []byte(yml))
	assert.Nil(t, results)
	assert.Len(t, results, 0)

}

func TestRuleOAS2HostNotExampleRule_Fail(t *testing.T) {

	yml := `swagger: 2.0
host: https://example.com`

	rules := make(map[string]*model.Rule)
	rules["oas2-host-not-example"] = rulesets.GetOAS2HostNotExampleRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	results, _ := ApplyRules(rs, []byte(yml))
	assert.NotNil(t, results)
	assert.Len(t, results, 1)

}

func TestRuleOAS2HostTrailingSlashRule_Fail(t *testing.T) {

	yml := `swagger: 2.0
host: https://quobix.com/`

	rules := make(map[string]*model.Rule)
	rules["oas2-host-trailing-slash"] = rulesets.GetOAS2HostTrailingSlashRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	results, _ := ApplyRules(rs, []byte(yml))
	assert.NotNil(t, results)
	assert.Len(t, results, 1)

}

func TestRuleOAS2HostTrailingSlashRule(t *testing.T) {

	yml := `swagger: 2.0
host: https://quobix.com`

	rules := make(map[string]*model.Rule)
	rules["oas2-host-trailing-slash"] = rulesets.GetOAS2HostTrailingSlashRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	results, _ := ApplyRules(rs, []byte(yml))
	assert.Nil(t, results)
	assert.Len(t, results, 0)

}

func TestRuleOAS3HostNotExampleRule_Fail(t *testing.T) {

	yml := `openapi: 3.0
servers:
  - url: https://example.com/
`

	rules := make(map[string]*model.Rule)
	rules["oas3-host-not-example"] = rulesets.GetOAS3HostNotExampleRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	results, _ := ApplyRules(rs, []byte(yml))
	assert.NotNil(t, results)
	assert.Len(t, results, 1)

}

func TestRuleOAS3HostNotExampleRule_Success(t *testing.T) {

	yml := `openapi: 3.0
servers:
  - url: https://quobix.com/
`

	rules := make(map[string]*model.Rule)
	rules["oas3-host-not-example"] = rulesets.GetOAS3HostNotExampleRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	results, _ := ApplyRules(rs, []byte(yml))
	assert.Nil(t, results)
	assert.Len(t, results, 0)

}

func TestPetstoreSpecAgainstDefaultRuleSet(t *testing.T) {

	b, _ := ioutil.ReadFile("../model/test_files/petstorev3.json")
	rs := rulesets.BuildDefaultRuleSets()
	results, err := ApplyRules(rs.GenerateOpenAPIDefaultRuleSet(), b)

	assert.NoError(t, err)
	assert.NotNil(t, results)

}

func TestStripeSpecAgainstDefaultRuleSet(t *testing.T) {

	b, _ := ioutil.ReadFile("../model/test_files/stripe.yaml")
	rs := rulesets.BuildDefaultRuleSets()
	results, err := ApplyRules(rs.GenerateOpenAPIDefaultRuleSet(), b)

	assert.NoError(t, err)
	assert.NotNil(t, results)

}

func Benchmark_K8sSpecAgainstDefaultRuleSet(b *testing.B) {
	m, _ := ioutil.ReadFile("../model/test_files/k8s.json")
	rs := rulesets.BuildDefaultRuleSets()
	for n := 0; n < b.N; n++ {
		ApplyRules(rs.GenerateOpenAPIDefaultRuleSet(), m)
	}
}
