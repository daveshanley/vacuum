package rulesets

import (
	"regexp"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/parser"
)

// rules taken from https://github.com/stoplightio/spectral-owasp-ruleset/blob/main/src/ruleset.ts

func GetOWASPNoNumericIDsRule() *model.Rule {

	// create a schema to match against.
	opts := make(map[string]interface{})
	yml := `type: object
not:
  properties:
    type:
      pattern: integer
properties:
  format:
    enum:
      - uuid`

	jsonSchema, _ := parser.ConvertYAMLIntoJSONSchema(yml, nil)
	opts["schema"] = jsonSchema
	opts["forceValidation"] = true // this will be picked up by the schema function to force validation.

	return &model.Rule{
		Name:         "Use random IDs that cannot be guessed.",
		Id:           OwaspNoNumericIDs,
		Formats:      model.AllFormats,
		Description:  "OWASP API1:2019 - Use random IDs that cannot be guessed. UUIDs are preferred",
		Given:        `$.paths..parameters[*][?(@.name == "id" || @.name =~ /(_id|Id|-id)$/)))]`,
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Field:           "schema",
			Function:        "schema",
			FunctionOptions: opts,
		},
		HowToFix: owaspNoNumericIDsFix,
	}
}

func GetOWASPNoHttpBasicRule() *model.Rule {

	// create a schema to match against.
	opts := make(map[string]interface{})
	opts["notMatch"] = "basic"
	comp, _ := regexp.Compile(opts["notMatch"].(string))

	return &model.Rule{
		Name:         "Security scheme uses HTTP Basic.",
		Id:           OwaspNoHttpBasic,
		Formats:      model.AllFormats,
		Description:  "Security scheme uses HTTP Basic. Use a more secure authentication method, like OAuth 2.0",
		Given:        `$.components.securitySchemes[*]`,
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Field:           "scheme",
			Function:        "pattern",
			FunctionOptions: opts,
		},
		PrecompiledPattern: comp,
		HowToFix:           owaspNoHttpBasicFix,
	}
}

func GetOWASPNoAPIKeysInURLRule() *model.Rule {

	// create a schema to match against.
	opts := make(map[string]interface{})
	opts["notMatch"] = "^(path|query)$"
	comp, _ := regexp.Compile(opts["notMatch"].(string))

	return &model.Rule{
		Name:         "API Key detected in URL",
		Id:           OwaspNoAPIKeysInURL,
		Formats:      model.OAS3AllFormat,
		Description:  "API Key has been detected in a URL",
		Given:        `$..securitySchemes[*][?(@.type=="apiKey")].in`,
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function:        "pattern",
			FunctionOptions: opts,
		},
		PrecompiledPattern: comp,
		HowToFix:           owaspNoAPIKeysInURLFix,
	}
}

func GetOWASPNoCredentialsInURLRule() *model.Rule {

	// create a schema to match against.
	opts := make(map[string]interface{})
	opts["notMatch"] = `(?i)^.*(client_?secret|token|access_?token|refresh_?token|id_?token|password|secret|api-?key).*$`
	comp, _ := regexp.Compile(opts["notMatch"].(string))

	return &model.Rule{
		Name:         "Security credentials detected in path parameter",
		Id:           OwaspNoCredentialsInURL,
		Formats:      model.OAS3AllFormat,
		Description:  "URL parameters must not contain credentials such as API key, password, or secret.",
		Given:        `$..parameters[*][?(@.in =~ /(query|path)/)].name`,
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function:        "pattern",
			FunctionOptions: opts,
		},
		PrecompiledPattern: comp,
		HowToFix:           owaspNoCredentialsInURLFix,
	}
}

func GetOWASPAuthInsecureSchemesRule() *model.Rule {

	// create a schema to match against.
	opts := make(map[string]interface{})
	opts["notMatch"] = `^(negotiate|oauth)$`
	comp, _ := regexp.Compile(opts["notMatch"].(string))

	return &model.Rule{
		Name:         "Authentication scheme is considered outdated or insecure",
		Id:           OwaspAuthInsecureSchemes,
		Formats:      model.OAS3AllFormat,
		Description:  "Authentication scheme is considered outdated or insecure",
		Given:        `$..securitySchemes[*][?(@.type=="http")].scheme`,
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function:        "pattern",
			FunctionOptions: opts,
		},
		PrecompiledPattern: comp,
		HowToFix:           owaspAuthInsecureSchemesFix,
	}
}

func GetOWASPJWTBestPracticesRule() *model.Rule {

	// create a schema to match against.
	opts := make(map[string]interface{})
	opts["match"] = `.*RFC8725.*`
	comp, _ := regexp.Compile(opts["match"].(string))

	return &model.Rule{
		Name:        "JWTs must explicitly declare support for `RFC8725`",
		Id:          OwaspJWTBestPractices,
		Description: "JWTs must explicitly declare support for RFC8725 in the description",
		Given: []string{
			`$..securitySchemes[*][?(@.type=="oauth2")]`,
			`$..securitySchemes[*][?(@.bearerFormat=="jwt" || @.bearerFormat=="JWT")]`,
		},
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Field:           "description",
			Function:        "pattern",
			FunctionOptions: opts,
		},
		PrecompiledPattern: comp,
		HowToFix:           owaspJWTBestPracticesFix,
	}
}

// https://github.com/italia/api-oas-checker/blob/master/security/security.yml
func GetOWASPProtectionGlobalUnsafeRule() *model.Rule {
	return &model.Rule{
		Name:         "Operation is not protected by any security scheme",
		Id:           OwaspProtectionGlobalUnsafe,
		Description:  "API should be protected by a `security` rule either at global or operation level.",
		Given:        `$`,
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "owaspCheckSecurity",
			FunctionOptions: map[string]interface{}{
				"schemesPath": []string{"securitySchemes"},
				"nullable":    true,
				"methods":     []string{"post", "put", "patch", "delete"},
			},
		},
		HowToFix: owaspProtectionFix,
	}
}

// https://github.com/italia/api-oas-checker/blob/master/security/security.yml
func GetOWASPProtectionGlobalUnsafeStrictRule() *model.Rule {
	return &model.Rule{
		Name:         "Operation is not protected by any security scheme",
		Id:           OwaspProtectionGlobalUnsafeStrict,
		Description:  "Check if the operation is protected at operation level. Otherwise, check the global `security` property",
		Given:        `$`,
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityInfo,
		Then: model.RuleAction{
			Function: "owaspCheckSecurity",
			FunctionOptions: map[string]interface{}{
				"schemesPath": []string{"securitySchemes"},
				"nullable":    false,
				"methods":     []string{"post", "put", "patch", "delete"},
			},
		},
		HowToFix: owaspProtectionFix,
	}
}

// https://github.com/italia/api-oas-checker/blob/master/security/security.yml
func GetOWASPProtectionGlobalSafeRule() *model.Rule {
	return &model.Rule{
		Name:         "Operation is not protected by any security scheme",
		Id:           OwaspProtectionGlobalSafe,
		Description:  "Check if the operation is protected at operation level. Otherwise, check the global `security` property",
		Given:        `$`,
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityInfo,
		Then: model.RuleAction{
			Function: "owaspCheckSecurity",
			FunctionOptions: map[string]interface{}{
				"schemesPath": []string{"securitySchemes"},
				"nullable":    true,
				"methods":     []string{"get", "head"},
			},
		},
		HowToFix: owaspProtectionFix,
	}
}

func GetOWASPDefineErrorValidationRule() *model.Rule {
	return &model.Rule{
		Name:         "Missing error response of either `400`, `422` or `4XX`",
		Id:           OwaspDefineErrorValidation,
		Description:  "Missing error response of either `400`, `422` or `4XX`, Ensure all errors are documented.",
		Given:        `$.paths..responses`,
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Function: "owaspDefineErrorDefinition",
		},
		HowToFix: owaspDefineErrorValidationFix,
	}
}

func GetOWASPDefineErrorResponses401Rule() *model.Rule {

	// create a schema to match against.
	opts := make(map[string]interface{})
	yml := `type: object
required:
  - content`

	jsonSchema, _ := parser.ConvertYAMLIntoJSONSchema(yml, nil)
	opts["schema"] = jsonSchema
	opts["forceValidation"] = true // this will be picked up by the schema function to force validation.

	return &model.Rule{
		Name:         "Operation is missing a `401` error response",
		Id:           OwaspDefineErrorResponses401,
		Description:  "OWASP API Security recommends defining schemas for all responses, even error: 401",
		Given:        `$.paths..responses`,
		Resolved:     true,
		Formats:      model.AllFormats,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityWarn,
		Then: []model.RuleAction{
			{
				Field:    "401",
				Function: "defined",
			},
			{
				Field:           "401",
				Function:        "schema",
				FunctionOptions: opts,
			},
		},
		HowToFix: owaspDefineErrorResponses401Fix,
	}
}

func GetOWASPDefineErrorResponses500Rule() *model.Rule {

	opts := make(map[string]interface{})
	yml := `type: object
required:
  - content`

	jsonSchema, _ := parser.ConvertYAMLIntoJSONSchema(yml, nil)
	opts["schema"] = jsonSchema
	opts["forceValidation"] = true // this will be picked up by the schema function to force validation.

	return &model.Rule{
		Name:         "Operation is missing a `500` error response",
		Id:           OwaspDefineErrorResponses500,
		Description:  "OWASP API Security recommends defining schemas for all responses, even error: 500",
		Given:        `$.paths..responses`,
		Resolved:     true,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityWarn,
		Then: []model.RuleAction{
			{
				Field:    "500",
				Function: "defined",
			},
			{
				Field:           "500",
				Function:        "schema",
				FunctionOptions: opts,
			},
		},
		HowToFix: owaspDefineErrorResponses500Fix,
	}
}

func GetOWASPRateLimitRule() *model.Rule {
	var (
		xRatelimitLimit = "X-RateLimit-Limit"
		xRateLimitLimit = "X-Rate-Limit-Limit"
		ratelimitLimit  = "RateLimit-Limit"
		ratelimitReset  = "RateLimit-Reset"
	)

	return &model.Rule{
		Name:         "`2XX` and `4XX` responses should define rate limiting headers",
		Id:           OwaspRateLimit,
		Description:  "Define proper rate limiting to avoid attackers overloading the API.",
		Given:        `$.paths..responses`,
		Resolved:     true,
		Formats:      model.OAS3AllFormat,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "owaspHeaderDefinition",
			FunctionOptions: map[string]interface{}{
				"headers": [][]string{
					{xRatelimitLimit},
					{xRateLimitLimit},
					{ratelimitLimit, ratelimitReset},
				},
			},
		},
		HowToFix: owaspRateLimitFix,
	}
}

func GetOWASPRateLimitRetryAfterRule() *model.Rule {

	return &model.Rule{
		Name:         "A `429` response should define a `Retry-After` header",
		Id:           OwaspRateLimitRetryAfter,
		Description:  "Ensure that any `429` response, contains a `Retry-After` header.  ",
		Given:        `$..responses.429.headers`,
		Resolved:     true,
		Formats:      model.OAS3AllFormat,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Field:    "Retry-After",
			Function: "defined",
		},
		HowToFix: owaspRateLimitRetryAfterFix,
	}
}

func GetOWASPDefineErrorResponses429Rule() *model.Rule {

	opts := make(map[string]interface{})
	yml := `type: object
required:
  - content`

	jsonSchema, _ := parser.ConvertYAMLIntoJSONSchema(yml, nil)
	opts["schema"] = jsonSchema
	opts["forceValidation"] = true // this will be picked up by the schema function to force validation.

	return &model.Rule{
		Name:         "Operation is missing a `429` rate limiting error response",
		Id:           OwaspDefineErrorResponses429,
		Description:  "OWASP API Security recommends defining schemas for all responses, even error: 429",
		Given:        `$.paths..responses`,
		Resolved:     true,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityWarn,
		Then: []model.RuleAction{
			{
				Field:    "429",
				Function: "defined",
			},
			{
				Field:           "429",
				Function:        "schema",
				FunctionOptions: opts,
			},
		},
		HowToFix: owaspDefineErrorResponses429Fix,
	}
}

// It will return duplicate errors for each branch of any if/else/then logic
func GetOWASPArrayLimitRule() *model.Rule {

	// create a schema to match against.
	opts := make(map[string]interface{})
	yml := `type: object
if:
  properties:
    type:
      enum: 
        - array
then:
  oneOf:
  - required:
    - maxItems
`

	jsonSchema, _ := parser.ConvertYAMLIntoJSONSchema(yml, nil)
	opts["schema"] = jsonSchema
	opts["forceValidationOnCurrentNode"] = true // use the current node to validate (field not needed)

	return &model.Rule{
		Name:        "Schema of type array must specify maxItems",
		Id:          OwaspArrayLimit,
		Description: "Array size should be limited to mitigate resource exhaustion attacks.",
		Given: []string{
			`$..[?(@.type)]`,
		},
		Resolved:     false,
		Formats:      model.AllFormats,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function:        "schema",
			FunctionOptions: opts,
		},
		HowToFix: owaspArrayLimitFix,
	}
}

// It will return duplicate errors for each branch of any if/else/then logic
func GetOWASPStringLimitRule() *model.Rule {

	// create a schema to match against.
	opts := make(map[string]interface{})
	yml := `type: object
if:
  properties:
    type:
      enum: 
        - string
then:
  oneOf:
  - required:
    - maxLength
  - required:
    - enum
  - required:
    - const
else:
  if:
    properties:
      type:
        type: array
  then:
    if:
      properties:
        type:
          contains:
            enum:
              - string
    then:
      oneOf:
      - required:
        - maxLength
      - required:
        - enum
      - required:
        - const
`

	jsonSchema, _ := parser.ConvertYAMLIntoJSONSchema(yml, nil)
	opts["schema"] = jsonSchema
	opts["forceValidationOnCurrentNode"] = true // use the current node to validate (field not needed)

	return &model.Rule{
		Name:        "Schema of type string must specify maxLength, enum, or const",
		Id:          OwaspStringLimit,
		Description: "String size should be limited to mitigate resource exhaustion attacks.",
		Given: []string{
			`$..[?(@.type)]`,
		},
		Resolved:     false,
		Formats:      model.AllFormats,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function:        "schema",
			FunctionOptions: opts,
		},
		HowToFix: owaspStringLimitFix,
	}
}

// It will return duplicate errors for each branch of any if/else/then logic
func GetOWASPStringRestrictedRule() *model.Rule {

	// create a schema to match against.
	opts := make(map[string]interface{})
	yml := `type: object
if:
  properties:
    type:
      enum: 
        - string
then:
  anyOf:
  - required:
    - format
  - required:
    - pattern
  - required:
    - enum
  - required:
    - const
else:
  if:
    properties:
      type:
        type: array
  then:
    if:
      properties:
        type:
          contains:
            enum:
              - string
    then:
      anyOf:
      - required:
        - format
      - required:
        - pattern
      - required:
        - enum
      - required:
        - const
`

	jsonSchema, _ := parser.ConvertYAMLIntoJSONSchema(yml, nil)
	opts["schema"] = jsonSchema
	opts["forceValidationOnCurrentNode"] = true // use the current node to validate (field not needed)

	return &model.Rule{
		Name:        "Schema of type string must specify a `format`, `pattern`, `enum`, or `const`",
		Id:          OwaspStringRestricted,
		Description: "String must specify a `format`, RegEx `pattern`, `enum`, or `const`",
		Given: []string{
			`$..[?(@.type)]`,
		},
		Resolved:     false,
		Formats:      model.AllFormats,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function:        "schema",
			FunctionOptions: opts,
		},
		HowToFix: owaspStringRestrictedFix,
	}
}

// It will return duplicate errors for each branch of any if/else/then logic
func GetOWASPIntegerLimitRule() *model.Rule {

	// create a schema to match against.
	opts := make(map[string]interface{})
	yml := `type: object
if:
  properties:
    type:
      enum: 
        - integer
then:
  not:
    oneOf:
      - required:
        - exclusiveMinimum
        - minimum
      - required:
        - exclusiveMaximum
        - maximum
  oneOf:
    - required:
      - minimum
      - maximum
    - required:
      - minimum
      - exclusiveMaximum
    - required:
      - exclusiveMinimum
      - maximum
    - required:
      - exclusiveMinimum
      - exclusiveMaximum
else:
  if:
    properties:
      type:
        type: array
  then:
    if:
      properties:
        type:
          contains:
            enum:
              - integer
    then:
      not:
        oneOf:
          - required:
            - exclusiveMinimum
            - minimum
          - required:
            - exclusiveMaximum
            - maximum
      oneOf:
        - required:
          - minimum
          - maximum
        - required:
          - minimum
          - exclusiveMaximum
        - required:
          - exclusiveMinimum
          - maximum
        - required:
          - exclusiveMinimum
          - exclusiveMaximum
`

	jsonSchema, _ := parser.ConvertYAMLIntoJSONSchema(yml, nil)
	opts["schema"] = jsonSchema
	opts["forceValidationOnCurrentNode"] = true // use the current node to validate (field not needed)

	return &model.Rule{
		Name:        "Schema of type integer must specify `minimum` and `maximum` or `exclusiveMinimum` and `exclusiveMaximum`",
		Id:          OwaspIntegerLimit,
		Description: "Integers should be limited via min/max values to mitigate resource exhaustion attacks.",
		Given: []string{
			`$..[?(@.type)]`,
		},
		Resolved:     false,
		Formats:      model.AllFormats,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function:        "schema",
			FunctionOptions: opts,
		},
		HowToFix: owaspIntegerLimitFix,
	}
}

// It will return duplicate errors for each branch of any if/else/then logic
func GetOWASPIntegerLimitLegacyRule() *model.Rule {

	// create a schema to match against.
	opts := make(map[string]interface{})
	yml := `type: object
if:
  properties:
    type:
      enum: 
        - integer
then:
  allOf:
    - required:
      - minimum
    - required:
      - maximum
else:
  if:
    properties:
      type:
        type: array
  then:
    if:
      properties:
        type:
          contains:
            enum:
              - integer
    then:
      allOf:
        - required:
          - minimum
        - required:
          - maximum
`

	jsonSchema, _ := parser.ConvertYAMLIntoJSONSchema(yml, nil)
	opts["schema"] = jsonSchema
	opts["forceValidationOnCurrentNode"] = true // use the current node to validate (field not needed)

	return &model.Rule{
		Name:        "Schema of type integer must specify `minimum` and `maximum`",
		Id:          OwaspIntegerLimitLegacy,
		Description: "Integers should be limited to mitigate resource exhaustion attacks.",
		Given: []string{
			`$..[?(@.type)]`,
		},
		Resolved:     false,
		Formats:      model.AllFormats,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function:        "schema",
			FunctionOptions: opts,
		},
		HowToFix: owaspIntegerLimitFix,
	}
}

// It will return duplicate errors for each branch of any if/else/then logic
func GetOWASPIntegerFormatRule() *model.Rule {

	// create a schema to match against.
	opts := make(map[string]interface{})
	yml := `type: object
if:
  properties:
    type:
      enum: 
        - integer
then:
  allOf:
    - required:
      - format
else:
  if:
    properties:
      type:
        type: array
  then:
    if:
      properties:
        type:
          contains:
            enum:
              - integer
    then:
      allOf:
        - required:
          - format
`

	jsonSchema, _ := parser.ConvertYAMLIntoJSONSchema(yml, nil)
	opts["schema"] = jsonSchema
	opts["forceValidationOnCurrentNode"] = true // use the current node to validate (field not needed)

	return &model.Rule{
		Name:        "Schema of type integer must specify format (int32 or int64)",
		Id:          OwaspIntegerFormat,
		Description: "Integers should be limited to mitigate resource exhaustion attacks.",
		Given: []string{
			`$..[?(@.type)]`,
		},
		Resolved:     false,
		Formats:      model.AllFormats,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function:        "schema",
			FunctionOptions: opts,
		},
		HowToFix: owaspIntegerFormatFix,
	}
}

func GetOWASPNoAdditionalPropertiesRule() *model.Rule {

	return &model.Rule{
		Name:         "If the additionalProperties keyword is used it must be set to false",
		Id:           OwaspNoAdditionalProperties,
		Description:  "By default JSON Schema allows additional properties, which can potentially lead to mass assignment issues.",
		Given:        `$..[?(@.type=="object" && @.additionalProperties)]`,
		Resolved:     false,
		Formats:      append(model.OAS2Format, model.OAS3Format...),
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityWarn,
		Then: []model.RuleAction{
			{
				Field:    "additionalProperties",
				Function: "falsy",
			},
		},
		HowToFix: owaspNoAdditionalPropertiesFix,
	}
}

func GetOWASPConstrainedAdditionalPropertiesRule() *model.Rule {

	return &model.Rule{
		Name:         "Objects should not allow unconstrained additionalProperties",
		Id:           OwaspConstrainedAdditionalProperties,
		Description:  "By default JSON Schema allows additional properties, which can potentially lead to mass assignment issues.",
		Given:        `$..[?(@.type=="object" && @.additionalProperties!=true && @.additionalProperties!=false )]`,
		Resolved:     false,
		Formats:      model.OAS3AllFormat,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Field:    "maxProperties",
			Function: "defined",
		},
		HowToFix: owaspNoAdditionalPropertiesFix,
	}
}

func GetOWASPSecurityHostsHttpsOAS2Rule() *model.Rule {

	opts := make(map[string]interface{})
	yml := `type: array
items:
  type: "string"
  enum: [https]`

	jsonSchema, _ := parser.ConvertYAMLIntoJSONSchema(yml, nil)
	opts["schema"] = jsonSchema
	opts["forceValidationOnCurrentNode"] = true // use the current node to validate (field not needed)

	return &model.Rule{
		Name:        "All servers defined must use TLS (https), no other protocol is permitted",
		Id:          OwaspSecurityHostsHttpsOAS2,
		Description: "All server interactions MUST use TLS, so the only OpenAPI scheme being used should be `https`.",
		Given: []string{
			`$.schemes`,
		},
		Resolved:     false,
		Formats:      model.OAS2Format,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function:        "schema",
			FunctionOptions: opts,
		},
		HowToFix: owaspSecurityHostsHttpsOAS2Fix,
	}
}

func GetOWASPSecurityHostsHttpsOAS3Rule() *model.Rule {

	// create a schema to match against.
	opts := make(map[string]interface{})
	opts["match"] = "^https:"
	comp, _ := regexp.Compile(opts["match"].(string))

	return &model.Rule{
		Name:        "Server URLs MUST begin with `https`. No other protocol is permitted",
		Id:          OwaspSecurityHostsHttpsOAS3,
		Description: "All server interactions MUST use the https protocol, meaning server URLs should begin `https://`.",
		Given: []string{
			`$.servers..url`,
		},
		Resolved:     false,
		Formats:      model.OAS3Format,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function:        "pattern",
			FunctionOptions: opts,
		},
		PrecompiledPattern: comp,
		HowToFix:           owaspSecurityHostsHttpsOAS3Fix,
	}
}
