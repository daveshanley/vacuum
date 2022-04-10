// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package rulesets

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRuleSet_ContactProperties(t *testing.T) {

	yml := `info:
  contact:
    name: pizza
    email: monkey`

	rules := make(map[string]*model.Rule)
	rules["contact-properties"] = GetContactPropertiesRule()

	rs := &model.RuleSet{
		Rules: rules,
	}

	results, _ := motor.ApplyRules(rs, []byte(yml))
	assert.NotNil(t, results)
	assert.Equal(t, "Contact details are incomplete: 'url' must be set", results[0].Message)

}

func TestRuleSet_InfoContact(t *testing.T) {

	yml := `info:
  title: Terrible API Spec
  description: No operations, no contact, useless.`

	rules := make(map[string]*model.Rule)
	rules["info-contact"] = GetInfoContactRule()

	rs := &model.RuleSet{
		Rules: rules,
	}

	results, _ := motor.ApplyRules(rs, []byte(yml))
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
	rules["info-description"] = GetInfoDescriptionRule()

	rs := &model.RuleSet{
		Rules: rules,
	}

	results, _ := motor.ApplyRules(rs, []byte(yml))
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
	rules["info-license"] = GetInfoLicenseRule()

	rs := &model.RuleSet{
		Rules: rules,
	}

	results, _ := motor.ApplyRules(rs, []byte(yml))
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
	rules["license-url"] = GetInfoLicenseUrlRule()

	rs := &model.RuleSet{
		Rules: rules,
	}

	results, _ := motor.ApplyRules(rs, []byte(yml))
	assert.NotNil(t, results)
	assert.Equal(t, "License should contain an url: 'url' must be set", results[0].Message)

}

func TestRuleSet_NoEvalInMarkdown(t *testing.T) {

	yml := `info:
  description: this has no eval('alert(1234') impact in vacuum, but JS tools might suffer.`

	rules := make(map[string]*model.Rule)
	rules["no-eval-in-markdown"] = GetNoEvalInMarkdownRule()

	rs := &model.RuleSet{
		Rules: rules,
	}

	results, _ := motor.ApplyRules(rs, []byte(yml))
	assert.NotNil(t, results)
	assert.Equal(t, "description contains content with 'eval\\(', forbidden", results[0].Message)

}

func TestRuleSet_NoScriptInMarkdown(t *testing.T) {

	yml := `info:
  description: this has no impact in vacuum, <script>alert('XSS for you')</script>`

	rules := make(map[string]*model.Rule)
	rules["no-script-tags-in-markdown"] = GetNoScriptTagsInMarkdownRule()

	rs := &model.RuleSet{
		Rules: rules,
	}

	results, _ := motor.ApplyRules(rs, []byte(yml))
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
	rules["openapi-tags-alphabetical"] = GetOpenApiTagsAlphabeticalRule()

	rs := &model.RuleSet{
		Rules: rules,
	}

	results, _ := motor.ApplyRules(rs, []byte(yml))
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
	rules["openapi-tags"] = GetOpenApiTagsRule()

	rs := &model.RuleSet{
		Rules: rules,
	}

	results, _ := motor.ApplyRules(rs, []byte(yml))
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
	rules["openapi-tags"] = GetOpenApiTagsRule()

	rs := &model.RuleSet{
		Rules: rules,
	}

	results, _ := motor.ApplyRules(rs, []byte(yml))
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
	rules["openapi-tags"] = GetOpenApiTagsRule()

	rs := &model.RuleSet{
		Rules: rules,
	}

	results, _ := motor.ApplyRules(rs, []byte(yml))
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
	rules["operation-operationId-valid-in-url"] = GetOperationIdValidInUrlRule()

	rs := &model.RuleSet{
		Rules: rules,
	}

	results, _ := motor.ApplyRules(rs, []byte(yml))
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
	rules["operation-tags"] = GetOperationTagsRule()

	rs := &model.RuleSet{
		Rules: rules,
	}

	results, _ := motor.ApplyRules(rs, []byte(yml))
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
	rules["operation-tags"] = GetOperationTagsRule()

	rs := &model.RuleSet{
		Rules: rules,
	}

	results, _ := motor.ApplyRules(rs, []byte(yml))
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
	rules["path-declarations-must-exist"] = GetPathDeclarationsMustExistRule()

	rs := &model.RuleSet{
		Rules: rules,
	}

	results, _ := motor.ApplyRules(rs, []byte(yml))
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
	rules["path-keys-no-trailing-slash"] = GetPathNoTrailingSlashRule()

	rs := &model.RuleSet{
		Rules: rules,
	}

	results, _ := motor.ApplyRules(rs, []byte(yml))
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
	rules["path-not-include-query"] = GetPathNotIncludeQueryRule()

	rs := &model.RuleSet{
		Rules: rules,
	}

	results, _ := motor.ApplyRules(rs, []byte(yml))
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
	rules["tag-description"] = GetTagDescriptionRequiredRule()

	rs := &model.RuleSet{
		Rules: rules,
	}

	results, _ := motor.ApplyRules(rs, []byte(yml))
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
	rules["oas2-api-host"] = GetOAS2APIHostRule()

	rs := &model.RuleSet{
		Rules: rules,
	}

	results, _ := motor.ApplyRules(rs, []byte(yml))
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
	rules["oas2-api-host"] = GetOAS2APIHostRule()

	rs := &model.RuleSet{
		Rules: rules,
	}

	results, _ := motor.ApplyRules(rs, []byte(yml))
	assert.Nil(t, results)
	assert.Len(t, results, 0)

}
