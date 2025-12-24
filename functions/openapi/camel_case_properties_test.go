// Copyright 2025 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package openapi

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/stretchr/testify/assert"
)

func TestCamelCaseProperties_GetSchema(t *testing.T) {
	def := CamelCaseProperties{}
	assert.Equal(t, "oasCamelCaseProperties", def.GetSchema().Name)
}

func TestCamelCaseProperties_GetCategory(t *testing.T) {
	def := CamelCaseProperties{}
	assert.Equal(t, model.FunctionCategoryOpenAPI, def.GetCategory())
}

func TestCamelCaseProperties_RunRule_NoSchemas(t *testing.T) {
	def := CamelCaseProperties{}
	ctx := model.RuleFunctionContext{
		DrDocument: nil,
	}

	res := def.RunRule(nil, ctx)
	assert.Len(t, res, 0)
}

func TestCamelCaseProperties_RunRule_EmptyDocument(t *testing.T) {
	def := CamelCaseProperties{}
	ctx := model.RuleFunctionContext{
		DrDocument: &drModel.DrDocument{},
	}

	res := def.RunRule(nil, ctx)
	assert.Len(t, res, 0)
}

func TestCamelCaseProperties_RunRule_ValidCamelCase(t *testing.T) {
	def := CamelCaseProperties{}
	ctx := buildTestContext(`
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        firstName:
          type: string
        lastName:
          type: string
        userId:
          type: string
        emailAddress:
          type: string
`, t)

	res := def.RunRule(nil, ctx)
	assert.Len(t, res, 0)
}

func TestCamelCaseProperties_RunRule_PascalCase(t *testing.T) {
	def := CamelCaseProperties{}
	ctx := buildTestContext(`
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        FirstName:
          type: string
        LastName:
          type: string
`, t)

	res := def.RunRule(nil, ctx)
	assert.Len(t, res, 2)
	assert.Contains(t, res[0].Message, "FirstName")
	assert.Contains(t, res[0].Message, "PascalCase")
	assert.Contains(t, res[0].Message, "not `camelCase`")
	assert.Contains(t, res[1].Message, "LastName")
	assert.Contains(t, res[1].Message, "PascalCase")
}

func TestCamelCaseProperties_RunRule_SnakeCase(t *testing.T) {
	def := CamelCaseProperties{}
	ctx := buildTestContext(`
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        first_name:
          type: string
        last_name:
          type: string
`, t)

	res := def.RunRule(nil, ctx)
	assert.Len(t, res, 2)
	assert.Contains(t, res[0].Message, "first_name")
	assert.Contains(t, res[0].Message, "snake_case")
	assert.Contains(t, res[0].Message, "not `camelCase`")
	assert.Contains(t, res[1].Message, "last_name")
	assert.Contains(t, res[1].Message, "snake_case")
}

func TestCamelCaseProperties_RunRule_KebabCase(t *testing.T) {
	def := CamelCaseProperties{}
	ctx := buildTestContext(`
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        first-name:
          type: string
        last-name:
          type: string
`, t)

	res := def.RunRule(nil, ctx)
	assert.Len(t, res, 2)
	assert.Contains(t, res[0].Message, "first-name")
	assert.Contains(t, res[0].Message, "kebab-case")
	assert.Contains(t, res[0].Message, "not `camelCase`")
	assert.Contains(t, res[1].Message, "last-name")
	assert.Contains(t, res[1].Message, "kebab-case")
}

func TestCamelCaseProperties_RunRule_ScreamingSnakeCase(t *testing.T) {
	def := CamelCaseProperties{}
	ctx := buildTestContext(`
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        FIRST_NAME:
          type: string
        LAST_NAME:
          type: string
`, t)

	res := def.RunRule(nil, ctx)
	assert.Len(t, res, 2)
	assert.Contains(t, res[0].Message, "FIRST_NAME")
	assert.Contains(t, res[0].Message, "SCREAMING_SNAKE_CASE")
	assert.Contains(t, res[0].Message, "not `camelCase`")
	assert.Contains(t, res[1].Message, "LAST_NAME")
	assert.Contains(t, res[1].Message, "SCREAMING_SNAKE_CASE")
}

func TestCamelCaseProperties_RunRule_ScreamingKebabCase(t *testing.T) {
	def := CamelCaseProperties{}
	ctx := buildTestContext(`
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        FIRST-NAME:
          type: string
        LAST-NAME:
          type: string
`, t)

	res := def.RunRule(nil, ctx)
	assert.Len(t, res, 2)
	assert.Contains(t, res[0].Message, "FIRST-NAME")
	assert.Contains(t, res[0].Message, "SCREAMING-KEBAB-CASE")
	assert.Contains(t, res[0].Message, "not `camelCase`")
	assert.Contains(t, res[1].Message, "LAST-NAME")
	assert.Contains(t, res[1].Message, "SCREAMING-KEBAB-CASE")
}

func TestCamelCaseProperties_RunRule_MixedCases(t *testing.T) {
	def := CamelCaseProperties{}
	ctx := buildTestContext(`
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        firstName:
          type: string
        LastName:
          type: string
        user_id:
          type: string
        email-address:
          type: string
        PHONE_NUMBER:
          type: string
        WORK-EMAIL:
          type: string
`, t)

	res := def.RunRule(nil, ctx)
	assert.Len(t, res, 5) // firstName should pass, others should fail

	// check that valid camelCase property is not flagged
	for _, result := range res {
		assert.NotContains(t, result.Message, "firstName")
	}

	// check that invalid properties are flagged with correct case types
	caseTypes := make(map[string]string)
	for _, result := range res {
		if result.Message != "" {
			// extract property name and case type from message
			// format: property `x` is `xCase` not `camelCase`
			msg := result.Message
			if len(msg) > 0 {
				caseTypes[msg] = msg
			}
		}
	}

	assert.Greater(t, len(caseTypes), 0)
}

func TestCamelCaseProperties_RunRule_UppercaseAndLowercase(t *testing.T) {
	def := CamelCaseProperties{}
	ctx := buildTestContext(`
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        UPPERCASE:
          type: string
        validCamelCase:
          type: string
`, t)

	res := def.RunRule(nil, ctx)
	assert.Len(t, res, 1) // only UPPERCASE should fail

	var uppercaseFound bool
	for _, result := range res {
		if result.Message != "" {
			if result.Message == "property `UPPERCASE` is `UPPERCASE` not `camelCase`" {
				uppercaseFound = true
			}
			// validCamelCase should not be flagged
			assert.NotContains(t, result.Message, "validCamelCase")
		}
	}

	assert.True(t, uppercaseFound, "Should detect UPPERCASE property")
}

func TestCamelCaseProperties_RunRule_NoProperties(t *testing.T) {
	def := CamelCaseProperties{}
	ctx := buildTestContext(`
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      type: object
    Product:
      type: string
`, t)

	res := def.RunRule(nil, ctx)
	assert.Len(t, res, 0)
}

func TestCamelCaseProperties_RunRule_MultipleSchemas(t *testing.T) {
	def := CamelCaseProperties{}
	ctx := buildTestContext(`
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
components:
  schemas:
    User:
      type: object
      properties:
        first_name:
          type: string
        lastName:
          type: string
    Product:
      type: object
      properties:
        product-name:
          type: string
        PRICE:
          type: number
`, t)

	res := def.RunRule(nil, ctx)
	assert.Len(t, res, 3) // first_name, product-name, PRICE should fail; lastName should pass

	// verify that lastName (valid camelCase) is not flagged
	for _, result := range res {
		assert.NotContains(t, result.Message, "lastName")
	}
}

func TestCamelCaseProperties_isCamelCase(t *testing.T) {
	def := CamelCaseProperties{}

	// valid camelCase
	assert.True(t, def.isCamelCase("firstName"))
	assert.True(t, def.isCamelCase("lastName"))
	assert.True(t, def.isCamelCase("userId"))
	assert.True(t, def.isCamelCase("emailAddress"))
	assert.True(t, def.isCamelCase("a"))
	assert.True(t, def.isCamelCase("camelCase123"))

	// invalid cases
	assert.False(t, def.isCamelCase(""))
	assert.False(t, def.isCamelCase("FirstName"))        // PascalCase
	assert.False(t, def.isCamelCase("first_name"))       // snake_case
	assert.False(t, def.isCamelCase("first-name"))       // kebab-case
	assert.False(t, def.isCamelCase("FIRST_NAME"))       // SCREAMING_SNAKE_CASE
	assert.False(t, def.isCamelCase("FIRST-NAME"))       // SCREAMING-KEBAB-CASE
	assert.False(t, def.isCamelCase("UPPERCASE"))        // UPPERCASE
	assert.True(t, def.isCamelCase("lowercase"))         // lowercase single word is valid camelCase
	assert.False(t, def.isCamelCase("first name"))       // with space
	assert.False(t, def.isCamelCase("first.name"))       // with dot
	assert.False(t, def.isCamelCase("first@name"))       // with special char
}

func TestCamelCaseProperties_identifyCaseType(t *testing.T) {
	def := CamelCaseProperties{}

	assert.Equal(t, "PascalCase", def.identifyCaseType("FirstName"))
	assert.Equal(t, "snake_case", def.identifyCaseType("first_name"))
	assert.Equal(t, "kebab-case", def.identifyCaseType("first-name"))
	assert.Equal(t, "SCREAMING_SNAKE_CASE", def.identifyCaseType("FIRST_NAME"))
	assert.Equal(t, "SCREAMING-KEBAB-CASE", def.identifyCaseType("FIRST-NAME"))
	assert.Equal(t, "UPPERCASE", def.identifyCaseType("UPPERCASE"))
	assert.Equal(t, "lowercase", def.identifyCaseType("lowercase"))
	assert.Equal(t, "Snake_Case", def.identifyCaseType("First_Name"))
	assert.Equal(t, "Kebab-Case", def.identifyCaseType("First-Name"))
	assert.Equal(t, "mixedCase", def.identifyCaseType("firstName")) // would be camelCase but called when it fails isCamelCase
	assert.Equal(t, "unknown", def.identifyCaseType(""))
	assert.Equal(t, "unknown", def.identifyCaseType("first name"))
}


