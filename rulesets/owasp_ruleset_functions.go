package rulesets

import (
	"regexp"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/parser"
)

// rules taken from https://github.com/stoplightio/spectral-owasp-ruleset/blob/main/src/ruleset.ts

func GetOwaspAPIRuleNoNumericIDs() *model.Rule {

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
		Name:         "OWASP API1:2019", // fix
		Id:           "",                // TODO
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
		HowToFix: "", // TODO
	}
}

func GetOWASPRuleSecuritySchemeUseHTTPBasic() *model.Rule {

	// create a schema to match against.
	opts := make(map[string]interface{})
	opts["notMatch"] = "basic"
	comp, _ := regexp.Compile(opts["notMatch"].(string))

	return &model.Rule{
		Name:         "Security scheme uses HTTP Basic",
		Id:           "", // TODO
		Formats:      model.AllFormats,
		Description:  "Security scheme uses HTTP Basic. Use a more secure authentication method, like OAuth 2.0",
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
		HowToFix:           "", // TODO
	}
}

func GetOWASPRuleNoAPIKeysInURL() *model.Rule {

	// create a schema to match against.
	opts := make(map[string]interface{})
	opts["notMatch"] = "^(path|query)$"
	comp, _ := regexp.Compile(opts["notMatch"].(string))

	return &model.Rule{
		Name:         "ApiKey passed in URL: {{error}}",
		Id:           "", // TODO
		Formats:      model.OAS3AllFormat,
		Description:  "API Keys are (usually opaque) strings that\nare passed in headers, cookies or query parameters\nto access APIs.\nThose keys can be eavesdropped, especially when they are stored\nin cookies or passed as URL parameters.\n```\nsecurity:\n- ApiKey: []\npaths:\n  /books: {}\n  /users: {}\nsecuritySchemes:\n  ApiKey:\n    type: apiKey\n    in: cookie\n    name: X-Api-Key\n```",
		Given:        `$..securitySchemes[*][?(@.type=="apiKey")].in`, // TODO, make apiKey case insensitive
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
		HowToFix:           "", // TODO
	}
}

func GetOWASPRuleSecurityCredentialsDetected() *model.Rule {

	// create a schema to match against.
	opts := make(map[string]interface{})
	opts["notMatch"] = `(?i)^.*(client_?secret|token|access_?token|refresh_?token|id_?token|password|secret|api-?key).*$`
	comp, _ := regexp.Compile(opts["notMatch"].(string))

	return &model.Rule{
		Name:         "Security credentials detected in path parameter: {{value}}",
		Id:           "", // TODO
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
		HowToFix:           "", // TODO
	}
}

func GetOWASPRuleAuthInsecureSchemes() *model.Rule {

	// create a schema to match against.
	opts := make(map[string]interface{})
	opts["notMatch"] = `^(negotiate|oauth)$`
	comp, _ := regexp.Compile(opts["notMatch"].(string))

	return &model.Rule{
		Name:         "Authentication scheme is considered outdated or insecure: {{value}}",
		Id:           "", // TODO
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
		HowToFix:           "", // TODO
	}
}

func GetOWASPRuleJWTBestPractices() *model.Rule {

	// create a schema to match against.
	opts := make(map[string]interface{})
	opts["match"] = `.*RFC8725.*`
	comp, _ := regexp.Compile(opts["match"].(string))

	return &model.Rule{
		Name:        "Security schemes using JWTs must explicitly declare support for RFC8725 in the description",
		Id:          "", // TODO
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
		HowToFix:           "", // TODO
	}
}

// TODO: create checkSecurity function similar to the one in spectral // owasp:api2:2019-protection-global-unsafe
func GetOWASPRuleProtectionGlobalUnsafe() *model.Rule {
	return nil
}

// TODO: create checkSecurity function similar to the one in spectral // owasp:api2:2019-protection-global-unsafe-strict
func GetOWASPRuleProtectionGlobalUnsafeStrict() *model.Rule {
	return nil
}

// TODO: create checkSecurity function similar to the one in spectral // owasp:api2:2019-protection-global-safe
func GetOWASPRuleProtectionGlobalSafe() *model.Rule {
	return nil
}

// TO REVIEW, Uses oasOpErrorResponse function by extending it
func GetOWASPRuleDefineErrorValidation() *model.Rule {

	return &model.Rule{
		Name:         "Missing error response of either 400, 422 or 4XX",
		Id:           "", // TODO
		Description:  "Carefully define schemas for all the API responses, including either 400, 422 or 4XX responses which describe errors caused by invalid requests",
		Given:        `$.paths..responses`,
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Function: "owaspDefineError",
		},
		HowToFix: "", // TODO
	}
}

// Had to split into GetOWASPRuleDefineErrorResponses401 and GetOWASPRuleDefineErrorResponses401Content since pb33f does not support path keys for now
func GetOWASPRuleDefineErrorResponses401() *model.Rule {

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
		Id:           "", // TODO
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
		HowToFix: "", // TODO
	}
}

func GetOWASPRuleDefineErrorResponses500() *model.Rule {

	opts := make(map[string]interface{})
	yml := `type: object
required:
  - content`

	jsonSchema, _ := parser.ConvertYAMLIntoJSONSchema(yml, nil)
	opts["schema"] = jsonSchema
	opts["forceValidation"] = true // this will be picked up by the schema function to force validation.

	return &model.Rule{
		Name:         "Operation is missing {{property}}",
		Id:           "", // TODO
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
		HowToFix: "", // TODO
	}
}

func GetOWASPRuleRateLimit() *model.Rule {
	return &model.Rule{
		Name:         "All 2XX and 4XX responses should define rate limiting headers",
		Id:           "", // TODO
		Description:  "Define proper rate limiting to avoid attackers overloading the API. There are many ways to implement rate-limiting, but most of them involve using HTTP headers, and there are two popular ways to do that:\n\nIETF Draft HTTP RateLimit Headers:. https://datatracker.ietf.org/doc/draft-ietf-httpapi-ratelimit-headers/\n\nCustomer headers like X-Rate-Limit-Limit (Twitter: https://developer.twitter.com/en/docs/twitter-api/rate-limits) or X-RateLimit-Limit (GitHub: https://docs.github.com/en/rest/overview/resources-in-the-rest-api)",
		Given:        `$.paths..responses`,
		Resolved:     false,
		Formats:      model.OAS3AllFormat,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "owaspRateLimitDefinition",
		},
		HowToFix: "", // TODO
	}
}

func GetOWASPRuleRateLimitRetryAfter() *model.Rule {

	return &model.Rule{
		Name:         "A 429 response should define a Retry-After header",
		Id:           "", // TODO
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
		HowToFix: "", // TODO
	}
}

// TODO: Not working as expected
func GetOWASPRuleArrayLimit() *model.Rule {

	return &model.Rule{
		Name:        "Schema of type array must specify maxItems",
		Id:          "", // TODO
		Description: "Array size should be limited to mitigate resource exhaustion attacks. This can be done using `maxItems`. You should ensure that the subschema in `items` is constrained too",
		Given: []string{
			`$..[?(@.type=="array")]`,
			`'$..[?(@.type.constructor.name === "Array" && @.type.includes("array"))]`, // only for oas 3
		},
		Resolved:     false,
		Formats:      model.AllFormats,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Field:    "maxItems",
			Function: "defined",
		},
		HowToFix: "", // TODO
	}
}

// TODO: Not working and wrong
func GetOWASPRuleIntegerLimit() *model.Rule {

	return &model.Rule{
		Name:         "Schema of type integer must specify minimum and maximum",
		Id:           "", // TODO
		Description:  "Integers should be limited to mitigate resource exhaustion attacks. This can be done using `minimum` and `maximum`, which can with e.g.: avoiding negative numbers when positive are expected, or reducing unreasonable iterations like doing something 1000 times when 10 is expected",
		Given:        []string{},
		Resolved:     false,
		Formats:      append(model.OAS2Format, model.OAS3Format...),
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Field:    "maxItems",
			Function: "defined",
		},
		HowToFix: "", // TODO
	}
}

func GetOWASPRuleNoAdditionalProperties() *model.Rule {

	return &model.Rule{
		Name:         "If the additionalProperties keyword is used it must be set to false",
		Id:           "", // TODO
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
		HowToFix: "", // TODO
	}
}

func GetOWASPRuleConstrainedAdditionalProperties() *model.Rule {

	return &model.Rule{
		Name:         "Objects should not allow unconstrained additionalProperties",
		Id:           "", // TODO
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
		HowToFix: "", // TODO
	}
}

func GetOWASPRuleDefineErrorResponses429() *model.Rule {

	opts := make(map[string]interface{})
	yml := `type: object
required:
  - content`

	jsonSchema, _ := parser.ConvertYAMLIntoJSONSchema(yml, nil)
	opts["schema"] = jsonSchema
	opts["forceValidation"] = true // this will be picked up by the schema function to force validation.

	return &model.Rule{
		Name:         "Operation is missing rate limiting response in {{property}}",
		Id:           "", // TODO
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
		HowToFix: "", // TODO
	}
}
