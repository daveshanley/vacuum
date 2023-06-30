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
	// TODO: not exactly equal to the one in spectral
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
		Id:           OwaspNoNumericIDs,
		Formats:      model.AllFormats,
		Description:  "OWASP API1:2019 - Use random IDs that cannot be guessed. UUIDs are preferred",
		Given:        `$.paths..parameters[*][?(@.name == "id" || @.name =~ /(_id|Id|-id)$/)))]`,
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Field:           "schema",
			Function:        "schema",
			FunctionOptions: opts,
		},
	}
}

func GetOWASPNoHttpBasicRule() *model.Rule {

	// create a schema to match against.
	opts := make(map[string]interface{})
	opts["notMatch"] = "basic"
	comp, _ := regexp.Compile(opts["notMatch"].(string))

	return &model.Rule{
		Name:         "Security scheme uses HTTP Basic. Use a more secure authentication method, like OAuth 2.0",
		Id:           OwaspNoHttpBasic,
		Formats:      model.AllFormats,
		Description:  "Basic authentication credentials transported over network are more susceptible to interception than other forms of authentication, and as they are not encrypted it means passwords and tokens are more easily leaked",
		Given:        `$.components.securitySchemes[*]`,
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Field:           "scheme",
			Function:        "pattern",
			FunctionOptions: opts,
		},
		PrecompiledPattern: comp,
	}
}

func GetOWASPNoAPIKeysInURLRule() *model.Rule {

	// create a schema to match against.
	opts := make(map[string]interface{})
	opts["notMatch"] = "^(path|query)$"
	comp, _ := regexp.Compile(opts["notMatch"].(string))

	return &model.Rule{
		Name:         "ApiKey passed in URL: {{error}}",
		Id:           OwaspNoAPIKeysInURL,
		Formats:      model.OAS3AllFormat,
		Description:  "API Keys are (usually opaque) strings that\nare passed in headers, cookies or query parameters\nto access APIs.\nThose keys can be eavesdropped, especially when they are stored\nin cookies or passed as URL parameters.\n```\nsecurity:\n- ApiKey: []\npaths:\n  /books: {}\n  /users: {}\nsecuritySchemes:\n  ApiKey:\n    type: apiKey\n    in: cookie\n    name: X-Api-Key\n```",
		Given:        `$..securitySchemes[*][?(@.type=="apiKey")].in`,
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function:        "pattern",
			FunctionOptions: opts,
		},
		PrecompiledPattern: comp,
	}
}

func GetOWASPNoCredentialsInURLRule() *model.Rule {

	// create a schema to match against.
	opts := make(map[string]interface{})
	opts["notMatch"] = `(?i)^.*(client_?secret|token|access_?token|refresh_?token|id_?token|password|secret|api-?key).*$`
	comp, _ := regexp.Compile(opts["notMatch"].(string))

	return &model.Rule{
		Name:         "Security credentials detected in path parameter: {{value}}",
		Id:           OwaspNoCredentialsInURL,
		Formats:      model.OAS3AllFormat,
		Description:  "URL parameters MUST NOT contain credentials such as API key, password, or secret. See [RAC_GEN_004](https://docs.italia.it/italia/piano-triennale-ict/lg-modellointeroperabilita-docs/it/bozza/doc/04_Raccomandazioni%20di%20implementazione/04_raccomandazioni-tecniche-generali/01_globali.html?highlight=credenziali#rac-gen-004-non-passare-credenziali-o-dati-riservati-nellurl)",
		Given:        `$..parameters[*][?(@.in =~ /(query|path)/)].name`,
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function:        "pattern",
			FunctionOptions: opts,
		},
		PrecompiledPattern: comp,
	}
}

func GetOWASPAuthInsecureSchemesRule() *model.Rule {

	// create a schema to match against.
	opts := make(map[string]interface{})
	opts["notMatch"] = `^(negotiate|oauth)$`
	comp, _ := regexp.Compile(opts["notMatch"].(string))

	return &model.Rule{
		Name:         "Authentication scheme is considered outdated or insecure: {{value}}",
		Id:           OwaspAuthInsecureSchemes,
		Formats:      model.OAS3AllFormat,
		Description:  "There are many [HTTP authorization schemes](https://www.iana.org/assignments/http-authschemes/) but some of them are now considered insecure, such as negotiating authentication using specifications like NTLM or OAuth v1",
		Given:        `$..securitySchemes[*][?(@.type=="http")].scheme`,
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function:        "pattern",
			FunctionOptions: opts,
		},
		PrecompiledPattern: comp,
	}
}

func GetOWASPJWTBestPracticesRule() *model.Rule {

	// create a schema to match against.
	opts := make(map[string]interface{})
	opts["match"] = `.*RFC8725.*`
	comp, _ := regexp.Compile(opts["match"].(string))

	return &model.Rule{
		Name:        "Security schemes using JWTs must explicitly declare support for RFC8725 in the description",
		Id:          OwaspJWTBestPractices,
		Description: "",
		Given: []string{
			`$..securitySchemes[*][?(@.type=="oauth2")]`,
			`$..securitySchemes[*][?(@.bearerFormat=="jwt" || @.bearerFormat=="JWT")]`,
		},
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Field:           "description",
			Function:        "pattern",
			FunctionOptions: opts,
		},
		PrecompiledPattern: comp,
	}
}

// https://github.com/italia/api-oas-checker/blob/master/security/security.yml
func GetOWASPProtectionGlobalUnsafeRule() *model.Rule {
	return &model.Rule{
		Name:         "This operation is not protected by any security scheme",
		Id:           OwaspProtectionGlobalUnsafe,
		Description:  "Your API should be protected by a `security` rule either at global or operation level. All operations should be protected especially when they\nnot safe (methods that do not alter the state of the server) \nHTTP methods like `POST`, `PUT`, `PATCH` and `DELETE`.\nThis is done with one or more non-empty `security` rules.\n\nSecurity rules are defined in the `securityScheme` section",
		Given:        `$`,
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
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
	}
}

// https://github.com/italia/api-oas-checker/blob/master/security/security.yml
func GetOWASPProtectionGlobalUnsafeStrictRule() *model.Rule {
	return &model.Rule{
		Name:         "This operation is not protected by any security scheme",
		Id:           OwaspProtectionGlobalUnsafeStrict,
		Description:  "Check if the operation is protected at operation level.\nOtherwise, check the global `#/security` property",
		Given:        `$`,
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
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
	}
}

// https://github.com/italia/api-oas-checker/blob/master/security/security.yml
func GetOWASPProtectionGlobalSafeRule() *model.Rule {
	return &model.Rule{
		Name:         "This operation is not protected by any security scheme",
		Id:           OwaspProtectionGlobalSafe,
		Description:  "Check if the operation is protected at operation level.\nOtherwise, check the global `#/security` property",
		Given:        `$`,
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
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
	}
}

func GetOWASPDefineErrorValidationRule() *model.Rule {
	return &model.Rule{
		Name:         "Missing error response of either 400, 422 or 4XX",
		Id:           OwaspDefineErrorValidation,
		Description:  "Carefully define schemas for all the API responses, including either 400, 422 or 4XX responses which describe errors caused by invalid requests",
		Given:        `$.paths..responses`,
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Function: "owaspDefineErrorDefinition",
		},
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
		Name:         "Operation is missing {{property}}",
		Id:           OwaspDefineErrorResponses401,
		Description:  "OWASP API Security recommends defining schemas for all responses, even errors. The 401 describes what happens when a request is unauthorized, so its important to define this not just for documentation, but to empower contract testing to make sure the proper JSON structure is being returned instead of leaking implementation details in backtraces",
		Given:        `$.paths..responses`,
		Resolved:     false,
		Formats:      model.AllFormats,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
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
		Name:         "Operation is missing {{property}}",
		Id:           OwaspDefineErrorResponses500,
		Description:  "OWASP API Security recommends defining schemas for all responses, even errors. The 500 describes what happens when a request fails with an internal server error, so its important to define this not just for documentation, but to empower contract testing to make sure the proper JSON structure is being returned instead of leaking implementation details in backtraces",
		Given:        `$.paths..responses`,
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
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
		Name:         "All 2XX and 4XX responses should define rate limiting headers",
		Id:           OwaspRateLimit,
		Description:  "Define proper rate limiting to avoid attackers overloading the API. There are many ways to implement rate-limiting, but most of them involve using HTTP headers, and there are two popular ways to do that:\n\nIETF Draft HTTP RateLimit Headers:. https://datatracker.ietf.org/doc/draft-ietf-httpapi-ratelimit-headers/\n\nCustomer headers like X-Rate-Limit-Limit (Twitter: https://developer.twitter.com/en/docs/twitter-api/rate-limits) or X-RateLimit-Limit (GitHub: https://docs.github.com/en/rest/overview/resources-in-the-rest-api)",
		Given:        `$.paths..responses`,
		Resolved:     false,
		Formats:      model.OAS3AllFormat,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
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
	}
}

func GetOWASPRateLimitRetryAfterRule() *model.Rule {

	return &model.Rule{
		Name:         "A 429 response should define a Retry-After header",
		Id:           OwaspRateLimitRetryAfter,
		Description:  "Define proper rate limiting to avoid attackers overloading the API. Part of that involves setting a Retry-After header so well meaning consumers are not polling and potentially exacerbating problems",
		Given:        `$..responses.429.headers`,
		Resolved:     false,
		Formats:      model.OAS3AllFormat,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Field:    "Retry-After",
			Function: "defined",
		},
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
		Name:         "Operation is missing rate limiting response in {{property}}",
		Id:           OwaspDefineErrorResponses429,
		Description:  "OWASP API Security recommends defining schemas for all responses, even errors. A HTTP 429 response signals the API client is making too many requests, and will supply information about when to retry so that the client can back off calmly without everything breaking. Defining this response is important not just for documentation, but to empower contract testing to make sure the proper JSON structure is being returned instead of leaking implementation details in backtraces. It also ensures your API/framework/gateway actually has rate limiting set up",
		Given:        `$.paths..responses`,
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
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
	}
}

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
		Description: "Array size should be limited to mitigate resource exhaustion attacks. This can be done using `maxItems`. You should ensure that the subschema in `items` is constrained too",
		Given: []string{
			`$..[?(@.type)]`,
		},
		Resolved:     false,
		Formats:      model.AllFormats,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function:        "schema",
			FunctionOptions: opts,
		},
	}
}

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
		Description: "String size should be limited to mitigate resource exhaustion attacks. This can be done using `maxLength`, `enum` or `const`",
		Given: []string{
			`$..[?(@.type)]`,
		},
		Resolved:     false,
		Formats:      model.AllFormats,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function:        "schema",
			FunctionOptions: opts,
		},
	}
}

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
		Name:        "Schema of type string must specify a format, pattern, enum, or const",
		Id:          OwaspStringRestricted,
		Description: "To avoid unexpected values being sent or leaked, ensure that strings have either a `format`, RegEx `pattern`, `enum`, or `const`",
		Given: []string{
			`$..[?(@.type)]`,
		},
		Resolved:     false,
		Formats:      model.AllFormats,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function:        "schema",
			FunctionOptions: opts,
		},
	}
}

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
		Name:        "Schema of type integer must specify minimum and maximum",
		Id:          OwaspIntegerLimit,
		Description: "Integers should be limited to mitigate resource exhaustion attacks. This can be done using `minimum` and `maximum`, which can with e.g.: avoiding negative numbers when positive are expected, or reducing unreasonable iterations like doing something 1000 times when 10 is expected",
		Given: []string{
			`$..[?(@.type)]`,
		},
		Resolved:     false,
		Formats:      model.AllFormats,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function:        "schema",
			FunctionOptions: opts,
		},
	}
}

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
		Name:        "Schema of type integer must specify minimum and maximum",
		Id:          OwaspIntegerLimitLegacy,
		Description: "Integers should be limited to mitigate resource exhaustion attacks. This can be done using `minimum` and `maximum`, which can with e.g.: avoiding negative numbers when positive are expected, or reducing unreasonable iterations like doing something 1000 times when 10 is expected",
		Given: []string{
			`$..[?(@.type)]`,
		},
		Resolved:     false,
		Formats:      model.AllFormats,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function:        "schema",
			FunctionOptions: opts,
		},
	}
}

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
		Description: "Integers should be limited to mitigate resource exhaustion attacks. Specifying whether int32 or int64 is expected via `format`",
		Given: []string{
			`$..[?(@.type)]`,
		},
		Resolved:     false,
		Formats:      model.AllFormats,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function:        "schema",
			FunctionOptions: opts,
		},
	}
}

func GetOWASPNoAdditionalPropertiesRule() *model.Rule {

	return &model.Rule{
		Name:         "If the additionalProperties keyword is used it must be set to false",
		Id:           OwaspNoAdditionalProperties,
		Description:  "By default JSON Schema allows additional properties, which can potentially lead to mass assignment issues, where unspecified fields are passed to the API without validation. Disable them with `additionalProperties: false` or add `maxProperties`",
		Given:        `$..[?(@.type=="object" && @.additionalProperties)]`,
		Resolved:     false,
		Formats:      append(model.OAS2Format, model.OAS3Format...),
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Field:    "additionalProperties",
			Function: "falsy",
		},
	}
}

func GetOWASPConstrainedAdditionalPropertiesRule() *model.Rule {

	return &model.Rule{
		Name:         "Objects should not allow unconstrained additionalProperties",
		Id:           OwaspConstrainedAdditionalProperties,
		Description:  "By default JSON Schema allows additional properties, which can potentially lead to mass assignment issues, where unspecified fields are passed to the API without validation. Disable them with `additionalProperties: false` or add `maxProperties`",
		Given:        `$..[?(@.type=="object" && @.additionalProperties!=true && @.additionalProperties!=false )]`,
		Resolved:     false,
		Formats:      model.OAS3AllFormat,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Field:    "maxProperties",
			Function: "defined",
		},
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
		Name:        "All servers defined MUST use https, and no other protocol is permitted",
		Id:          OwaspSecurityHostsHttpsOAS2,
		Description: "All server interactions MUST use the https protocol, so the only OpenAPI scheme being used should be `https`.\n\nLearn more about the importance of TLS (over SSL) here: https://cheatsheetseries.owasp.org/cheatsheets/Transport_Layer_Protection_Cheat_Sheet.html",
		Given: []string{
			`$.schemes`,
		},
		Resolved:     false,
		Formats:      model.OAS2Format,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function:        "schema",
			FunctionOptions: opts,
		},
	}
}

func GetOWASPSecurityHostsHttpsOAS3Rule() *model.Rule {

	// create a schema to match against.
	opts := make(map[string]interface{})
	opts["match"] = "^https:"
	comp, _ := regexp.Compile(opts["match"].(string))

	return &model.Rule{
		Name:        "Server URLs MUST begin https://, and no other protocol is permitted",
		Id:          OwaspSecurityHostsHttpsOAS3,
		Description: "All server interactions MUST use the https protocol, meaning server URLs should begin `https://`.\n\nLearn more about the importance of TLS (over SSL) here: https://cheatsheetseries.owasp.org/cheatsheets/Transport_Layer_Protection_Cheat_Sheet.html",
		Given: []string{
			`$.servers..url`,
		},
		Resolved:     false,
		Formats:      model.OAS3Format,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function:        "pattern",
			FunctionOptions: opts,
		},
		PrecompiledPattern: comp,
	}
}
