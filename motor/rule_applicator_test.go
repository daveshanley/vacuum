package motor

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/plugin"
	jsplugin "github.com/daveshanley/vacuum/plugin/javascript"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
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

func TestApplyRules_PostResponseSuccessWithDocument(t *testing.T) {

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

	var err error
	// create a new document.
	docConfig := datamodel.NewDocumentConfiguration()
	doc, err := libopenapi.NewDocumentWithConfiguration(burgershop, docConfig)
	assert.NoError(t, err)

	rse := &RuleSetExecution{
		RuleSet:  rs,
		Document: doc,
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
        "function": "postResponseSuccess",
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
      "given": "$.paths['/burgers'].post.requestBody.content['application/json']",
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
      "given": "$.paths['/burgers'].post.requestBody.content['application/json']",
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
	assert.Equal(t, "'length' needs 'min' or 'max' (or both) function options being set to operate: minimum property number not met (1)",
		results.Results[0].Message)

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
	assert.Equal(t, "`contact` section must contain a `url`", results.Results[0].Message)

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
	assert.Equal(t, "`info` section must contain `contact` details", results.Results[0].Message)

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
	assert.Equal(t, "`info` section must have a `description`", results.Results[0].Message)

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
	assert.Equal(t, "`info` section must contain a `license`", results.Results[0].Message)

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
	assert.Equal(t, "`license` section must contain a `url` or an `identifier`", results.Results[0].Message)
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
	assert.Equal(t, "Top level spec `tags` must not be empty, and must be an array: got string, want array",
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
	assert.Equal(t, "Top level spec `tags` must not be empty, and must be an array: got string, want object",
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
	assert.Equal(t, "tag `cinnamon` must have a description", results.Results[0].Message)
	assert.Equal(t, 5, results.Results[0].StartNode.Line)

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

func (t *testRule) GetCategory() string {
	return model.CategoryValidation
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

	panicChan := make(chan bool)

	saveMePlease := func(r any) {
		panicChan <- true
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

	//nolint:staticcheck // ignore this linting issue, it's not a bug, it's on purpose.
	<-panicChan

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
	assert.Equal(t, "component `#/components/schemas/one` does not exist in the specification",
		results.Results[0].Message)
	assert.Equal(t, "resolving-references", results.Results[0].RuleId)
}

type testRuleNotResolved struct{}

func (t *testRuleNotResolved) GetCategory() string {
	return model.CategoryValidation
}

func (r *testRuleNotResolved) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	paths := context.Index.GetAllPaths()
	oneRef := paths["/one"]["get"].Node.Content[1].Content[1].Content[1].Content[1].Content[1].Content[0].Value
	one := paths["/one"]["get"].Node.Content[1].Content[1].Content[1].Content[1].Content[1].Content[1].Value

	if oneRef != "$ref" && one != "#/components/schemas/one" {
		return []model.RuleFunctionResult{
			{
				Message: "the reference was resolved when it should not be.",
			},
		}
	} else {
		return []model.RuleFunctionResult{}
	}
}
func (r *testRuleNotResolved) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "notResolved",
	}
}

type testRuleResolved struct{}

func (r *testRuleResolved) GetCategory() string {
	return model.CategoryValidation
}

func (r *testRuleResolved) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	paths := context.Index.GetAllPaths()
	oneRef := paths["/one"]["get"].Node.Content[1].Content[1].Content[1].Content[1].Content[1].Content[0].Value
	one := paths["/one"]["get"].Node.Content[1].Content[1].Content[1].Content[1].Content[1].Content[1].Value

	if oneRef == "$ref" && one == "#/components/schemas/one" {
		return []model.RuleFunctionResult{
			{
				Message: "the reference was not resolved when it should not be.",
			},
		}
	} else {
		return []model.RuleFunctionResult{}
	}
}
func (r *testRuleResolved) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "resolved",
	}
}

func TestRuleSet_TestDocumentNotResolved(t *testing.T) {

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
      type: string`

	config := datamodel.NewDocumentConfiguration()

	doc, err := libopenapi.NewDocumentWithConfiguration([]byte(yml), config)
	if err != nil {
		panic(err)
	}

	ex := &RuleSetExecution{
		RuleSet: &rulesets.RuleSet{
			Rules: map[string]*model.Rule{
				"test": {
					Id:           "test",
					Resolved:     false,
					Given:        "$",
					RuleCategory: model.RuleCategories[model.CategoryValidation],
					Type:         rulesets.Validation,
					Severity:     model.SeverityError,
					Then: model.RuleAction{
						Function: "notResolved",
					},
				},
			},
		},
		Document: doc,
		CustomFunctions: map[string]model.RuleFunction{
			"notResolved": &testRuleNotResolved{},
		},
	}

	results := ApplyRulesToRuleSet(ex)
	assert.Len(t, results.Errors, 0)
	assert.Nil(t, results.Results)
}

func TestRuleSet_TestDocumentResolved(t *testing.T) {

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
      type: string`

	config := datamodel.NewDocumentConfiguration()

	doc, err := libopenapi.NewDocumentWithConfiguration([]byte(yml), config)
	if err != nil {
		panic(err)
	}

	ex := &RuleSetExecution{
		RuleSet: &rulesets.RuleSet{
			Rules: map[string]*model.Rule{
				"test": {
					Id:           "test",
					Resolved:     true,
					Given:        "$",
					RuleCategory: model.RuleCategories[model.CategoryValidation],
					Type:         rulesets.Validation,
					Severity:     model.SeverityError,
					Then: model.RuleAction{
						Function: "resolved",
					},
				},
			},
		},
		NodeLookupTimeout: 10 * time.Duration(time.Millisecond), // override the default 500ms.
		Document:          doc,
		CustomFunctions: map[string]model.RuleFunction{
			"resolved": &testRuleResolved{},
		},
	}

	results := ApplyRulesToRuleSet(ex)
	assert.Len(t, results.Errors, 0)
	assert.Nil(t, results.Results)
}

func TestRuleSet_TestDocument_SetNodeLookupTimeout(t *testing.T) {

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
      type: string`

	config := datamodel.NewDocumentConfiguration()

	doc, err := libopenapi.NewDocumentWithConfiguration([]byte(yml), config)
	if err != nil {
		panic(err)
	}

	ex := &RuleSetExecution{
		RuleSet: &rulesets.RuleSet{
			Rules: map[string]*model.Rule{
				"test": {
					Id:           "test",
					Resolved:     true,
					Given:        "$..paths",
					RuleCategory: model.RuleCategories[model.CategoryValidation],
					Type:         rulesets.Validation,
					Severity:     model.SeverityError,
					Then: model.RuleAction{
						Function: "resolved",
					},
				},
			},
		},
		NodeLookupTimeout: 1 * time.Duration(time.Nanosecond), // override the default 500ms to 1ns.(1ns is not enough to resolve the node)
		Document:          doc,
		CustomFunctions: map[string]model.RuleFunction{
			"resolved": &testRuleResolved{},
		},
	}

	results := ApplyRulesToRuleSet(ex)
	assert.Len(t, results.Errors, 1)
	assert.Nil(t, results.Results)
	assert.Equal(t, "node lookup timeout exceeded (1ns)", results.Errors[0].Error())
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
	assert.Equal(t, "infinite circular reference detected: one: one -> two -> one [9:15]",
		results.Results[0].Message)
	assert.Equal(t, "resolving-references", results.Results[0].RuleId)
}

func TestRuleSet_InfiniteCircularLoop_AllowArrayRecursion(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
  /one:
    get:
      responses:
        '200':
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/arr'
components:
  schemas:
    arr:
      type: array
      items:
        $ref: '#/components/schemas/obj'
    obj:
      type: object
      properties:
        self:
          $ref: '#/components/schemas/obj'`

	rules := make(map[string]*model.Rule)
	rules["openapi-tags-alphabetical"] = rulesets.GetOpenApiTagsAlphabeticalRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet:                rs,
		Spec:                   []byte(yml),
		IgnoreCircularArrayRef: true,
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)

	assert.NotNil(t, results)
	assert.Len(t, results.Results, 0)
}

func TestRuleSet_InfiniteCircularLoop_AllowPolymorphicRecursion(t *testing.T) {
	yml := `openapi: 3.1.0
paths:
  /one:
    get:
      responses:
        '200':
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/One'
components:
  schemas:
    One:
      properties:
        thing:
          oneOf:
            - "$ref": "#/components/schemas/Two"
            - "$ref": "#/components/schemas/Three"
      required:
        - thing
    Two:
      description: "test two"
      properties:
        testThing:
          "$ref": "#/components/schemas/One"
    Three:
      description: "test three"
      properties:
        testThing:
          "$ref": "#/components/schemas/One"`

	rules := make(map[string]*model.Rule)
	rules["openapi-tags-alphabetical"] = rulesets.GetOpenApiTagsAlphabeticalRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet:                      rs,
		Spec:                         []byte(yml),
		IgnoreCircularPolymorphicRef: true,
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)

	assert.NotNil(t, results)
	assert.Len(t, results.Results, 0)
}

func TestApplyRules_TestRules_Custom_Document_Pattern(t *testing.T) {

	yaml := `rules:
  my-new-rule:
    description: "Check the version is correct"
    given: $._format_version
    severity: error
    then:
      function: pattern
      functionOptions:
        match: "^1.1$"
`
	rc := CreateRuleComposer()
	rs, _ := rc.ComposeRuleSet([]byte(yaml))

	random, err := os.ReadFile("../model/test_files/non-openapi.yaml")

	assert.NoError(t, err)
	// create a new document.
	docConfig := datamodel.NewDocumentConfiguration()
	docConfig.BypassDocumentCheck = true
	doc, err := libopenapi.NewDocumentWithConfiguration([]byte(random), docConfig)
	assert.NoError(t, err)

	rse := &RuleSetExecution{
		RuleSet:           rs,
		Document:          doc,
		SkipDocumentCheck: true,
	}
	results := ApplyRulesToRuleSet(rse)

	assert.Len(t, results.Results, 1)
}

func TestApplyRules_TestRules_Custom_JS_Function_CustomDoc(t *testing.T) {

	yamlBytes := `rules:
  my-custom-js-rule:
    description: "check for a name and an id"
    given: $.custom
    severity: error
    then:
      function: checkForNameAndId
`

	defaultRuleSets := rulesets.BuildDefaultRuleSets()
	userRS, userErr := rulesets.CreateRuleSetFromData([]byte(yamlBytes))
	assert.NoError(t, userErr)

	rs := defaultRuleSets.GenerateRuleSetFromSuppliedRuleSet(userRS)

	random := `
custom:
  name: "hello"
  id: "1234"
`
	// load custom functions
	pm, err := plugin.LoadFunctions("../plugin/sample/js", false)
	assert.NoError(t, err)

	// create a new document.
	docConfig := datamodel.NewDocumentConfiguration()
	docConfig.BypassDocumentCheck = true
	doc, err := libopenapi.NewDocumentWithConfiguration([]byte(random), docConfig)
	assert.NoError(t, err)

	rse := &RuleSetExecution{
		RuleSet:           rs,
		Document:          doc,
		CustomFunctions:   pm.GetCustomFunctions(),
		Base:              "../plugin/sample/js",
		SkipDocumentCheck: true,
	}
	results := ApplyRulesToRuleSet(rse)

	assert.Len(t, results.Results, 1)
	assert.Equal(t, "name 'hello' and id '1234' are not 'some_name' or 'some_id'", results.Results[0].Message)
	assert.Equal(t, "my-custom-js-rule", results.Results[0].Rule.Id)
	assert.Equal(t, 3, results.Results[0].Range.Start.Line)
}

func TestApplyRules_TestRules_Custom_JS_Function_CustomDoc_CoreFunction(t *testing.T) {

	yamlBytes := `rules:
  my-custom-js-rule:
    description: "core me up"
    given: $
    severity: error
    then:
      function: useCoreFunction
      field: "custom"
`

	defaultRuleSets := rulesets.BuildDefaultRuleSets()
	userRS, userErr := rulesets.CreateRuleSetFromData([]byte(yamlBytes))
	assert.NoError(t, userErr)

	rs := defaultRuleSets.GenerateRuleSetFromSuppliedRuleSet(userRS)

	random := `
notCustom: true"
`
	// load custom functions
	pm, err := plugin.LoadFunctions("../plugin/sample/js", false)
	assert.NoError(t, err)

	// create a new document.
	docConfig := datamodel.NewDocumentConfiguration()
	docConfig.BypassDocumentCheck = true
	doc, err := libopenapi.NewDocumentWithConfiguration([]byte(random), docConfig)
	assert.NoError(t, err)

	rse := &RuleSetExecution{
		RuleSet:           rs,
		Document:          doc,
		CustomFunctions:   pm.GetCustomFunctions(),
		Base:              "../plugin/sample/js",
		SkipDocumentCheck: true,
	}
	results := ApplyRulesToRuleSet(rse)

	assert.Len(t, results.Results, 2)
	assert.Equal(t, "core me up: `custom` must be set", results.Results[0].Message)
	assert.Equal(t, "my-custom-js-rule", results.Results[0].Rule.Id)
	assert.Equal(t, "this is a message, added after truthy was called", results.Results[1].Message)
	assert.Equal(t, 2, results.Results[0].Range.Start.Line)
}

func TestApplyRules_TestRules_Custom_JS_Function_CustomDoc_CheckPaths(t *testing.T) {

	yamlBytes := `rules:
  my-custom-js-rule:
    description: "core me up"
    given: $.paths
    severity: error
    then:
      function: checkSinglePath
`

	defaultRuleSets := rulesets.BuildDefaultRuleSets()
	userRS, userErr := rulesets.CreateRuleSetFromData([]byte(yamlBytes))
	assert.NoError(t, userErr)

	rs := defaultRuleSets.GenerateRuleSetFromSuppliedRuleSet(userRS)

	random := `
paths:
  /one:
    get: something
  /two:
    post: something
  /three:
    patch: something
`
	// load custom functions
	pm, err := plugin.LoadFunctions("../plugin/sample/js", false)
	assert.NoError(t, err)

	// create a new document.
	docConfig := datamodel.NewDocumentConfiguration()
	docConfig.BypassDocumentCheck = true
	doc, err := libopenapi.NewDocumentWithConfiguration([]byte(random), docConfig)
	assert.NoError(t, err)

	rse := &RuleSetExecution{
		RuleSet:           rs,
		Document:          doc,
		CustomFunctions:   pm.GetCustomFunctions(),
		Base:              "../plugin/sample/js",
		SkipDocumentCheck: true,
	}
	results := ApplyRulesToRuleSet(rse)

	assert.Len(t, results.Results, 1)
	assert.Equal(t, "More than a single path exists, found 3 paths", results.Results[0].Message)
}

func TestApplyRules_TestRules_Custom_JS_Function_CustomDoc_CoreFunction_FunctionOptions(t *testing.T) {

	yamlBytes := `rules:
  my-custom-js-rule:
    description: "core me up"
    given: $
    severity: error
    then:
      function: useFunctionOptions
      field: "custom"
      functionOptions:
         someOption: "someValue"
`

	defaultRuleSets := rulesets.BuildDefaultRuleSets()
	userRS, userErr := rulesets.CreateRuleSetFromData([]byte(yamlBytes))
	assert.NoError(t, userErr)

	rs := defaultRuleSets.GenerateRuleSetFromSuppliedRuleSet(userRS)

	random := `
notCustom: true"
`
	// load custom functions
	pm, err := plugin.LoadFunctions("../plugin/sample/js", false)
	assert.NoError(t, err)

	// create a new document.
	docConfig := datamodel.NewDocumentConfiguration()
	docConfig.BypassDocumentCheck = true
	doc, err := libopenapi.NewDocumentWithConfiguration([]byte(random), docConfig)
	assert.NoError(t, err)

	rse := &RuleSetExecution{
		RuleSet:           rs,
		Document:          doc,
		CustomFunctions:   pm.GetCustomFunctions(),
		Base:              "../plugin/sample/js",
		SkipDocumentCheck: true,
	}
	results := ApplyRulesToRuleSet(rse)

	assert.Len(t, results.Results, 1)
	assert.Equal(t, "someOption is set to someValue", results.Results[0].Message)
	assert.Equal(t, "my-custom-js-rule", results.Results[0].Rule.Id)
	assert.Equal(t, 2, results.Results[0].Range.Start.Line)
}

func TestApplyRules_TestRules_Custom_Document_Truthy(t *testing.T) {

	json := `{
  "documentationUrl": "quobix.com",
  "rules": {
    "hello-test": {
      "description": "this is a test for checking basic mechanics",
      "recommended": true,
      "type": "style",
      "given": "$.pizza.pie",
      "then": {
        "function": "truthy",
		"field": "cake"
      }
    }
  }
}
`
	rc := CreateRuleComposer()
	rs, _ := rc.ComposeRuleSet([]byte(json))

	random := `lemons: nice
pizza:
  anything: wow
  pie:
   yes: true
   cake:
     - 1`

	var err error
	// create a new document.
	docConfig := datamodel.NewDocumentConfiguration()
	docConfig.BypassDocumentCheck = true
	doc, err := libopenapi.NewDocumentWithConfiguration([]byte(random), docConfig)
	assert.NoError(t, err)

	rse := &RuleSetExecution{
		RuleSet:           rs,
		Document:          doc,
		SkipDocumentCheck: true,
	}
	results := ApplyRulesToRuleSet(rse)

	assert.Len(t, results.Results, 0)
}

func TestRuleSetExecutionResultRelease_ReleasesOwnedResources(t *testing.T) {
	burgershop, err := os.ReadFile("../model/test_files/burgershop.openapi.yaml")
	assert.NoError(t, err)

	results := ApplyRulesToRuleSet(&RuleSetExecution{
		RuleSet: rulesets.BuildDefaultRuleSets().GenerateOpenAPIRecommendedRuleSet(),
		Spec:    burgershop,
	})

	assert.NotNil(t, results)
	assert.NotNil(t, results.RuleSetExecution)
	assert.NotNil(t, results.Index)
	assert.NotNil(t, results.SpecInfo)
	assert.NotNil(t, results.DocumentConfig)
	assert.NotNil(t, results.ownedDocument)
	assert.NotNil(t, results.unresolvedDoc)
	assert.True(t, results.ownsIndex)
	execution := results.RuleSetExecution
	assert.NotNil(t, execution)
	assert.NotNil(t, execution.DrDocument)

	results.Release()

	assert.Nil(t, results.RuleSetExecution)
	assert.Nil(t, results.Results)
	assert.Nil(t, results.IgnoredResults)
	assert.Nil(t, results.FixedResults)
	assert.Nil(t, results.Index)
	assert.Nil(t, results.SpecInfo)
	assert.Nil(t, results.Errors)
	assert.Nil(t, results.DocumentConfig)
	assert.Nil(t, results.ModifiedSpec)
	assert.Nil(t, results.ownedDocument)
	assert.Nil(t, results.unresolvedDoc)
	assert.False(t, results.ownsIndex)
	assert.Nil(t, execution.DrDocument)
}

func TestRuleSetExecutionResultRelease_PreservesCallerOwnedDocument(t *testing.T) {
	burgershop, err := os.ReadFile("../model/test_files/burgershop.openapi.yaml")
	assert.NoError(t, err)

	doc, err := libopenapi.NewDocumentWithConfiguration(burgershop, datamodel.NewDocumentConfiguration())
	assert.NoError(t, err)

	results := ApplyRulesToRuleSet(&RuleSetExecution{
		RuleSet:  rulesets.BuildDefaultRuleSets().GenerateOpenAPIRecommendedRuleSet(),
		Document: doc,
	})

	assert.NotNil(t, results)
	assert.NotNil(t, results.Index)
	assert.NotNil(t, results.SpecInfo)
	assert.Nil(t, results.ownedDocument)
	assert.NotNil(t, results.unresolvedDoc)
	assert.False(t, results.ownsIndex)

	results.Release()

	assert.NotNil(t, doc.GetSpecInfo())
	model, buildErr := doc.BuildV3Model()
	assert.NoError(t, buildErr)
	assert.NotNil(t, model)
	assert.NotNil(t, model.Index)
}

func TestRuleSetExecutionResultRelease_IsIdempotent(t *testing.T) {
	burgershop, err := os.ReadFile("../model/test_files/burgershop.openapi.yaml")
	assert.NoError(t, err)

	results := ApplyRulesToRuleSet(&RuleSetExecution{
		RuleSet: rulesets.BuildDefaultRuleSets().GenerateOpenAPIRecommendedRuleSet(),
		Spec:    burgershop,
	})

	assert.NotPanics(t, func() {
		results.Release()
		results.Release()
	})
	assert.Nil(t, results.Index)
	assert.Nil(t, results.ownedDocument)
	assert.Nil(t, results.unresolvedDoc)
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

func Benchmark_StripeSpecAgainstDefaultRuleSet(b *testing.B) {
	m, _ := os.ReadFile("../model/test_files/stripe.yaml")
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

func Benchmark_PetStoreSpecAgainstDefaultRuleSet(b *testing.B) {
	m, _ := os.ReadFile("../model/test_files/petstorev3.json")
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

func Test_Issue486(t *testing.T) {

	yml := `openapi: 3.1.0
paths:
  /hi:
    get:
      requestBody:
        content:
          application/json:
            schema:
              $ref: "oas3-schema.yaml"
`

	rules := make(map[string]*model.Rule)
	rules["oas3-host-not-example"] = rulesets.GetOAS3HostNotExampleRule()

	rs := &rulesets.RuleSet{
		Rules: rules,
	}

	rse := &RuleSetExecution{
		RuleSet:      rs,
		Spec:         []byte(yml),
		AllowLookup:  true,
		SpecFileName: "../model/test_files/issue486.yaml",
	}
	results := ApplyRulesToRuleSet(rse)
	assert.Len(t, results.Errors, 0)
	assert.Len(t, results.Results, 3)
	expected := map[string]bool{
		"circular reference detected from #/definitions/Schema":   true,
		"circular reference detected from #/definitions/Header":   true,
		"circular reference detected from #/definitions/Callback": true,
	}
	for _, r := range results.Results {
		assert.Equal(t, "circular-references", r.RuleId)
		assert.True(t, expected[r.Message])
		delete(expected, r.Message)
	}
	assert.Len(t, expected, 0)

}

func Test_EnsureConsistentDeterministicOutcome_Issue673(t *testing.T) {

	// a glitch in the doctor was causing mayhem. This test ensures that the glitch stays dead.
	//https://github.com/daveshanley/vacuum/issues/673

	yml := []byte(`openapi: 3.0.3

info:
  title: API title
  description: API description
  version: 0.0.1

servers:
  - url: http://some.url

tags:
  - name: tag_a

paths:
  /a:
    get:
      summary: Summary of GET /a
      description: Description of GET /a
      tags:
        - tag_a
      operationId: operationA
      responses:
        "200":
          description: Description of the 200 response to GET /a
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Schema1'
              examples:
                Example1:
                  value:
                    a: foo
                    b: bar

components:
  schemas:
    Schema1:
      type: object
      description: Description of Schema1
      properties:
        a:
          type: string
          example: fieldA
        b:
          type: string
          example: fieldB

  securitySchemes:
    Default:
      description: Description of the securityScheme.
      type: http
      scheme: bearer

security:
  - Default: []
`)

	run := func(wg *sync.WaitGroup) {
		docConf := &datamodel.DocumentConfiguration{}
		d, err := libopenapi.NewDocumentWithConfiguration(yml, docConf)
		if err != nil {
			log.Fatalf("error: %v", err)
		}

		ex := &RuleSetExecution{
			RuleSet:           rulesets.BuildDefaultRuleSets().GenerateOpenAPIDefaultRuleSet(),
			Document:          d,
			NodeLookupTimeout: 5 * time.Second,  // Increase timeout for CI/CD environments
			Timeout:           30 * time.Second, // Increase rule timeout for CI/CD environments
		}

		results := ApplyRulesToRuleSet(ex)
		assert.Len(t, results.Results, 4) // Should be exactly 4 with contextual inline rendering fix
		if wg != nil {
			wg.Done()
		}
	}

	// run it a thousand times to ensure that the results are consistent and deterministic.
	for i := 0; i < 1000; i++ {
		run(nil)
	}

	// now run it async
	var wg sync.WaitGroup
	wg.Add(1000)
	for i := 0; i < 1000; i++ {
		go run(&wg)
	}
	wg.Wait()
}

// TestIssue523_ExtensionReferenceToNonYAMLFile tests that vacuum handles references
// to non-YAML files (like .go files) in extensions when ext-refs flag is enabled
// Issue: https://github.com/daveshanley/vacuum/issues/523
func TestIssue523_ExtensionReferenceToNonYAMLFile(t *testing.T) {
	// Load the test spec
	testFile := "../model/test_files/issue523_spec.yaml"
	specBytes, err := os.ReadFile(testFile)
	assert.NoError(t, err)

	// Test with ext-refs disabled (default behavior)
	t.Run("ext-refs disabled", func(t *testing.T) {
		// Load default rulesets
		defaultRuleSets := rulesets.BuildDefaultRuleSets()

		// Use the recommended ruleset
		selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()

		// Set base path to where the files are
		execution := &RuleSetExecution{
			RuleSet:                         selectedRS,
			SpecFileName:                    testFile,
			Spec:                            specBytes,
			ExtractReferencesFromExtensions: false, // ext-refs disabled
			Base:                            filepath.Dir(testFile),
			AllowLookup:                     true,
			SilenceLogs:                     true,
		}

		results := ApplyRulesToRuleSet(execution)

		// With ext-refs disabled, references in extensions should be ignored
		// so we shouldn't get errors about references in x-codeSamples
		refErrors := 0
		for _, res := range results.Results {
			if res.Rule.Id == "resolving-references" && strings.Contains(res.Message, "issue523") {
				refErrors++
			}
		}

		// When ext-refs is disabled, references in extensions should be ignored
		assert.Equal(t, 0, refErrors, "Should not have reference errors for files in x-codeSamples when ext-refs is disabled")
	})

	// Test with ext-refs enabled
	t.Run("ext-refs enabled", func(t *testing.T) {
		// Load default rulesets
		defaultRuleSets := rulesets.BuildDefaultRuleSets()

		// Use the recommended ruleset
		selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()

		// Set base path to where the files are
		execution := &RuleSetExecution{
			RuleSet:                         selectedRS,
			SpecFileName:                    testFile,
			Spec:                            specBytes,
			ExtractReferencesFromExtensions: true, // ext-refs enabled
			Base:                            filepath.Dir(testFile),
			AllowLookup:                     true,
			SilenceLogs:                     true,
		}

		results := ApplyRulesToRuleSet(execution)

		// With ext-refs enabled and the fix applied, vacuum should now be able to
		// find and load the referenced files without errors
		refErrors := 0
		for _, res := range results.Results {
			// Look for any errors related to our test files
			if strings.Contains(res.Message, "issue523_echo.go") || strings.Contains(res.Message, "issue523_test.yaml") {
				if res.Rule.Id == "resolving-references" || res.Rule.Id == "oas3-unused-component" {
					refErrors++
				}
			}
		}

		// After the fix, the files should be found and loaded successfully
		// The rolodex can handle non-YAML files by creating a fake YAML node with the content
		assert.Equal(t, 0, refErrors, "Should not have reference errors after fix - files are found and loaded")
	})
}

// testResolvedSchemaCounter is a test rule that counts schemas in resolved document
type testResolvedSchemaCounter struct {
	schemaCount int
}

func (r *testResolvedSchemaCounter) GetCategory() string {
	return model.CategoryValidation
}

func (r *testResolvedSchemaCounter) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	// Count all schemas in the resolved document
	r.schemaCount = len(context.Index.GetAllComponentSchemas())
	return nil
}

func (r *testResolvedSchemaCounter) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "schemaCounter",
	}
}

type testResolvedExecutionRecorder struct {
	resolvedExecution bool
	documentSet       bool
	documentResolved  bool
	indexSet          bool
}

func (r *testResolvedExecutionRecorder) GetCategory() string {
	return model.CategoryValidation
}

func responseHasResolvedContent(node *yaml.Node) bool {
	if node == nil {
		return false
	}

	var decoded map[string]interface{}
	if err := node.Decode(&decoded); err != nil {
		return false
	}

	paths, ok := decoded["paths"].(map[string]interface{})
	if !ok {
		return false
	}
	testPath, ok := paths["/test"].(map[string]interface{})
	if !ok {
		return false
	}
	getOp, ok := testPath["get"].(map[string]interface{})
	if !ok {
		return false
	}
	responses, ok := getOp["responses"].(map[string]interface{})
	if !ok {
		return false
	}
	notFound, ok := responses["404"].(map[string]interface{})
	if !ok {
		return false
	}
	if _, hasRef := notFound["$ref"]; hasRef {
		return false
	}
	if _, hasContent := notFound["content"]; hasContent {
		return true
	}
	_, hasDescription := notFound["description"]
	return hasDescription
}

func (r *testResolvedExecutionRecorder) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	if len(nodes) > 0 {
		r.resolvedExecution = responseHasResolvedContent(nodes[0])
	}
	r.documentSet = context.Document != nil
	r.documentResolved = documentHasResolvedContent(context.Document)
	r.indexSet = context.Index != nil
	return nil
}

func (r *testResolvedExecutionRecorder) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "resolvedExecutionRecorder",
	}
}

func documentHasResolvedContent(document libopenapi.Document) bool {
	if document == nil || document.GetRolodex() == nil {
		return false
	}
	return responseHasResolvedContent(document.GetRolodex().GetRootNode())
}

type testNestedDocumentContextRecorder struct {
	documentContextEnabled bool
	indexContextEnabled    bool
}

func (r *testNestedDocumentContextRecorder) GetCategory() string {
	return model.CategoryValidation
}

func (r *testNestedDocumentContextRecorder) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	r.documentContextEnabled = context.Document != nil &&
		context.Document.GetConfiguration() != nil &&
		context.Document.GetConfiguration().ResolveNestedRefsWithDocumentContext
	r.indexContextEnabled = context.Index != nil &&
		context.Index.GetConfig() != nil &&
		context.Index.GetConfig().ResolveNestedRefsWithDocumentContext

	if r.documentContextEnabled && r.indexContextEnabled {
		return nil
	}

	result := model.BuildFunctionResultString("nested document context not enabled")
	result.Rule = context.Rule
	return []model.RuleFunctionResult{result}
}

func (r *testNestedDocumentContextRecorder) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "nestedDocumentContextRecorder",
	}
}

type testExecutionConfigRecorder struct {
	expectedLogger *slog.Logger

	documentLoggerMatches        bool
	indexLoggerMatches           bool
	documentBaseURL              string
	indexBaseURL                 string
	documentBasePath             string
	indexBasePath                string
	documentRemoteURLHandlerSet  bool
	documentExtractSequentially  bool
	documentSkipJSONConversion   bool
	documentAllowFileReferences  bool
	documentAllowRemoteReference bool
	indexAllowFileLookup         bool
	indexAllowRemoteLookup       bool
}

func (r *testExecutionConfigRecorder) GetCategory() string {
	return model.CategoryValidation
}

func (r *testExecutionConfigRecorder) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	if context.Document != nil && context.Document.GetConfiguration() != nil {
		cfg := context.Document.GetConfiguration()
		r.documentLoggerMatches = cfg.Logger == r.expectedLogger
		r.documentBasePath = cfg.BasePath
		r.documentRemoteURLHandlerSet = cfg.RemoteURLHandler != nil
		r.documentExtractSequentially = cfg.ExtractRefsSequentially
		r.documentSkipJSONConversion = cfg.SkipJSONConversion
		r.documentAllowFileReferences = cfg.AllowFileReferences
		r.documentAllowRemoteReference = cfg.AllowRemoteReferences
		if cfg.BaseURL != nil {
			r.documentBaseURL = cfg.BaseURL.String()
		}
	}
	if context.Index != nil && context.Index.GetConfig() != nil {
		cfg := context.Index.GetConfig()
		r.indexLoggerMatches = cfg.Logger == r.expectedLogger
		r.indexBasePath = cfg.BasePath
		r.indexAllowFileLookup = cfg.AllowFileLookup
		r.indexAllowRemoteLookup = cfg.AllowRemoteLookup
		if cfg.BaseURL != nil {
			r.indexBaseURL = cfg.BaseURL.String()
		}
	}
	return nil
}

func (r *testExecutionConfigRecorder) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "executionConfigRecorder",
	}
}

func buildExecutionConfigRuleSet(functionName string) *rulesets.RuleSet {
	return &rulesets.RuleSet{
		Rules: map[string]*model.Rule{
			"execution-config": {
				Id:           "execution-config",
				Resolved:     true,
				Given:        "$",
				RuleCategory: model.RuleCategories[model.CategoryValidation],
				Type:         rulesets.Validation,
				Severity:     model.SeverityError,
				Then: model.RuleAction{
					Function: functionName,
				},
			},
		},
	}
}

func buildResolvedInputJSFunction(t *testing.T) model.RuleFunction {
	t.Helper()

	script := `
function runRule(input) {
  if (!input.properties || !input.properties.shared) {
    return [{ message: "shared property missing" }];
  }
  if (input.properties.shared.$ref) {
    return [{ message: "shared ref not resolved" }];
  }
  if (input.properties.shared.type !== "object") {
    return [{ message: "shared schema type missing" }];
  }
  if (!input.properties.shared.properties || !input.properties.shared.properties.id || input.properties.shared.properties.id.type !== "string") {
    return [{ message: "shared schema properties missing" }];
  }
  return [];
}
`

	ruleFunc := jsplugin.NewJSRuleFunction("jsResolvedInputCheck", script)
	assert.NoError(t, ruleFunc.CheckScript())
	return ruleFunc
}

// TestIssue441_DeterministicResolvedRules verifies that resolved rules produce
// consistent results across multiple runs. This tests the fix for issue #441
// where custom rules with Resolved: true were getting non-deterministic results.
func TestIssue441_DeterministicResolvedRules(t *testing.T) {
	// Load a large spec with many references
	specPath := "../model/test_files/stripe.yaml"
	specBytes, err := os.ReadFile(specPath)
	if err != nil {
		t.Skipf("Could not load spec: %v", err)
	}

	const runs = 10
	var baselineSchemaCount int
	var baselineResultCount int

	for i := 0; i < runs; i++ {
		counter := &testResolvedSchemaCounter{}

		ex := &RuleSetExecution{
			RuleSet: &rulesets.RuleSet{
				Rules: map[string]*model.Rule{
					"schema-counter": {
						Id:           "schema-counter",
						Resolved:     true,
						Given:        "$",
						RuleCategory: model.RuleCategories[model.CategoryValidation],
						Type:         rulesets.Validation,
						Severity:     model.SeverityInfo,
						Then: model.RuleAction{
							Function: "schemaCounter",
						},
					},
				},
			},
			Spec:        specBytes,
			SilenceLogs: true,
			CustomFunctions: map[string]model.RuleFunction{
				"schemaCounter": counter,
			},
		}

		results := ApplyRulesToRuleSet(ex)

		if i == 0 {
			baselineSchemaCount = counter.schemaCount
			baselineResultCount = len(results.Results)
			t.Logf("Baseline: %d schemas, %d results", baselineSchemaCount, baselineResultCount)
		} else {
			if counter.schemaCount != baselineSchemaCount {
				t.Errorf("Run %d: different schema count: got %d, want %d", i, counter.schemaCount, baselineSchemaCount)
			}
			if len(results.Results) != baselineResultCount {
				t.Errorf("Run %d: different result count: got %d, want %d", i, len(results.Results), baselineResultCount)
			}
		}
	}

	t.Logf("All %d runs produced identical results: %d schemas", runs, baselineSchemaCount)
}

func TestRuleSet_ResolvedRuleSetsResolvedExecution(t *testing.T) {
	spec := []byte(`openapi: "3.0.2"
info:
  title: Test
  version: "1.0"
paths:
  /test:
    get:
      responses:
        '404':
          $ref: '#/components/responses/NotFound'
components:
  responses:
    NotFound:
      description: Not Found
`)

	recorder := &testResolvedExecutionRecorder{}
	results := ApplyRulesToRuleSetWithOptions(&RuleSetExecution{
		RuleSet: &rulesets.RuleSet{
			Rules: map[string]*model.Rule{
				"resolved-recorder": {
					Id:           "resolved-recorder",
					Resolved:     true,
					Given:        "$",
					RuleCategory: model.RuleCategories[model.CategoryValidation],
					Type:         rulesets.Validation,
					Severity:     model.SeverityInfo,
					Then: model.RuleAction{
						Function: "resolvedExecutionRecorder",
					},
				},
			},
		},
		Spec:        spec,
		SilenceLogs: true,
		CustomFunctions: map[string]model.RuleFunction{
			"resolvedExecutionRecorder": recorder,
		},
	}, nil)

	assert.Empty(t, results.Errors)
	assert.True(t, recorder.resolvedExecution)
	assert.True(t, recorder.documentSet)
	assert.False(t, recorder.documentResolved)
	assert.True(t, recorder.indexSet)
}

func TestRuleSet_ResolveAllRefsSetsResolvedExecution(t *testing.T) {
	spec := []byte(`openapi: "3.0.2"
info:
  title: Test
  version: "1.0"
paths:
  /test:
    get:
      responses:
        '404':
          $ref: '#/components/responses/NotFound'
components:
  responses:
    NotFound:
      description: Not Found
`)

	recorder := &testResolvedExecutionRecorder{}
	results := ApplyRulesToRuleSetWithOptions(&RuleSetExecution{
		RuleSet: &rulesets.RuleSet{
			Rules: map[string]*model.Rule{
				"resolved-recorder": {
					Id:           "resolved-recorder",
					Resolved:     false,
					Given:        "$",
					RuleCategory: model.RuleCategories[model.CategoryValidation],
					Type:         rulesets.Validation,
					Severity:     model.SeverityInfo,
					Then: model.RuleAction{
						Function: "resolvedExecutionRecorder",
					},
				},
			},
		},
		Spec:        spec,
		SilenceLogs: true,
		CustomFunctions: map[string]model.RuleFunction{
			"resolvedExecutionRecorder": recorder,
		},
	}, &ExecutionOptions{ResolveAllRefs: true})

	assert.Empty(t, results.Errors)
	assert.True(t, recorder.resolvedExecution)
	assert.True(t, recorder.documentSet)
	assert.True(t, recorder.documentResolved)
	assert.True(t, recorder.indexSet)
}

func TestRuleSet_JSResolvedExecution_UsesResolvedInput(t *testing.T) {
	spec := []byte(`openapi: "3.0.2"
info:
  title: Test
  version: "1.0"
paths: {}
components:
  schemas:
    Shared:
      type: object
      properties:
        id:
          type: string
    Wrapper:
      type: object
      properties:
        shared:
          $ref: '#/components/schemas/Shared'
`)

	ruleFunc := buildResolvedInputJSFunction(t)

	tests := []struct {
		name           string
		ruleResolved   bool
		resolveAllRefs bool
		wantViolations int
	}{
		{
			name:           "unresolved_rule",
			ruleResolved:   false,
			resolveAllRefs: false,
			wantViolations: 1,
		},
		{
			name:           "resolved_rule",
			ruleResolved:   true,
			resolveAllRefs: false,
			wantViolations: 0,
		},
		{
			name:           "global_resolve_all_refs",
			ruleResolved:   false,
			resolveAllRefs: true,
			wantViolations: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := ApplyRulesToRuleSetWithOptions(&RuleSetExecution{
				RuleSet: &rulesets.RuleSet{
					Rules: map[string]*model.Rule{
						"js-resolve-refs": {
							Id:           "js-resolved-input",
							Resolved:     tt.ruleResolved,
							Given:        "$.components.schemas.Wrapper",
							RuleCategory: model.RuleCategories[model.CategoryValidation],
							Type:         rulesets.Validation,
							Severity:     model.SeverityError,
							Then: model.RuleAction{
								Function: "jsResolvedInputCheck",
							},
						},
					},
				},
				Spec:        spec,
				SilenceLogs: true,
				CustomFunctions: map[string]model.RuleFunction{
					"jsResolvedInputCheck": ruleFunc,
				},
			}, &ExecutionOptions{ResolveAllRefs: tt.resolveAllRefs})

			assert.Empty(t, results.Errors)
			ruleResults := filterResultsByRuleId(results.Results, "js-resolved-input")
			assert.Len(t, ruleResults, tt.wantViolations)
		})
	}
}

func TestRuleSet_SchemaFunction_UsesResolvedInput(t *testing.T) {
	spec := []byte(`openapi: "3.0.2"
info:
  title: Test
  version: "1.0"
paths: {}
components:
  schemas:
    Shared:
      type: object
      properties:
        id:
          type: string
    Wrapper:
      type: object
      properties:
        shared:
          $ref: '#/components/schemas/Shared'
`)

	tests := []struct {
		name           string
		ruleResolved   bool
		resolveAllRefs bool
		wantViolations int
	}{
		{
			name:           "unresolved_rule",
			ruleResolved:   false,
			resolveAllRefs: false,
			wantViolations: 1,
		},
		{
			name:           "resolved_rule",
			ruleResolved:   true,
			resolveAllRefs: false,
			wantViolations: 0,
		},
		{
			name:           "global_resolve_all_refs",
			ruleResolved:   false,
			resolveAllRefs: true,
			wantViolations: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := ApplyRulesToRuleSetWithOptions(&RuleSetExecution{
				RuleSet: &rulesets.RuleSet{
					Rules: map[string]*model.Rule{
						"schema-resolved-input": {
							Id:           "schema-resolved-input",
							Resolved:     tt.ruleResolved,
							Given:        "$.components.schemas.Wrapper",
							RuleCategory: model.RuleCategories[model.CategoryValidation],
							Type:         rulesets.Validation,
							Severity:     model.SeverityError,
							Then: model.RuleAction{
								Field:    "properties.shared",
								Function: "schema",
								FunctionOptions: map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"required": []string{
											"type",
										},
										"properties": map[string]interface{}{
											"type": map[string]interface{}{
												"const": "object",
											},
										},
									},
								},
							},
						},
					},
				},
				Spec:        spec,
				SilenceLogs: true,
			}, &ExecutionOptions{ResolveAllRefs: tt.resolveAllRefs})

			assert.Empty(t, results.Errors)
			ruleResults := filterResultsByRuleId(results.Results, "schema-resolved-input")
			assert.Len(t, ruleResults, tt.wantViolations)
		})
	}
}

func TestRuleSet_NestedRefsDocContextSetsResolvedExecutionConfig(t *testing.T) {
	spec := []byte(`openapi: "3.0.2"
info:
  title: Test
  version: "1.0"
paths: {}
`)

	tests := []struct {
		name                                 string
		ruleResolved                         bool
		resolveAllRefs                       bool
		resolveNestedRefsWithDocumentContext bool
		wantViolations                       int
	}{
		{
			name:                                 "resolved_rule_without_nested_document_context",
			ruleResolved:                         true,
			resolveNestedRefsWithDocumentContext: false,
			wantViolations:                       1,
		},
		{
			name:                                 "resolved_rule_with_nested_document_context",
			ruleResolved:                         true,
			resolveNestedRefsWithDocumentContext: true,
			wantViolations:                       0,
		},
		{
			name:                                 "resolve_all_refs_without_nested_document_context",
			resolveAllRefs:                       true,
			resolveNestedRefsWithDocumentContext: false,
			wantViolations:                       1,
		},
		{
			name:                                 "resolve_all_refs_with_nested_document_context",
			resolveAllRefs:                       true,
			resolveNestedRefsWithDocumentContext: true,
			wantViolations:                       0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := &testNestedDocumentContextRecorder{}
			results := ApplyRulesToRuleSetWithOptions(&RuleSetExecution{
				RuleSet: &rulesets.RuleSet{
					Rules: map[string]*model.Rule{
						"nested-document-context": {
							Id:           "nested-document-context",
							Resolved:     tt.ruleResolved,
							Given:        "$",
							RuleCategory: model.RuleCategories[model.CategoryValidation],
							Type:         rulesets.Validation,
							Severity:     model.SeverityError,
							Then: model.RuleAction{
								Function: "nestedDocumentContextRecorder",
							},
						},
					},
				},
				Spec:        spec,
				SilenceLogs: true,
				CustomFunctions: map[string]model.RuleFunction{
					"nestedDocumentContextRecorder": recorder,
				},
			}, &ExecutionOptions{
				ResolveAllRefs:       tt.resolveAllRefs,
				NestedRefsDocContext: tt.resolveNestedRefsWithDocumentContext,
			})

			assert.Empty(t, results.Errors)
			ruleResults := filterResultsByRuleId(results.Results, "nested-document-context")
			assert.Len(t, ruleResults, tt.wantViolations)
			if tt.wantViolations == 0 {
				assert.True(t, recorder.documentContextEnabled)
				assert.True(t, recorder.indexContextEnabled)
			}
		})
	}
}

func TestRuleSet_ExecutionConfigPropagatesLoggerBaseURLAndTurboSettings(t *testing.T) {
	spec := []byte(`openapi: "3.0.2"
info:
  title: Test
  version: "1.0"
paths: {}
`)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	recorder := &testExecutionConfigRecorder{expectedLogger: logger}

	results := ApplyRulesToRuleSet(&RuleSetExecution{
		RuleSet:                       buildExecutionConfigRuleSet("executionConfigRecorder"),
		Spec:                          spec,
		Logger:                        logger,
		Base:                          "https://example.com/apis",
		ExtractReferencesSequentially: true,
		TurboMode:                     true,
		HTTPClientConfig: utils.HTTPClientConfig{
			Insecure: true,
		},
		CustomFunctions: map[string]model.RuleFunction{
			"executionConfigRecorder": recorder,
		},
	})

	assert.Empty(t, results.Errors)
	assert.True(t, recorder.documentLoggerMatches)
	assert.True(t, recorder.indexLoggerMatches)
	assert.Equal(t, "https://example.com/apis", recorder.documentBaseURL)
	assert.Equal(t, "https://example.com/apis", recorder.indexBaseURL)
	assert.Empty(t, recorder.documentBasePath)
	assert.Empty(t, recorder.indexBasePath)
	assert.True(t, recorder.documentRemoteURLHandlerSet)
	assert.True(t, recorder.documentExtractSequentially)
	assert.True(t, recorder.documentSkipJSONConversion)
	assert.True(t, recorder.indexAllowRemoteLookup)
	assert.False(t, recorder.indexAllowFileLookup)
}

func TestRuleSet_AllowLookupUsesCurrentDirectoryWhenSpecPathHasNoParent(t *testing.T) {
	spec := []byte(`openapi: "3.0.2"
info:
  title: Test
  version: "1.0"
paths: {}
`)

	currentDir, err := os.Getwd()
	assert.NoError(t, err)
	tempDir := t.TempDir()
	assert.NoError(t, os.Chdir(tempDir))
	defer func() {
		_ = os.Chdir(currentDir)
	}()

	recorder := &testExecutionConfigRecorder{}
	results := ApplyRulesToRuleSet(&RuleSetExecution{
		RuleSet:      buildExecutionConfigRuleSet("executionConfigRecorder"),
		Spec:         spec,
		SpecFileName: "spec.yaml",
		AllowLookup:  true,
		SilenceLogs:  true,
		CustomFunctions: map[string]model.RuleFunction{
			"executionConfigRecorder": recorder,
		},
	})

	assert.Empty(t, results.Errors)
	assert.Equal(t, ".", recorder.documentBasePath)
	assert.Equal(t, ".", recorder.indexBasePath)
	assert.True(t, recorder.documentAllowFileReferences)
	assert.True(t, recorder.documentAllowRemoteReference)
	assert.True(t, recorder.indexAllowFileLookup)
	assert.True(t, recorder.indexAllowRemoteLookup)
}

func TestRuleSet_SuppliedDocumentNestedContextIsPreservedOrRebuilt(t *testing.T) {
	spec := []byte(`openapi: "3.0.2"
info:
  title: Test
  version: "1.0"
paths: {}
`)

	tests := []struct {
		name                    string
		suppliedDocumentContext bool
		executionContext        bool
	}{
		{
			name:                    "preserve_supplied_document_context",
			suppliedDocumentContext: true,
			executionContext:        false,
		},
		{
			name:                    "rebuild_supplied_document_with_requested_context",
			suppliedDocumentContext: false,
			executionContext:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docConfig := datamodel.NewDocumentConfiguration()
			docConfig.ResolveNestedRefsWithDocumentContext = tt.suppliedDocumentContext
			doc, err := libopenapi.NewDocumentWithConfiguration(spec, docConfig)
			assert.NoError(t, err)

			recorder := &testNestedDocumentContextRecorder{}
			results := ApplyRulesToRuleSetWithOptions(&RuleSetExecution{
				RuleSet:     buildExecutionConfigRuleSet("nestedDocumentContextRecorder"),
				Document:    doc,
				SilenceLogs: true,
				CustomFunctions: map[string]model.RuleFunction{
					"nestedDocumentContextRecorder": recorder,
				},
			}, &ExecutionOptions{NestedRefsDocContext: tt.executionContext})

			assert.Empty(t, results.Errors)
			ruleResults := filterResultsByRuleId(results.Results, "execution-config")
			assert.Len(t, ruleResults, 0)
			assert.True(t, recorder.documentContextEnabled)
			assert.True(t, recorder.indexContextEnabled)
		})
	}
}

func TestRuleSet_SuppliedDocumentRebuildErrorIsReturned(t *testing.T) {
	spec := []byte(`openapi: "3.0.2"
info:
  title: Test
  version: "1.0"
paths: {}
`)

	docConfig := datamodel.NewDocumentConfiguration()
	doc, err := libopenapi.NewDocumentWithConfiguration(spec, docConfig)
	assert.NoError(t, err)

	badSpec := []byte("openapi: [")
	doc.GetSpecInfo().SpecBytes = &badSpec

	results := ApplyRulesToRuleSetWithOptions(&RuleSetExecution{
		RuleSet:     buildExecutionConfigRuleSet("executionConfigRecorder"),
		Document:    doc,
		SilenceLogs: true,
		CustomFunctions: map[string]model.RuleFunction{
			"executionConfigRecorder": &testExecutionConfigRecorder{},
		},
	}, &ExecutionOptions{NestedRefsDocContext: true})

	assert.Len(t, results.Errors, 1)
	assert.ErrorContains(t, results.Errors[0], "unable to parse specification")
}

// TestIssue804_ResolvedTrueDeepJSONPath verifies that rules with resolved: true
// evaluate JSONPath against the resolved document (with $ref values dereferenced).
// This is a regression test for issue #804 where the autofix commit hardcoded all
// JSONPath evaluation to use the unresolved document.
func TestIssue804_ResolvedTrueDeepJSONPath(t *testing.T) {
	spec := []byte(`openapi: "3.0.2"
info:
  title: Test
  version: "1.0"
paths:
  /test:
    get:
      responses:
        '404':
          $ref: '#/components/responses/NotFound'
components:
  responses:
    NotFound:
      description: Not Found
      content:
        application/json:
          schema:
            type: object
`)

	rs := &rulesets.RuleSet{
		Rules: map[string]*model.Rule{
			"resolve-refs-test": {
				Id:           "resolve-refs-test",
				Resolved:     true,
				Given:        "$.paths[*][*].responses['404']",
				Severity:     model.SeverityError,
				RuleCategory: model.RuleCategories[model.CategoryValidation],
				Type:         rulesets.Validation,
				Then: model.RuleAction{
					Field:    "content",
					Function: "defined",
				},
			},
		},
	}

	results := ApplyRulesToRuleSet(&RuleSetExecution{
		RuleSet:     rs,
		Spec:        spec,
		SilenceLogs: true,
	})

	// With resolved: true, the $ref is dereferenced and "content" exists on the resolved node.
	// There should be zero violations from our custom rule.
	ruleResults := filterResultsByRuleId(results.Results, "resolve-refs-test")
	assert.Len(t, ruleResults, 0,
		"resolved: true should find 'content' on the dereferenced response, got violations: %v", ruleResults)
}

// TestIssue804_ResolvedFalseDeepJSONPath verifies that rules with resolved: false
// evaluate JSONPath against the unresolved document (with $ref nodes intact).
func TestIssue804_ResolvedFalseDeepJSONPath(t *testing.T) {
	spec := []byte(`openapi: "3.0.2"
info:
  title: Test
  version: "1.0"
paths:
  /test:
    get:
      responses:
        '404':
          $ref: '#/components/responses/NotFound'
components:
  responses:
    NotFound:
      description: Not Found
      content:
        application/json:
          schema:
            type: object
`)

	rs := &rulesets.RuleSet{
		Rules: map[string]*model.Rule{
			"unresolved-refs-test": {
				Id:           "unresolved-refs-test",
				Resolved:     false,
				Given:        "$.paths[*][*].responses['404']",
				Severity:     model.SeverityError,
				RuleCategory: model.RuleCategories[model.CategoryValidation],
				Type:         rulesets.Validation,
				Then: model.RuleAction{
					Field:    "content",
					Function: "defined",
				},
			},
		},
	}

	results := ApplyRulesToRuleSet(&RuleSetExecution{
		RuleSet:     rs,
		Spec:        spec,
		SilenceLogs: true,
	})

	// With resolved: false, the $ref node is NOT dereferenced, so "content" is missing.
	// This should produce a violation.
	ruleResults := filterResultsByRuleId(results.Results, "unresolved-refs-test")
	assert.Greater(t, len(ruleResults), 0,
		"resolved: false should NOT find 'content' on the unresolved $ref node")
}

// TestIssue804_ResolvedTrueRootPath verifies that resolved: true works correctly
// even when the given path is "$" (root node selection).
func TestIssue804_ResolvedTrueRootPath(t *testing.T) {
	spec := []byte(`openapi: "3.0.2"
info:
  title: Test
  version: "1.0"
paths:
  /test:
    get:
      responses:
        '404':
          $ref: '#/components/responses/NotFound'
components:
  responses:
    NotFound:
      description: Not Found
      content:
        application/json:
          schema:
            type: object
`)

	rs := &rulesets.RuleSet{
		Rules: map[string]*model.Rule{
			"root-resolved-test": {
				Id:           "root-resolved-test",
				Resolved:     true,
				Given:        "$",
				Severity:     model.SeverityInfo,
				RuleCategory: model.RuleCategories[model.CategoryValidation],
				Type:         rulesets.Validation,
				Then: model.RuleAction{
					Field:    "openapi",
					Function: "defined",
				},
			},
		},
	}

	results := ApplyRulesToRuleSet(&RuleSetExecution{
		RuleSet:     rs,
		Spec:        spec,
		SilenceLogs: true,
	})

	// The root node (resolved) should have "openapi" defined.
	ruleResults := filterResultsByRuleId(results.Results, "root-resolved-test")
	assert.Len(t, ruleResults, 0,
		"resolved: true with given '$' should find 'openapi' on the root node")
}

func TestIssue849_ExtendsSeverityShorthandPreservesInheritedPathAndRuleID(t *testing.T) {
	dir := t.TempDir()

	spec := []byte(`openapi: 3.0.2
info:
  title: Extends path reporting repro
  version: 1.0.0
paths:
  /items/{itemId}:
    delete:
      responses:
        "204":
          description: Item deleted
          content:
            application/json;charset=UTF-8:
              schema:
                $ref: '#/components/schemas/DeleteResult'
components:
  schemas:
    DeleteResult:
      type: object
      properties:
        id:
          type: string
`)

	baseRulesetYAML := `rules:
  no-content-for-204:
    description: "A 204 response must not have a content body."
    severity: error
    resolved: false
    given: "$.paths[*][*].responses.['204']"
    then:
      - field: content
        function: falsy
`

	basePath := filepath.Join(dir, "ruleset-base.yaml")
	err := os.WriteFile(basePath, []byte(baseRulesetYAML), 0o600)
	assert.NoError(t, err)

	parentRulesetYAML := fmt.Sprintf(`extends:
  - "%s"

rules:
  no-content-for-204: error
`, filepath.ToSlash(basePath))

	defaultRuleSets := rulesets.BuildDefaultRuleSets()

	baseInput, err := rulesets.CreateRuleSetFromData([]byte(baseRulesetYAML))
	assert.NoError(t, err)
	parentInput, err := rulesets.CreateRuleSetFromData([]byte(parentRulesetYAML))
	assert.NoError(t, err)

	baseRS := defaultRuleSets.GenerateRuleSetFromSuppliedRuleSet(baseInput)
	parentRS := defaultRuleSets.GenerateRuleSetFromSuppliedRuleSet(parentInput)

	if assert.NotNil(t, baseRS.Rules["no-content-for-204"]) {
		assert.False(t, baseRS.Rules["no-content-for-204"].Resolved)
	}
	if assert.NotNil(t, parentRS.Rules["no-content-for-204"]) {
		assert.False(t, parentRS.Rules["no-content-for-204"].Resolved)
	}

	baseResults := ApplyRulesToRuleSet(&RuleSetExecution{
		RuleSet:     baseRS,
		Spec:        spec,
		SilenceLogs: true,
	})
	parentResults := ApplyRulesToRuleSet(&RuleSetExecution{
		RuleSet:     parentRS,
		Spec:        spec,
		SilenceLogs: true,
	})

	baseRuleResults := filterResultsByRuleId(baseResults.Results, "no-content-for-204")
	parentRuleResults := filterResultsByRuleId(parentResults.Results, "no-content-for-204")

	if assert.Len(t, baseRuleResults, 1) && assert.Len(t, parentRuleResults, 1) {
		assert.Equal(t, "$.paths['/items/{itemId}'].delete.responses['204']", baseRuleResults[0].Path)
		assert.Equal(t, baseRuleResults[0].Path, parentRuleResults[0].Path)
		assert.Equal(t, "no-content-for-204", parentRuleResults[0].RuleId)
		if assert.NotNil(t, parentRuleResults[0].Rule) {
			assert.Equal(t, "no-content-for-204", parentRuleResults[0].Rule.Id)
		}
	}
}

// TestIssue829_MigrateZallyIgnoreCircularReferences verifies that the
// migrate-zally-ignore rule runs against unresolved YAML and does not recurse
// indefinitely when the document contains indirect circular references.
func TestIssue829_MigrateZallyIgnoreCircularReferences(t *testing.T) {
	spec := []byte(`openapi: 3.1.0
info:
  title: some title
  description: my description
  version: 2.5.0
paths:
  /endpoint:
    get:
      summary: summary
      operationId: endpoint
      responses:
        '200':
          description: get all
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Object'
components:
  schemas:
    BaseObject:
      type: object
      x-zally-ignore: schema-rule
      properties:
        attr1:
          $ref: '#/components/schemas/IndirectReference'
    Object:
      type: object
      properties:
        attr1:
          $ref: '#/components/schemas/BaseObject'
    IndirectReference:
      $ref: '#/components/schemas/BaseObject'
`)

	rs := &rulesets.RuleSet{
		Rules: map[string]*model.Rule{
			rulesets.MigrateZallyIgnoreRule: rulesets.GetMigrateZallyIgnoreRule(),
		},
	}

	results := ApplyRulesToRuleSet(&RuleSetExecution{
		RuleSet:     rs,
		Spec:        spec,
		SilenceLogs: true,
	})

	assert.Len(t, results.Errors, 0)

	ruleResults := filterResultsByRuleId(results.Results, rulesets.MigrateZallyIgnoreRule)
	assert.Len(t, ruleResults, 1)
	assert.Equal(t, "$.components.schemas.BaseObject.x-zally-ignore", ruleResults[0].Path)
}

// filterResultsByRuleId returns only results matching the given rule ID.
func filterResultsByRuleId(results []model.RuleFunctionResult, ruleId string) []model.RuleFunctionResult {
	var filtered []model.RuleFunctionResult
	for _, r := range results {
		if r.RuleId == ruleId {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
