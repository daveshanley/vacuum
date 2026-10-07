package rulesets

import (
	"github.com/daveshanley/vacuum/model"
)

// rules taken from https://github.com/stoplightio/spectral-owasp-ruleset/blob/main/src/ruleset.ts

func GetOWASPNoNumericIDsRule() *model.Rule {
	return &model.Rule{
		Name:         "Use random IDs that cannot be guessed.",
		Id:           OwaspNoNumericIDs,
		Formats:      model.OAS3AllFormat,
		Description:  "Use random IDs that cannot be guessed. UUIDs are preferred",
		Given:        `$`,
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "owaspNoNumericIds",
		},
		HowToFix: owaspNoNumericIDsFix,
	}
}

func GetOWASPNoHttpBasicRule() *model.Rule {
	return &model.Rule{
		Name:         "Security scheme uses HTTP Basic.",
		Id:           OwaspNoHttpBasic,
		Formats:      model.OAS3AllFormat,
		Description:  "Security scheme uses HTTP Basic. Use a more secure authentication method, like OAuth 2.0",
		Given:        `$`,
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "owaspNoBasicAuth",
		},
		HowToFix: owaspNoHttpBasicFix,
	}
}

func GetOWASPNoAPIKeysInURLRule() *model.Rule {
	return &model.Rule{
		Name:         "API Key detected in URL",
		Id:           OwaspNoAPIKeysInURL,
		Formats:      model.OAS3AllFormat,
		Description:  "API Key has been detected in a URL",
		Given:        `$`,
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "owaspNoApiKeyInUrl",
		},
		HowToFix: owaspNoAPIKeysInURLFix,
	}
}

func GetOWASPNoCredentialsInURLRule() *model.Rule {
	return &model.Rule{
		Name:         "Security credentials detected in path parameter",
		Id:           OwaspNoCredentialsInURL,
		Formats:      model.OAS3AllFormat,
		Description:  "URL parameters must not contain credentials such as API key, password, or secret.",
		Given:        `$`,
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "owaspNoCredentialsInUrl",
		},
		HowToFix: owaspNoCredentialsInURLFix,
	}
}

func GetOWASPAuthInsecureSchemesRule() *model.Rule {

	return &model.Rule{
		Name:         "Authentication scheme is considered outdated or insecure",
		Id:           OwaspAuthInsecureSchemes,
		Formats:      model.OAS3AllFormat,
		Description:  "Authentication scheme is considered outdated or insecure",
		Given:        `$`,
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "owaspAuthInsecureSchemes",
		},
		HowToFix: owaspAuthInsecureSchemesFix,
	}
}

func GetOWASPJWTBestPracticesRule() *model.Rule {

	return &model.Rule{
		Name:         "JWT descriptions should reference `RFC8725`",
		Id:           OwaspJWTBestPractices,
		Formats:      model.OAS3AllFormat,
		Description:  "Explicitly declared JWTs should document RFC8725. This checks documentation, not runtime token validation.",
		Given:        `$`,
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Function: "owaspJWTBestPractice",
		},
		HowToFix: owaspJWTBestPracticesFix,
	}
}

// https://github.com/italia/api-oas-checker/blob/master/security/security.yml
func GetOWASPProtectionGlobalUnsafeRule() *model.Rule {
	return &model.Rule{
		Name:         "Operation is not protected by any security scheme",
		Id:           OwaspProtectionGlobalUnsafe,
		Formats:      model.OAS3AllFormat,
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
		Formats:      model.OAS3AllFormat,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
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
		Formats:      model.OAS3AllFormat,
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

	opts := make(map[string]interface{})
	opts["codes"] = []interface{}{"400", "422", "4XX"}

	return &model.Rule{
		Name:         "Missing error response of either `400`, `422` or `4XX`",
		Id:           OwaspDefineErrorValidation,
		Description:  "Missing error response of either `400`, `422` or `4XX`, Ensure all errors are documented.",
		Given:        `$`,
		Resolved:     false,
		Formats:      model.OAS3AllFormat,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Function:        "owaspDefineErrorDefinition",
			FunctionOptions: opts,
		},
		HowToFix: owaspDefineErrorValidationFix,
	}
}

func GetOWASPDefineErrorResponses401Rule() *model.Rule {
	opts := make(map[string]interface{})
	opts["code"] = "401"

	return &model.Rule{
		Name:         "Operation is missing a `401` error response",
		Id:           OwaspDefineErrorResponses401,
		Description:  "OWASP API Security recommends defining schemas for all responses, even error: 401",
		Given:        `$`,
		Resolved:     false,
		Formats:      model.OAS3AllFormat,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Function:        "owaspCheckErrorResponse",
			FunctionOptions: opts,
		},
		HowToFix: owaspDefineErrorResponses401Fix,
	}

}

func GetOWASPDefineErrorResponses500Rule() *model.Rule {

	opts := make(map[string]interface{})
	opts["code"] = "500"

	return &model.Rule{
		Name:         "Operation is missing a `500` error response",
		Id:           OwaspDefineErrorResponses500,
		Description:  "OWASP API Security recommends defining schemas for all responses, even error: 500",
		Given:        `$`,
		Resolved:     false,
		Formats:      model.OAS3AllFormat,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Function:        "owaspCheckErrorResponse",
			FunctionOptions: opts,
		},
		HowToFix: owaspDefineErrorResponses500Fix,
	}
}

func GetOWASPRateLimitRule() *model.Rule {
	var (
		xRatelimitLimit = "X-RateLimit-Limit"
		xRateLimitLimit = "X-Rate-Limit-Limit"
		ratelimitLimit  = "RateLimit-Limit||RateLimit-Reset"
		ratelimit       = "RateLimit"
	)

	return &model.Rule{
		Name:         "`2XX` and `4XX` responses should define rate limiting headers",
		Id:           OwaspRateLimit,
		Description:  "Define proper rate limiting to avoid attackers overloading the API.",
		Given:        `$`,
		Resolved:     false,
		Formats:      model.OAS3AllFormat,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "owaspHeaderDefinition",
			FunctionOptions: map[string]interface{}{
				"headers": []string{
					xRatelimitLimit,
					xRateLimitLimit,
					ratelimitLimit,
					ratelimit,
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
		Description:  "Ensure that any `429` response, contains a `Retry-After` header.",
		Given:        `$`,
		Resolved:     true,
		Formats:      model.OAS3AllFormat,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "owaspRatelimitRetryAfter",
		},
		HowToFix: owaspRateLimitRetryAfterFix,
	}
}

func GetOWASPDefineErrorResponses429Rule() *model.Rule {

	opts := make(map[string]interface{})
	opts["code"] = "429"

	return &model.Rule{
		Name:         "Operation is missing a `429` rate limiting error response",
		Id:           OwaspDefineErrorResponses429,
		Description:  "OWASP API Security recommends defining schemas for all responses, even error: 429",
		Given:        `$`,
		Resolved:     true,
		Formats:      model.OAS3AllFormat,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Function:        "owaspCheckErrorResponse",
			FunctionOptions: opts,
		},
		HowToFix: owaspDefineErrorResponses429Fix,
	}
}

func GetOWASPArrayLimitRule() *model.Rule {
	return &model.Rule{
		Name:         "Schema of type array must specify maxItems",
		Id:           OwaspArrayLimit,
		Description:  "Array size should be limited to mitigate resource exhaustion attacks.",
		Given:        `$`,
		Resolved:     false,
		Formats:      model.OAS3AllFormat,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "owaspArrayLimit",
		},
		HowToFix: owaspArrayLimitFix,
	}
}

func GetOWASPStringLimitRule() *model.Rule {
	return &model.Rule{
		Name:         "Schema of type string must specify maxLength, enum, or const",
		Id:           OwaspStringLimit,
		Description:  "String size should be limited to mitigate resource exhaustion attacks.",
		Given:        `$`,
		Resolved:     false,
		Formats:      model.OAS3AllFormat,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "owaspStringLimit",
		},
		HowToFix: owaspStringLimitFix,
	}
}

func GetOWASPStringRestrictedRule() *model.Rule {

	return &model.Rule{
		Name:         "Schema of type string must specify a `format`, `pattern`, `enum`, or `const`",
		Id:           OwaspStringRestricted,
		Description:  "String must specify a `format`, RegEx `pattern`, `enum`, or `const`",
		Given:        `$`,
		Resolved:     false,
		Formats:      model.OAS3AllFormat,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "owaspStringRestricted",
		},
		HowToFix: owaspStringRestrictedFix,
	}
}

func GetOWASPIntegerLimitRule() *model.Rule {
	return &model.Rule{
		Name:         "Schema of type integer must specify `minimum` and `maximum` or `exclusiveMinimum` and `exclusiveMaximum`",
		Id:           OwaspIntegerLimit,
		Description:  "Integers should be limited via min/max values to mitigate resource exhaustion attacks.",
		Given:        `$`,
		Resolved:     false,
		Formats:      model.OAS3AllFormat,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "owaspIntegerLimit",
		},
		HowToFix: owaspIntegerLimitFix,
	}
}

// OwaspIntegerLimitLegacy removed in 0.7.0

func GetOWASPIntegerFormatRule() *model.Rule {
	return &model.Rule{
		Name:         "Schema of type integer must specify format (int32 or int64)",
		Id:           OwaspIntegerFormat,
		Description:  "Integers should be limited to mitigate resource exhaustion attacks.",
		Given:        `$`,
		Resolved:     false,
		Formats:      model.OAS3AllFormat,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "owaspIntegerFormat",
		},
		HowToFix: owaspIntegerFormatFix,
	}
}

func GetOWASPNoAdditionalPropertiesRule() *model.Rule {

	return &model.Rule{
		Name:         "Request objects should restrict additional properties",
		Id:           OwaspNoAdditionalProperties,
		Description:  "By default JSON Schema allows additional properties, which can potentially lead to mass assignment issues.",
		Given:        `$`,
		Resolved:     false,
		Formats:      model.OAS3AllFormat,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityWarn,
		Then: []model.RuleAction{
			{
				Function: "owaspNoAdditionalProperties",
			},
		},
		HowToFix: owaspNoAdditionalPropertiesFix,
	}
}

func GetOWASPConstrainedAdditionalPropertiesRule() *model.Rule {

	return &model.Rule{
		Name:         "Request maps should limit their property count",
		Id:           OwaspConstrainedAdditionalProperties,
		Description:  "Request maps should declare maxProperties to limit resource consumption. This does not restrict which properties may be assigned.",
		Given:        `$`,
		Resolved:     false,
		Formats:      model.OAS3AllFormat,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Function: "owaspNoAdditionalPropertiesConstrained",
		},
		HowToFix: owaspConstrainedAdditionalPropertiesFix,
	}
}

// OwaspSecurityHostsHttpsOAS2 removed in 0.7.0

func GetOWASPSecurityHostsHttpsOAS3Rule() *model.Rule {

	return &model.Rule{
		Name:         "Explicit server URL schemes must use HTTPS",
		Id:           OwaspSecurityHostsHttpsOAS3,
		Description:  "Root, path, and operation server URLs must use HTTPS when an explicit scheme is declared. Relative URLs inherit their deployment transport.",
		Given:        `$`,
		Resolved:     false,
		Formats:      model.OAS3AllFormat,
		RuleCategory: model.RuleCategories[model.CategoryOWASP],
		Recommended:  true,
		Type:         Validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "owaspHostsHttps",
		},
		HowToFix: owaspSecurityHostsHttpsOAS3Fix,
	}
}

// GetOWASPOAuthNoPasswordRule rejects the OAuth password flow.
func GetOWASPOAuthNoPasswordRule() *model.Rule {
	return &model.Rule{
		Name: "OAuth password flow should not be used", Id: OwaspOAuthNoPassword,
		Description: "Use authorization code with PKCE for user authorization instead of the OAuth password flow.",
		Given:       `$`, Formats: model.OAS3AllFormat, Resolved: false,
		RuleCategory: model.RuleCategories[model.CategoryOWASP], Recommended: true,
		Type: Validation, Severity: model.SeverityError,
		Then:     model.RuleAction{Function: "owaspAuthInsecureSchemes", FunctionOptions: map[string]interface{}{"flow": "password"}},
		HowToFix: owaspOAuthFlowFix,
	}
}

// GetOWASPOAuthNoImplicitRule rejects the OAuth implicit flow.
func GetOWASPOAuthNoImplicitRule() *model.Rule {
	return &model.Rule{
		Name: "OAuth implicit flow should not be used", Id: OwaspOAuthNoImplicit,
		Description: "Use authorization code with PKCE for user authorization instead of the OAuth implicit flow.",
		Given:       `$`, Formats: model.OAS3AllFormat, Resolved: false,
		RuleCategory: model.RuleCategories[model.CategoryOWASP], Recommended: true,
		Type: Validation, Severity: model.SeverityError,
		Then:     model.RuleAction{Function: "owaspAuthInsecureSchemes", FunctionOptions: map[string]interface{}{"flow": "implicit"}},
		HowToFix: owaspOAuthFlowFix,
	}
}

// GetOWASPAuthURLsHTTPSRule checks OAuth and OpenID endpoint transport.
func GetOWASPAuthURLsHTTPSRule() *model.Rule {
	return &model.Rule{
		Name: "Authentication endpoint URLs must use HTTPS", Id: OwaspAuthURLsHTTPS,
		Description: "Explicit OAuth and OpenID endpoint URL schemes must use HTTPS. Relative URLs inherit their deployment transport.",
		Given:       `$`, Formats: model.OAS3AllFormat, Resolved: false,
		RuleCategory: model.RuleCategories[model.CategoryOWASP], Recommended: true,
		Type: Validation, Severity: model.SeverityError,
		Then:     model.RuleAction{Function: "owaspAuthURLsHTTPS"},
		HowToFix: owaspAuthURLsHTTPSFix,
	}
}
