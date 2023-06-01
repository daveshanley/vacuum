package rulesets

import (
	"regexp"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/parser"
)

// rules taken from https://github.com/stoplightio/spectral-owasp-ruleset/blob/main/src/ruleset.ts

func GetOwaspAPIRule1NoNumericIDs() *model.Rule {

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

func GetOWASPRule2HTTPBasic() *model.Rule {

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
