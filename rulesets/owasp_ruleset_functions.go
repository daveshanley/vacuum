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
		Name:         "OWASP API1:2019",
		Id:           "", // TODO
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
