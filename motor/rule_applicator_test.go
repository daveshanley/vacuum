package motor

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
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
	burgershop, _ := os.ReadFile("../model/test_files/burgershop.openapi.yaml")

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    burgershop,
	}
	results := ApplyRulesToRuleSet(rse)

	assert.Len(t, results.Results, 0)
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
	burgershop, _ := os.ReadFile("../model/test_files/burgershop.openapi.yaml")

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    burgershop,
	}
	results := ApplyRulesToRuleSet(rse)

	assert.Len(t, results.Results, 1)
	assert.Equal(t, "operations must define a success response with one of the following codes: "+
		"'900, 300, 750, 600'", results.Results[0].Message)

}

func TestApplyRules_TruthyTest_MultipleElements_Fail(t *testing.T) {

	json := fmt.Sprintf(`{
  "documentationUrl": "quobix.com",
  "rules": {
    "truthy-test": {
      "description": "this is a test for checking truthy",
      "recommended": true,
      "type": "style",
      "given": "$.info.contact",
      "severity": "%s",
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
`, model.SeverityError)
	rc := CreateRuleComposer()
	rs, _ := rc.ComposeRuleSet([]byte(json))
	burgershop, _ := os.ReadFile("../model/test_files/burgershop.openapi.yaml")

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    burgershop,
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)
	assert.Len(t, results.Results, 2)

}

func TestApplyRules_LengthTestFail(t *testing.T) {

	json := fmt.Sprintf(`{
  "documentationUrl": "quobix.com",
  "rules": {
    "length-test": {
      "description": "this is a test for checking the length function",
      "recommended": true,
      "type": "style",
      "given": "$.paths./burgers.post.requestBody.content.application/json",
      "severity": "%s",
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
`, model.SeverityError)
	rc := CreateRuleComposer()
	rs, err := rc.ComposeRuleSet([]byte(json))
	assert.NoError(t, err)

	burgershop, _ := os.ReadFile("../model/test_files/burgershop.openapi.yaml")

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    burgershop,
	}
	results := ApplyRulesToRuleSet(rse)

	assert.Len(t, results.Errors, 0)
	assert.Len(t, results.Results, 1)
	assert.Equal(t, "this is a test for checking the length function: 'examples' must not be longer/greater than '1'", results.Results[0].Message)

}

func TestApplyRules_LengthTestSuccess(t *testing.T) {

	json := fmt.Sprintf(`{
  "documentationUrl": "quobix.com",
  "rules": {
    "length-test": {
      "description": "this is a test for checking the length function",
      "recommended": true,
      "type": "style",
      "given": "$.paths./burgers.post.requestBody.content.application/json",
      "severity": "%s",
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
`, model.SeverityError)
	rc := CreateRuleComposer()
	rs, err := rc.ComposeRuleSet([]byte(json))
	assert.NoError(t, err)

	burgershop, _ := os.ReadFile("../model/test_files/burgershop.openapi.yaml")

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    burgershop,
	}
	results := ApplyRulesToRuleSet(rse)

	assert.Len(t, results.Errors, 0)
	assert.Len(t, results.Results, 0)
}

func TestApplyRules_PatternTestSuccess_NotMatch(t *testing.T) {

	json := fmt.Sprintf(`{
  "documentationUrl": "quobix.com",
  "rules": {
    "pattern-test-description": {
      "description": "this is a test for checking the pattern function",
      "recommended": true,
      "type": "style",
      "given": "$..description",
      "severity": "%s",
      "then": {
        "function": "pattern",
		"functionOptions" : { 
			"notMatch" : "eval\\("
		}
      }
    }
  }
}
`, model.SeverityError)
	rc := CreateRuleComposer()
	rs, err := rc.ComposeRuleSet([]byte(json))
	assert.NoError(t, err)

	burgershop, _ := os.ReadFile("../model/test_files/burgershop.openapi.yaml")

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    burgershop,
	}
	results := ApplyRulesToRuleSet(rse)

	assert.Len(t, results.Errors, 0)
	assert.Len(t, results.Results, 0)
}

func TestApplyRules_AlphabeticalTestFail_Tags(t *testing.T) {

	json := fmt.Sprintf(`{
  "documentationUrl": "quobix.com",
  "rules": {
    "alpha-test-description": {
      "description": "this is a test for checking the alphabetical function",
      "recommended": true,
      "type": "style",
      "given": "$.tags",
      "severity": "%s",
      "then": {
        "function": "alphabetical",
		"functionOptions" : { 
			"keyedBy" : "name"
		}
      }
    }
  }
}
`, model.SeverityError)
	rc := CreateRuleComposer()
	rs, err := rc.ComposeRuleSet([]byte(json))
	assert.NoError(t, err)

	burgershop, _ := os.ReadFile("../model/test_files/burgershop.openapi.yaml")

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    burgershop,
	}
	results := ApplyRulesToRuleSet(rse)

	assert.Len(t, results.Errors, 0)
	assert.Len(t, results.Results, 0)
}

func TestApplyRules_LengthFail_Tags(t *testing.T) {

	json := fmt.Sprintf(`{
  "documentationUrl": "quobix.com",
  "rules": {
    "length-test-description": {
      "description": "this is a test for checking the length function",
      "recommended": true,
      "type": "style",
      "given": "$.tags",
      "severity": "%s",
      "then": {
        "function": "length",
		"functionOptions" : { 
			"max" : "1"
		}
      }
    }
  }
}
`, model.SeverityError)
	rc := CreateRuleComposer()
	rs, err := rc.ComposeRuleSet([]byte(json))
	assert.NoError(t, err)

	burgershop, _ := os.ReadFile("../model/test_files/burgershop.openapi.yaml")

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    burgershop,
	}
	results := ApplyRulesToRuleSet(rse)

	assert.Len(t, results.Errors, 0)
	assert.Len(t, results.Results, 1)
}

func TestApplyRules_LengthSuccess_Description(t *testing.T) {

	json := fmt.Sprintf(`{
  "documentationUrl": "quobix.com",
  "rules": {
    "length-test-description": {
      "description": "this is a test for checking the length function",
      "recommended": true,
      "type": "style",
      "given": "$.components.schemas.Burger",
      "severity": "%s",
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
`, model.SeverityError)
	rc := CreateRuleComposer()
	rs, err := rc.ComposeRuleSet([]byte(json))
	assert.NoError(t, err)

	burgershop, _ := os.ReadFile("../model/test_files/burgershop.openapi.yaml")

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    burgershop,
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)
	assert.Len(t, results.Results, 0)
}

func TestApplyRules_Xor_Success(t *testing.T) {

	json := fmt.Sprintf(`{
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
      "severity": "%s",
      "then": {
        "function": "xor",
		"functionOptions" : { 
			"properties" : ["externalValue", "value"]
		}
      }
    }
  }
}
`, model.SeverityError)
	rc := CreateRuleComposer()
	rs, err := rc.ComposeRuleSet([]byte(json))
	assert.NoError(t, err)

	burgershop, _ := os.ReadFile("../model/test_files/burgershop.openapi.yaml")

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    burgershop,
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Results, 0)
}

func TestApplyRules_Xor_Fail(t *testing.T) {

	json := fmt.Sprintf(`{
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
      "severity": "%s",
      "then": {
        "function": "xor",
		"functionOptions" : { 
			"properties" : ["externalValue", "value"]
		}
      }
    }
  }
}
`, model.SeverityError)
	rc := CreateRuleComposer()
	rs, err := rc.ComposeRuleSet([]byte(json))
	assert.NoError(t, err)

	burgershop, _ := os.ReadFile("../model/test_files/burgershop.openapi.yaml")

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    burgershop,
	}
	results := ApplyRulesToRuleSet(rse)

	assert.Len(t, results.Results, 0)
}

func TestApplyRules_BadData(t *testing.T) {

	json := fmt.Sprintf(`{
  "documentationUrl": "quobix.com",
  "rules": {
    "length-test": {
      "description": "this is a test for checking the length function",
      "recommended": true,
      "type": "style",
      "given": "$.paths./burgers.post.requestBody.content.application/json",
      "severity": "%s",
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
`, model.SeverityError)
	rc := CreateRuleComposer()
	rs, err := rc.ComposeRuleSet([]byte(json))
	assert.NoError(t, err)

	burgershop := []byte("!@#$%^&*()(*&^%$#%^&*(*)))]")

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    burgershop,
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 1)

}

func TestApplyRules_CircularReferences(t *testing.T) {

	burgershop, _ := os.ReadFile("../model/test_files/circular-tests.yaml")

	// circular references can still be extracted, even without a ruleset.
	rse := &RuleSetExecution{
		RuleSet: nil,
		Spec:    burgershop,
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)
	assert.Len(t, results.Results, 3)
}

func TestApplyRules_LengthSuccess_Description_Rootnode(t *testing.T) {

	json := fmt.Sprintf(`{
  "documentationUrl": "quobix.com",
  "rules": {
    "length-test-description": {
      "description": "this is a test for checking the length function",
      "recommended": true,
      "type": "style",
      "given": "$",
      "severity": "%s",
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
`, model.SeverityError)
	rc := CreateRuleComposer()
	rs, err := rc.ComposeRuleSet([]byte(json))
	assert.NoError(t, err)

	burgershop, _ := os.ReadFile("../model/test_files/burgershop.openapi.yaml")

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    burgershop,
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)
	assert.Len(t, results.Results, 0)
}

func TestApplyRules_Length_Description_BadPath(t *testing.T) {

	json := fmt.Sprintf(`{
  "documentationUrl": "quobix.com",
  "rules": {
    "length-test-description": {
      "description": "this is a test for checking the length function",
      "recommended": true,
      "type": "style",
      "given": "I AM NOT A PATH",
      "severity": "%s",
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
`, model.SeverityError)
	rc := CreateRuleComposer()
	rs, err := rc.ComposeRuleSet([]byte(json))
	assert.NoError(t, err)

	burgershop, _ := os.ReadFile("../model/test_files/burgershop.openapi.yaml")

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    burgershop,
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 1)
	assert.Len(t, results.Results, 0)

}

func TestApplyRules_Length_Description_BadConfig(t *testing.T) {

	json := fmt.Sprintf(`{
  "documentationUrl": "quobix.com",
  "rules": {
    "length-test-description": {
      "description": "this is a test for checking the length function",
      "recommended": true,
      "type": "style",
      "given": "$.info",
      "severity": "%s",
      "then": {
        "function": "length",
		"field": "required",
		"functionOptions" : {
		}
      }
    }
  }
}
`, model.SeverityError)
	rc := CreateRuleComposer()
	rs, err := rc.ComposeRuleSet([]byte(json))
	assert.NoError(t, err)

	burgershop, _ := os.ReadFile("../model/test_files/burgershop.openapi.yaml")

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    burgershop,
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)
	assert.Len(t, results.Results, 1)

}

func TestApplyRulesToRuleSet_Length_Description_BadPath(t *testing.T) {

	json := fmt.Sprintf(`{
  "documentationUrl": "quobix.com",
  "rules": {
    "length-test-description": {
      "description": "this is a test for checking the length function",
      "recommended": true,
      "type": "style",
      "given": "I AM NOT A PATH",
      "severity": "%s",
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
`, model.SeverityError)
	rc := CreateRuleComposer()
	rs, err := rc.ComposeRuleSet([]byte(json))
	assert.NoError(t, err)

	burgershop, _ := os.ReadFile("../model/test_files/burgershop.openapi.yaml")

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    burgershop,
	}
	result := ApplyRulesToRuleSet(rse)
	assert.Len(t, result.Errors, 1)
}

func TestApplyRulesToRuleSet_CircularReferences(t *testing.T) {

	json := fmt.Sprintf(`{
  "documentationUrl": "quobix.com",
  "rules": {
    "length-test-description": {
      "description": "this is a test for checking the length function",
      "recommended": true,
      "type": "style",
      "given": "$",
      "severity": "%s",
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
`, model.SeverityError)
	rc := CreateRuleComposer()
	rs, err := rc.ComposeRuleSet([]byte(json))
	assert.NoError(t, err)

	burgershop, _ := os.ReadFile("../model/test_files/circular-tests.yaml")

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    burgershop,
	}

	result := ApplyRulesToRuleSet(rse)

	assert.Len(t, result.Results, 3)
	assert.Equal(t, "resolving-references", result.Results[0].Rule.Id)

}

func TestRuleSet_ContactProperties(t *testing.T) {

	yml := `openapi: 3.1.0
info:
  contact:
    name: pizza
    email: monkey`

	rules := make(map[string]*model.Rule)
	rules["contact-properties"] = rulesets.GetContactPropertiesRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)
	assert.Equal(t, "Contact details are incomplete: `url` must be set", results.Results[0].Message)

}

func TestRuleSet_InfoContact(t *testing.T) {

	yml := `openapi: 3.1.0
info:
  title: Terrible API Spec
  description: No operations, no contact, useless.`

	rules := make(map[string]*model.Rule)
	rules["info-contact"] = rulesets.GetInfoContactRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)

	assert.NotNil(t, results)
	assert.Equal(t, "Info section is missing contact details: `contact` must be set", results.Results[0].Message)

}

func TestRuleSet_InfoDescription(t *testing.T) {

	yml := `openapi: 3.1.0
info:
  title: Terrible API Spec
  contact:
    name: rubbish
    email: no@acme.com`

	rules := make(map[string]*model.Rule)
	rules["info-description"] = rulesets.GetInfoDescriptionRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)
	assert.Equal(t, "Info section is missing a description: `description` must be set", results.Results[0].Message)

}

func TestRuleSet_InfoLicense(t *testing.T) {

	yml := `openapi: 3.1.0
info:
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

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)
	assert.Equal(t, "Info section should contain a license: `license` must be set", results.Results[0].Message)

}

func TestRuleSet_InfoLicenseUrl(t *testing.T) {

	yml := `openapi: 3.1.0
info:
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

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)

	assert.NotNil(t, results)
	assert.Equal(t, "License should contain an url: `url` must be set", results.Results[0].Message)

}

func TestRuleSet_NoEvalInMarkdown(t *testing.T) {

	yml := `openapi: 3.1.0
info:
  description: this has no eval('alert(1234') impact in vacuum, but JS tools might suffer.`

	rules := make(map[string]*model.Rule)
	rules["no-eval-in-markdown"] = rulesets.GetNoEvalInMarkdownRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)

	assert.NotNil(t, results)
	assert.Equal(t, "description contains content with `eval\\(`, forbidden", results.Results[0].Message)

}

func TestRuleSet_NoScriptInMarkdown(t *testing.T) {

	yml := `openapi: 3.1.0
info:
  description: this has no impact in vacuum, <script>alert('XSS for you')</script>`

	rules := make(map[string]*model.Rule)
	rules["no-script-tags-in-markdown"] = rulesets.GetNoScriptTagsInMarkdownRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)

	assert.NotNil(t, results)
	assert.Equal(t, "description contains content with `<script`, forbidden",
		results.Results[0].Message)

}

func TestRuleSet_TagsAlphabetical(t *testing.T) {

	yml := `openapi: 3.1.0
tags:
  - name: zebra
  - name: chicken
  - name: puppy`

	rules := make(map[string]*model.Rule)
	rules["openapi-tags-alphabetical"] = rulesets.GetOpenApiTagsAlphabeticalRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)

	assert.NotNil(t, results)
	assert.Equal(t, "Tags must be in alphabetical order: `chicken` must be placed before `zebra` (alphabetical)",
		results.Results[0].Message)

}

func TestRuleSet_TagsMissing(t *testing.T) {

	yml := `openapi: 3.1.0
info:
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

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)

	assert.NotNil(t, results)
	assert.Equal(t, "Top level spec `tags` must not be empty, and must be an array: `tags`, is missing and is required",
		results.Results[0].Message)

}

func TestRuleSet_TagsNotArray(t *testing.T) {

	yml := `openapi: 3.1.0
info:
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

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)

	assert.NotNil(t, results)
	assert.Equal(t, "Top level spec `tags` must not be empty, and must be an array: expected array, but got string",
		results.Results[0].Message)

}

func TestRuleSet_TagsWrongType(t *testing.T) {

	yml := `openapi: 3.1.0
info:
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

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)

	assert.NotNil(t, results)
	assert.Equal(t, "Top level spec `tags` must not be empty, and must be an array: expected object, but got string",
		results.Results[0].Message)

}

func TestRuleSet_OperationIdInvalidInUrl(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
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

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)
	assert.Len(t, results.Results, 2)

}

func TestRuleSetGetOperationTagsRule(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
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

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)
	assert.Len(t, results.Results, 2)

}

func TestRuleSetGetOperationTagsMultipleRule(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
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

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)
	assert.Len(t, results.Results, 3)

}

func TestRuleSetGetPathDeclarationsMustExist(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
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

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)
	assert.Len(t, results.Results, 1)
	assert.Equal(t, "Path parameter declarations must not be empty ex. `/api/{}` is invalid:"+
		" matches the expression `{}`", results.Results[0].Message)

}

func TestRuleSetNoPathTrailingSlashTest(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
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

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)
	assert.Len(t, results.Results, 2)

}

func TestRuleSetNoPathQueryString(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
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

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)
	assert.Len(t, results.Results, 2)

}

func TestRuleTagDescriptionRequiredRule(t *testing.T) {

	yml := `openapi: 3.1.0
tags:
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

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)
	assert.Len(t, results.Results, 1)
	assert.Equal(t, "Tag must have a description defined: `description` must be set", results.Results[0].Message)

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

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)
	assert.Len(t, results.Results, 1)

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

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)
	assert.Len(t, results.Results, 0)

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

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)
	assert.Len(t, results.Results, 1)

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

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)
	assert.Len(t, results.Results, 0)

}

func TestRuleOAS2HostNotExampleRule(t *testing.T) {

	yml := `swagger: 2.0
host: https://quobix.com`

	rules := make(map[string]*model.Rule)
	rules["oas2-host-not-example"] = rulesets.GetOAS2HostNotExampleRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)
	assert.Len(t, results.Results, 0)

}

func TestRuleOAS2HostNotExampleRule_Fail(t *testing.T) {

	yml := `swagger: 2.0
host: https://example.com`

	rules := make(map[string]*model.Rule)
	rules["oas2-host-not-example"] = rulesets.GetOAS2HostNotExampleRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)
	assert.Len(t, results.Results, 1)

}

func TestRuleOAS2HostTrailingSlashRule_Fail(t *testing.T) {

	yml := `swagger: 2.0
host: https://quobix.com/`

	rules := make(map[string]*model.Rule)
	rules["oas2-host-trailing-slash"] = rulesets.GetOAS2HostTrailingSlashRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)
	assert.Len(t, results.Results, 1)

}

func TestRuleOAS2HostTrailingSlashRule(t *testing.T) {

	yml := `swagger: 2.0
host: https://quobix.com`

	rules := make(map[string]*model.Rule)
	rules["oas2-host-trailing-slash"] = rulesets.GetOAS2HostTrailingSlashRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)
	assert.Len(t, results.Results, 0)

}

func TestRuleOAS3HostTrailingSlashRule(t *testing.T) {

	yml := `servers:
 - url: https://quobix.com
 - url: https://pb33f.io
`

	rules := make(map[string]*model.Rule)
	rules["oas3-host-trailing-slash"] = rulesets.GetOAS3HostTrailingSlashRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}

	results := ApplyRulesToRuleSet(rse)
	assert.NotNil(t, results)
	assert.Len(t, results.Results, 0)

}

func TestRuleOAS3HostTrailingSlashRule_Fail(t *testing.T) {

	yml := `openapi: 3.1.0
servers:
 - url: https://quobix.com/
 - url: https://pb33f.io/
`

	rules := make(map[string]*model.Rule)
	rules["oas3-host-trailing-slash"] = rulesets.GetOAS3HostTrailingSlashRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}

	results := ApplyRulesToRuleSet(rse)
	assert.NotNil(t, results)
	assert.Len(t, results.Results, 2)

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

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)
	assert.Len(t, results.Results, 1)

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

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)
	assert.Len(t, results.Results, 0)

}

type testRule struct{}

func (t *testRule) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "test",
	}
}

func (t *testRule) RunRule(nodes []*yaml.Node,
	context model.RuleFunctionContext,
) []model.RuleFunctionResult {
	panic("run away!")
}

func TestCustomRuleHandlePanic(t *testing.T) {

	rules := map[string]*model.Rule{
		"test": {
			Id:           "test",
			Formats:      model.OAS3AllFormat,
			Given:        "$",
			Recommended:  true,
			RuleCategory: model.RuleCategories[model.CategoryValidation],
			Type:         "validation",
			Severity:     model.SeverityError,
			Then: model.RuleAction{
				Function: "test",
			},
		},
	}

	set := &rulesets.RuleSet{
		DocumentationURI: "",
		Rules:            rules,
		Description:      "",
	}

	spec := []byte(`openapi: 3.1
components:
  schemas:
    none:
      type: int`)

	panicRan := false
	saveMePlease := func(r any) {
		panicRan = true
	}

	ApplyRulesToRuleSet(
		&RuleSetExecution{
			PanicFunction: saveMePlease,
			RuleSet:       set,
			Spec:          spec,
			CustomFunctions: map[string]model.RuleFunction{
				"test": &testRule{},
			},
		})

	//nolint:staticcheck // ignore this linting issue, its no a bug, it's on purpose.
	time.Sleep(100)
	assert.True(t, panicRan)

}

func TestPetstoreSpecAgainstDefaultRuleSet(t *testing.T) {

	b, _ := os.ReadFile("../model/test_files/petstorev3.json")
	rs := rulesets.BuildDefaultRuleSets()
	rse := &RuleSetExecution{
		RuleSet: rs.GenerateOpenAPIDefaultRuleSet(),
		Spec:    b,
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)
	assert.NotNil(t, results)

}

func TestStripeSpecAgainstDefaultRuleSet(t *testing.T) {

	b, _ := os.ReadFile("../model/test_files/stripe.yaml")
	rs := rulesets.BuildDefaultRuleSets()
	rse := &RuleSetExecution{
		RuleSet: rs.GenerateOpenAPIDefaultRuleSet(),
		Spec:    b,
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)
	assert.NotNil(t, results)

}

func TestRuleSet_TestBadRef(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
  /one:
    get:
      responses:
        '200':
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/one'
components:
  schemas:
    none:
      type: string`

	rules := make(map[string]*model.Rule)
	rules["openapi-tags-alphabetical"] = rulesets.GetOpenApiTagsAlphabeticalRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)

	assert.NotNil(t, results)
	assert.Equal(t, "component '#/components/schemas/one' does not exist in the specification",
		results.Results[0].Message)
	assert.Equal(t, "resolving-references", results.Results[0].RuleId)
}

func TestRuleSet_InfiniteCircularLoop(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
  /one:
    get:
      responses:
        '200':
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/one'
components:
  schemas:
    one:
      type: string
      required: [two]
      properties:
        two:
          $ref: '#/components/schemas/two'
    two:
      required: [three]
      properties:
        three:
          $ref: '#/components/schemas/one'`

	rules := make(map[string]*model.Rule)
	rules["openapi-tags-alphabetical"] = rulesets.GetOpenApiTagsAlphabeticalRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)

	assert.NotNil(t, results)
	assert.Equal(t, "Infinite circular reference detected: one: one -> two -> one [14:7]",
		results.Results[0].Message)
	assert.Equal(t, "resolving-references", results.Results[0].RuleId)
}

func Benchmark_K8sSpecAgainstDefaultRuleSet(b *testing.B) {
	m, _ := os.ReadFile("../model/test_files/k8s.json")
	rs := rulesets.BuildDefaultRuleSets()
	for n := 0; n < b.N; n++ {
		rse := &RuleSetExecution{
			RuleSet: rs.GenerateOpenAPIDefaultRuleSet(),
			Spec:    m,
		}
		results := ApplyRulesToRuleSet(rse)
		assert.Len(b, results.Errors, 0)
		assert.NotNil(b, results)
	}
}

func TestRuleSet_TestOWASP1(t *testing.T) {

	yml := `openapi: "3.1.0"
paths:
  /foo/{id}/:
    get:
      description: "get"
      parameters:
        - name: id
          in: path
          schema:
            type: string
            format: uuid`

	rules := make(map[string]*model.Rule)
	rules["owasp-1"] = rulesets.GetOwaspAPIRule1NoNumericIDs()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Results, 0)
}

func TestRuleSet_TestOWASP1Error(t *testing.T) {

	yml := `openapi: "3.1.0"
paths:
  /foo/{id}/:
    get:
      description: "get"
      parameters:
        - name: id
          in: path
          schema:
            type: integer
        - name: notanid
          in: path
          schema:
            type: integer
        - name: underscore_id
          in: path
          schema:
            type: integer
        - name: hyphen-id
          in: path
          schema:
            type: integer
            format: int32
        - name: camelId
          in: path
          schema:
            type: integer`

	rules := make(map[string]*model.Rule)
	rules["owasp-1"] = rulesets.GetOwaspAPIRule1NoNumericIDs()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Results, 5)
}

func TestRuleSet_TestOWASP2Success(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  securitySchemes:
    "anything-else":
      type: "http"
      scheme: "bearer"`

	rules := make(map[string]*model.Rule)
	rules["owasp-2"] = rulesets.GetOWASPRule2HTTPBasic()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Results, 0)
}

func TestRuleSet_TestOWASP2Error(t *testing.T) {

	yml := `openapi: "3.1.0"
info:
  version: "1.0"
components:
  securitySchemes:
    "bad negotiate":
      type: "http"
      scheme: "negotiate"
    "please-hack-me":
      type: "http"
      scheme: basic`

	rules := make(map[string]*model.Rule)
	rules["owasp-2"] = rulesets.GetOWASPRule2HTTPBasic()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet: rs,
		Spec:    []byte(yml),
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Results, 1)
}
