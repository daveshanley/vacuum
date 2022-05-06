// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package rulesets

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"github.com/mitchellh/mapstructure"
	"github.com/xeipuuv/gojsonschema"
	"strings"
	"sync"
)

//go:embed schemas/ruleset.schema.json
var rulesetSchema string

const (
	warn          = "warn"
	err           = "error"
	info          = "info"
	hint          = "hint"
	style         = "style"
	validation    = "validation"
	allPaths      = "$.paths[*]"
	allOperations = "[?(@.get || @.post || @.put || @.patch || @.delete || @.trace || @.options || @.head)]"

	operationSuccessResponse          = "operation-success-response"
	operationOperationIdUnique        = "operation-operationId-unique"
	operationOperationId              = "operation-operationId"
	operationParameters               = "operation-parameters"
	operationSingularTag              = "operation-singular-tag"
	operationTagDefined               = "operation-tag-defined"
	pathParamsRule                    = "path-params"
	contactProperties                 = "contact-properties"
	infoContact                       = "info-contact"
	infoDescription                   = "info-description"
	infoLicense                       = "info-license"
	licenseUrl                        = "license-url"
	openAPITagsAlphabetical           = "openapi-tags-alphabetical"
	openAPITags                       = "openapi-tags"
	operationTags                     = "operation-tags"
	operationDescription              = "operation-description"
	componentDescription              = "component-description"
	operationOperationIdValidInUrl    = "operation-operationId-valid-in-url"
	pathDeclarationsMustExist         = "path-declarations-must-exist"
	pathKeysNoTrailingSlash           = "path-keys-no-trailing-slash"
	pathNotIncludeQuery               = "path-not-include-query"
	tagDescription                    = "tag-description"
	noRefSiblings                     = "no-$ref-siblings"
	oas3UnusedComponent               = "oas3-unused-component"
	oas2UnusedDefinition              = "oas2-unused-definition"
	oas2APIHost                       = "oas2-api-host"
	oas2APISchemes                    = "oas2-api-schemes"
	oas2Discriminator                 = "oas2-discriminator"
	oas2HostNotExample                = "oas2-host-not-example"
	oas3HostNotExample                = "oas3-host-not-example.com"
	oas2HostTrailingSlash             = "oas2-host-trailing-slash"
	oas2ParameterDescription          = "oas2-parameter-description"
	oas3ParameterDescription          = "oas3-parameter-description"
	oas3OperationSecurityDefined      = "oas3-operation-security-defined"
	oas2OperationSecurityDefined      = "oas2-operation-security-defined"
	oas3ValidSchemaExample            = "oas3-valid-schema-example"
	oas2ValidSchemaExample            = "oas2-valid-schema-example"
	typedEnum                         = "typed-enum"
	duplicatedEntryInEnum             = "duplicated-entry-in-enum"
	noEvalInMarkdown                  = "no-eval-in-markdown"
	noScriptTagsInMarkdown            = "no-script-tags-in-markdown"
	descriptionDuplication            = "description-duplication"
	oas3APIServers                    = "oas3-api-servers"
	oas2OperationFormDataConsumeCheck = "oas2-operation-formData-consume-check"
	oas2AnyOf                         = "oas2-anyOf"
	oas2OneOf                         = "oas2-oneOf"
	oas2Schema                        = "oas2-schema"
	oas3Schema                        = "oas3-schema"
	SpectralOpenAPI                   = "spectral:oas"
	SpectralRecommended               = "recommended"
	SpectralAll                       = "all"
	SpectralOff                       = "off"
)

var AllOperationsPath = fmt.Sprintf("%s%s", allPaths, allOperations)

type ruleSetsModel struct {
	openAPIRuleSet *RuleSet
}

// RuleSets is used to generate default RuleSets built into vacuum
type RuleSets interface {

	// GenerateOpenAPIDefaultRuleSet generates a ready to run pointer to a model.RuleSet containing all
	// OpenAPI rules supported by vacuum. Passing all these rules would be considered a very good quality specification.
	GenerateOpenAPIDefaultRuleSet() *RuleSet

	// GenerateOpenAPIRecommendedRuleSet generates a ready to run pointer to a model.RuleSet that contains only
	// recommended rules (not all rules). Passing all these rules would result in a quality specification
	GenerateOpenAPIRecommendedRuleSet() *RuleSet

	// GenerateRuleSetFromSuppliedRuleSet will generate a ready to run ruleset based on a supplied configuration. This
	// will look for any extensions and apply all rules turned on, turned off and any custom rules.
	GenerateRuleSetFromSuppliedRuleSet(config *RuleSet) *RuleSet
}

var rulesetsSingleton *ruleSetsModel
var openAPIRulesGrab sync.Once

func BuildDefaultRuleSets() RuleSets {
	openAPIRulesGrab.Do(func() {
		rulesetsSingleton = &ruleSetsModel{
			openAPIRuleSet: generateDefaultOpenAPIRuleSet(),
		}
	})

	return rulesetsSingleton
}

func (rsm ruleSetsModel) GenerateOpenAPIDefaultRuleSet() *RuleSet {
	return rsm.openAPIRuleSet
}

func (rsm ruleSetsModel) GenerateOpenAPIRecommendedRuleSet() *RuleSet {

	filtered := make(map[string]*model.Rule)
	for ruleName, rule := range rsm.openAPIRuleSet.Rules {
		if rule.Recommended {
			filtered[ruleName] = rule
		}
	}

	// copy.
	modifiedRS := *rsm.openAPIRuleSet
	modifiedRS.Rules = filtered

	return &modifiedRS
}

func (rsm ruleSetsModel) GenerateRuleSetFromSuppliedRuleSet(ruleset *RuleSet) *RuleSet {

	extends := ruleset.GetExtendsValue()

	rs := &RuleSet{
		DocumentationURI: "https://quobix.com/vacuum/rulesets",
		Formats:          ruleset.Formats,
		Extends:          ruleset.Extends,
		RuleDefinitions:  ruleset.RuleDefinitions,
	}

	// default and explicitly recommended
	if extends[SpectralOpenAPI] == SpectralRecommended || extends[SpectralOpenAPI] == SpectralOpenAPI {
		rs = rsm.GenerateOpenAPIRecommendedRuleSet()
		rs.DocumentationURI = "https://quobix.com/vacuum/rulesets/recommended"
	}

	// all rules
	if extends[SpectralOpenAPI] == SpectralAll {
		rs = rsm.openAPIRuleSet
		rs.DocumentationURI = "https://quobix.com/vacuum/rulesets/all"
	}

	// all rules
	if extends[SpectralOpenAPI] == SpectralOff {
		rs.DocumentationURI = "https://quobix.com/vacuum/rulesets/off"
	}

	// add definitions.
	rs.RuleDefinitions = ruleset.RuleDefinitions

	// now all the base rules are in, let's run through the raw definitions and decide
	// what we need to add, enable, disable, replace or change severity on.
	for k, v := range rs.RuleDefinitions {

		// let's try to cast to a string first (enable/disable/severity)
		if evalStr, ok := v.(string); ok {

			// let's check to see if this rule exists
			if rs.Rules[k] == nil {

				// we don't know anything about this rule, so skip it.
				continue

			}

			switch evalStr {
			case err, warn, info, hint:
				rs.Rules[k].Severity = evalStr
			case SpectralOff:
				delete(rs.Rules, k) // remove it completely
			}
		}
	}

	return rs
}

func generateDefaultOpenAPIRuleSet() *RuleSet {

	rules := make(map[string]*model.Rule)
	rules[operationSuccessResponse] = GetOperationSuccessResponseRule()
	rules[operationOperationIdUnique] = GetOperationIdUniqueRule()
	rules[operationOperationId] = GetOperationIdRule()
	rules[operationParameters] = GetOperationParametersRule()
	rules[operationSingularTag] = GetOperationSingleTagRule()
	rules[operationTagDefined] = GetGlobalOperationTagsRule()
	rules[pathParamsRule] = GetPathParamsRule()
	rules[contactProperties] = GetContactPropertiesRule()
	rules[infoContact] = GetInfoContactRule()
	rules[infoDescription] = GetInfoDescriptionRule()
	rules[infoLicense] = GetInfoLicenseRule()
	rules[licenseUrl] = GetInfoLicenseUrlRule()
	rules[openAPITagsAlphabetical] = GetOpenApiTagsAlphabeticalRule()
	rules[openAPITags] = GetOpenApiTagsRule()
	rules[operationTags] = GetOperationTagsRule()
	rules[operationDescription] = GetOperationDescriptionRule()
	rules[componentDescription] = GetComponentDescriptionsRule()
	rules[operationOperationIdValidInUrl] = GetOperationIdValidInUrlRule()
	rules[pathDeclarationsMustExist] = GetPathDeclarationsMustExistRule()
	rules[pathKeysNoTrailingSlash] = GetPathNoTrailingSlashRule()
	rules[pathNotIncludeQuery] = GetPathNotIncludeQueryRule()
	rules[tagDescription] = GetTagDescriptionRequiredRule()
	rules[noRefSiblings] = GetNoRefSiblingsRule()
	rules[oas3UnusedComponent] = GetOAS3UnusedComponentRule()
	rules[oas2UnusedDefinition] = GetOAS2UnusedComponentRule()
	rules[oas2APIHost] = GetOAS2APIHostRule()
	rules[oas2APISchemes] = GetOAS2APISchemesRule()
	rules[oas2Discriminator] = GetOAS2DiscriminatorRule()
	rules[oas2HostNotExample] = GetOAS2HostNotExampleRule()
	rules[oas3HostNotExample] = GetOAS3HostNotExampleRule()
	rules[oas2HostTrailingSlash] = GetOAS2HostTrailingSlashRule()
	rules[oas2ParameterDescription] = GetOAS2ParameterDescriptionRule()
	rules[oas3ParameterDescription] = GetOAS3ParameterDescriptionRule()
	rules[oas3OperationSecurityDefined] = GetOAS3SecurityDefinedRule()
	rules[oas2OperationSecurityDefined] = GetOAS2SecurityDefinedRule()
	rules[oas3ValidSchemaExample] = GetOAS3ExamplesRule()
	rules[oas2ValidSchemaExample] = GetOAS2ExamplesRule()
	rules[typedEnum] = GetTypedEnumRule()
	rules[duplicatedEntryInEnum] = GetDuplicatedEntryInEnumRule()
	rules[noEvalInMarkdown] = GetNoEvalInMarkdownRule()
	rules[noScriptTagsInMarkdown] = GetNoScriptTagsInMarkdownRule()
	rules[descriptionDuplication] = GetDescriptionDuplicationRule()
	rules[oas3APIServers] = GetAPIServersRule()
	rules[oas2OperationFormDataConsumeCheck] = GetOAS2FormDataConsumesRule()
	rules[oas2AnyOf] = GetOAS2PolymorphicAnyOfRule()
	rules[oas2OneOf] = GetOAS2PolymorphicOneOfRule()

	// TODO: enable for a different ruleset.
	//rules[oas2Schema] = GetOAS2SchemaRule()
	//rules[oas3Schema] = GetOAS3SchemaRule()

	set := &RuleSet{
		DocumentationURI: "https://quobix.com/vacuum/rules/openapi",
		Rules:            rules,
	}

	return set

}

// RuleSet represents a collection of Rule definitions.
type RuleSet struct {
	DocumentationURI string                 `json:"documentationUrl"`
	Formats          []string               `json:"formats"`
	RuleDefinitions  map[string]interface{} `json:"rules"` // this can be either a string, or an entire rule (super annoying, stoplight).
	Rules            map[string]*model.Rule `json:"-"`
	Extends          interface{}            `json:"extends"` // can be string or tuple (again... why stoplight?)
	extendsMeta      map[string]string
	schemaLoader     gojsonschema.JSONLoader
}

// GetExtendsValue returns an array of maps defining which ruleset this one extends. The value can be
// a single string or an array of tuples, so this normalizes things into a standard structure.
func (rs *RuleSet) GetExtendsValue() map[string]string {
	if rs.extendsMeta != nil {
		return rs.extendsMeta
	}
	m := make(map[string]string)

	if rs.Extends != nil {
		if extStr, ok := rs.Extends.(string); ok {
			m[extStr] = extStr
		}
		if extArray, ok := rs.Extends.([]interface{}); ok {
			for _, arr := range extArray {
				if castArr, k := arr.([]interface{}); k {
					if len(castArr) == 2 {
						m[castArr[0].(string)] = castArr[1].(string)
					}
				}
				if castArr, k := arr.(string); k {
					m[castArr] = castArr
				}
			}
		}
	}
	rs.extendsMeta = m
	return m
}

// CreateRuleSetUsingJSON will create a new RuleSet instance from a JSON byte array
func CreateRuleSetUsingJSON(jsonData []byte) (*RuleSet, error) {
	jsonString := string(jsonData)
	if !utils.IsJSON(jsonString) {
		return nil, errors.New("data is not JSON")
	}

	jsonLoader := gojsonschema.NewStringLoader(jsonString)
	schemaLoader := LoadRulesetSchema()

	// check blob is a valid contract, before creating ruleset.
	res, err := gojsonschema.Validate(schemaLoader, jsonLoader)
	if err != nil {
		return nil, err
	}

	if !res.Valid() {
		var buf strings.Builder
		for _, e := range res.Errors() {
			buf.WriteString(fmt.Sprintf("%s (line),", e.Description()))
		}

		return nil, fmt.Errorf("rules not valid: %s", buf.String())
	}

	// unmarshal JSON into new RuleSet
	rs := &RuleSet{}
	err = json.Unmarshal(jsonData, rs)

	// raw rules are unpacked, lets copy them over

	rs.Rules = make(map[string]*model.Rule)
	for k, v := range rs.RuleDefinitions {
		if b, ok := v.(map[string]interface{}); ok {
			var rule model.Rule
			mapstructure.Decode(b, &rule)
			rs.Rules[k] = &rule
		}

		if b, ok := v.(model.Rule); ok {
			rs.Rules[k] = &b
		}
	}

	if err != nil {
		return nil, err
	}

	// save our loaded schema for later.
	rs.schemaLoader = schemaLoader
	return rs, nil
}

// LoadRulesetSchema creates a new JSON Schema loader for the RuleSet schema.
func LoadRulesetSchema() gojsonschema.JSONLoader {
	return gojsonschema.NewStringLoader(rulesetSchema)
}

// CreateRuleSetFromData will create a new RuleSet instance from either a JSON or YAML input
func CreateRuleSetFromData(data []byte) (*RuleSet, error) {
	d := data
	if !utils.IsJSON(string(d)) {
		j, err := utils.ConvertYAMLtoJSON(data)
		if err != nil {
			return nil, err
		}
		d = j
	}
	return CreateRuleSetUsingJSON(d)
}
