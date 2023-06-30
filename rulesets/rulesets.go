// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package rulesets

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/mitchellh/mapstructure"
	"github.com/pb33f/libopenapi/utils"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"go.uber.org/zap"
)

//go:embed schemas/ruleset.schema.json
var rulesetSchema string

const (
	Style                                = "style"
	Validation                           = "validation"
	NoVerbsInPath                        = "no-http-verbs-in-path"
	PathsKebabCase                       = "paths-kebab-case"
	NoAmbiguousPathsRule                 = "no-ambiguous-paths"
	OperationErrorResponse               = "operation-4xx-response"
	OperationSuccessResponse             = "operation-success-response"
	OperationOperationIdUnique           = "operation-operationId-unique"
	OperationOperationId                 = "operation-operationId"
	OperationParameters                  = "operation-parameters"
	OperationSingularTag                 = "operation-singular-tag"
	OperationTagDefined                  = "operation-tag-defined"
	PathParamsRule                       = "path-params"
	ContactProperties                    = "contact-properties"
	InfoContact                          = "info-contact"
	InfoDescription                      = "info-description"
	InfoLicense                          = "info-license"
	LicenseUrl                           = "license-url"
	OpenAPITagsAlphabetical              = "openapi-tags-alphabetical"
	OpenAPITags                          = "openapi-tags"
	OperationTags                        = "operation-tags"
	OperationDescription                 = "operation-description"
	ComponentDescription                 = "component-description"
	OperationOperationIdValidInUrl       = "operation-operationId-valid-in-url"
	PathDeclarationsMustExist            = "path-declarations-must-exist"
	PathKeysNoTrailingSlash              = "path-keys-no-trailing-slash"
	PathNotIncludeQuery                  = "path-not-include-query"
	TagDescription                       = "tag-description"
	NoRefSiblings                        = "no-$ref-siblings"
	Oas3UnusedComponent                  = "oas3-unused-component"
	Oas2UnusedDefinition                 = "oas2-unused-definition"
	Oas2APIHost                          = "oas2-api-host"
	Oas2APISchemes                       = "oas2-api-schemes"
	Oas2Discriminator                    = "oas2-discriminator"
	Oas2HostNotExample                   = "oas2-host-not-example"
	Oas3HostNotExample                   = "oas3-host-not-example.com"
	Oas2HostTrailingSlash                = "oas2-host-trailing-slash"
	Oas3HostTrailingSlash                = "oas3-host-trailing-slash"
	Oas2ParameterDescription             = "oas2-parameter-description"
	Oas3ParameterDescription             = "oas3-parameter-description"
	Oas3OperationSecurityDefined         = "oas3-operation-security-defined"
	Oas2OperationSecurityDefined         = "oas2-operation-security-defined"
	Oas3ValidSchemaExample               = "oas3-valid-schema-example"
	Oas2ValidSchemaExample               = "oas2-valid-schema-example"
	TypedEnum                            = "typed-enum"
	DuplicatedEntryInEnum                = "duplicated-entry-in-enum"
	NoEvalInMarkdown                     = "no-eval-in-markdown"
	NoScriptTagsInMarkdown               = "no-script-tags-in-markdown"
	DescriptionDuplication               = "description-duplication"
	Oas3APIServers                       = "oas3-api-servers"
	Oas2OperationFormDataConsumeCheck    = "oas2-operation-formData-consume-check"
	Oas2AnyOf                            = "oas2-anyOf"
	Oas2OneOf                            = "oas2-oneOf"
	Oas2Schema                           = "oas2-schema"
	Oas3Schema                           = "oas3-schema"
	OwaspNoNumericIDs                    = "owasp-no-numeric-ids"
	OwaspNoHttpBasic                     = "owasp-no-http-basic"
	OwaspNoAPIKeysInURL                  = "owasp-no-api-keys-in-url"
	OwaspNoCredentialsInURL              = "owasp-no-credentials-in-url"
	OwaspAuthInsecureSchemes             = "owasp-auth-insecure-schemes"
	OwaspJWTBestPractices                = "owasp-jwt-best-practices"
	OwaspProtectionGlobalUnsafe          = "owasp-protection-global-unsafe"
	OwaspProtectionGlobalUnsafeStrict    = "owasp-protection-global-unsafe-strict"
	OwaspProtectionGlobalSafe            = "owasp-protection-global-safe"
	OwaspDefineErrorValidation           = "owasp-define-error-validation"
	OwaspDefineErrorResponses401         = "owasp-define-error-responses-401"
	OwaspDefineErrorResponses500         = "owasp-define-error-responses-500"
	OwaspRateLimit                       = "owasp-rate-limit"
	OwaspRateLimitRetryAfter             = "owasp-rate-limit-retry-after"
	OwaspDefineErrorResponses429         = "owasp-define-error-responses-429"
	OwaspArrayLimit                      = "owasp-array-limit"
	OwaspStringLimit                     = "owasp-string-limit"
	OwaspStringRestricted                = "owasp-string-restricted"
	OwaspIntegerLimit                    = "owasp-integer-limit"
	OwaspIntegerLimitLegacy              = "owasp-integer-limit-legacy"
	OwaspIntegerFormat                   = "owasp-integer-format"
	OwaspNoAdditionalProperties          = "owasp-no-additionalProperties"
	OwaspConstrainedAdditionalProperties = "owasp-constrained-additionalProperties"
	OwaspSecurityHostsHttpsOAS2          = "owasp-security-hosts-https-oas2"
	OwaspSecurityHostsHttpsOAS3          = "owasp-security-hosts-https-oas3"
	SpectralOpenAPI                      = "spectral:oas"
	SpectralOwasp                        = "spectral:owasp"
	VacuumOwasp                          = "vacuum:owasp"
	SpectralRecommended                  = "recommended"
	SpectralAll                          = "all"
	SpectralOff                          = "off"
)

var log *zap.Logger

//var log *zap.SugaredLogger

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

func BuildDefaultRuleSets() RuleSets {
	log = zap.NewExample()

	rulesetsSingleton = &ruleSetsModel{
		openAPIRuleSet: GenerateDefaultOpenAPIRuleSet(),
	}
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
	modifiedRS.DocumentationURI = "https://quobix.com/vacuum/rulesets/recommended"
	modifiedRS.Description = "Recommended rules for a high quality specification."
	return &modifiedRS
}

func (rsm ruleSetsModel) GenerateRuleSetFromSuppliedRuleSet(ruleset *RuleSet) *RuleSet {

	extends := ruleset.GetExtendsValue()

	rs := &RuleSet{
		DocumentationURI: ruleset.DocumentationURI,
		Formats:          ruleset.Formats,
		Extends:          ruleset.Extends,
		Description:      ruleset.Description,
		RuleDefinitions:  ruleset.RuleDefinitions,
		Rules:            ruleset.Rules,
	}

	// default and explicitly recommended
	if extends[SpectralOpenAPI] == SpectralRecommended || extends[SpectralOpenAPI] == SpectralOpenAPI {
		rs = rsm.GenerateOpenAPIRecommendedRuleSet()
	}

	// all rules
	if extends[SpectralOpenAPI] == SpectralAll {
		rs = rsm.openAPIRuleSet
	}

	// no rules!
	if extends[SpectralOpenAPI] == SpectralOff {
		if rs.DocumentationURI == "" {
			rs.DocumentationURI = "https://quobix.com/vacuum/rulesets/no-rules"
		}
		rs.Rules = make(map[string]*model.Rule)
		rs.Description = fmt.Sprintf("All disabled ruleset, processing %d supplied rules", len(rs.RuleDefinitions))
	}

	// add owasp rules
	if extends[SpectralOwasp] == SpectralOwasp {
		for ruleName, rule := range GetAllOWASPRules() {
			rs.Rules[ruleName] = rule
		}
	}

	if ruleset.DocumentationURI == "" {
		ruleset.DocumentationURI = "https://quobix.com/vacuum/rulesets/understanding"
	}

	// make sure the map is never nil.
	if rs.Rules == nil {
		rs.Rules = make(map[string]*model.Rule)
	}

	// owasp rules with spectral and vacuum namespace
	if extends[SpectralOwasp] == SpectralAll || extends[VacuumOwasp] == SpectralAll {
		for ruleName, rule := range GetAllOWASPRules() {
			rs.Rules[ruleName] = rule
		}
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

				log.Warn("Rule does not exist, ignoring it", zap.String("rule", k))

				// we don't know anything about this rule, so skip it.
				continue
			}

			switch evalStr {
			case model.SeverityError, model.SeverityWarn, model.SeverityInfo, model.SeverityHint:
				rs.Rules[k].Severity = evalStr
			case SpectralOff:
				delete(rs.Rules, k) // remove it completely
			}
		}

		// let's try to cast to a bool, this means we want to enable a rule.
		// otherwise it means delete it
		if eval, ok := v.(bool); ok {
			if eval {
				if rsm.openAPIRuleSet.Rules[k] == nil {
					log.Warn("Rule does not exist, ignoring it", zap.String("rule", k))
					continue
				}
				rs.Rules[k] = rsm.openAPIRuleSet.Rules[k]
			} else {
				delete(rs.Rules, k) // remove it completely
			}
		}

		// let's try to cast to a model.Rule, this means we want to add a new rule.
		if newRule, ok := v.(map[string]interface{}); ok {

			// decode into a rule, we don't need to check for an error here, if the supplied rule
			// breaks the schema, it will have already failed, and we will have caught that message.
			var nr model.Rule
			var rc model.RuleCategory

			dErr := mapstructure.Decode(newRule, &nr)
			if dErr != nil {
				log.Error("Unable to decode rule", zap.String("error", dErr.Error()))
			}
			dErr = mapstructure.Decode(newRule["category"], &rc)
			if dErr != nil {
				log.Error("Unable to decode rule category", zap.String("error", dErr.Error()))
			}

			// add to validation category if it's not supplied
			if rc.Id == "" {
				nr.RuleCategory = model.RuleCategories[model.CategoryValidation]
				nr.Id = k
			} else {
				if model.RuleCategories[rc.Id] != nil {
					nr.RuleCategory = model.RuleCategories[rc.Id]
				}
			}

			// default new rule to be resolved if not supplied.
			if newRule["resolved"] == nil {
				nr.Resolved = true
			}

			rs.Rules[k] = &nr
		}
	}
	return rs
}

// CreateRuleSetFromRuleMap creates a RuleSet from a map of rules. Built-in rules can can be exposed by using
// the GetAllBuiltInRules() function.
func CreateRuleSetFromRuleMap(rules map[string]*model.Rule) *RuleSet {
	rs := &RuleSet{
		DocumentationURI: "https://quobix.com/vacuum/rulesets/understanding",
		Formats:          []string{"oas2", "oas3"},
		Extends:          map[string]string{SpectralOpenAPI: SpectralOff},
		Description:      fmt.Sprintf("a custom ruleset composed of %d rules", len(rules)),
		RuleDefinitions:  make(map[string]interface{}),
		Rules:            rules,
	}
	return rs
}

// GetAllBuiltInRules returns a map of all the built-in rules available, ready to be used in a RuleSet.
func GetAllBuiltInRules() map[string]*model.Rule {
	rules := make(map[string]*model.Rule)
	rules[OperationSuccessResponse] = GetOperationSuccessResponseRule()
	rules[OperationOperationIdUnique] = GetOperationIdUniqueRule()
	rules[OperationOperationId] = GetOperationIdRule()
	rules[OperationParameters] = GetOperationParametersRule()
	rules[OperationSingularTag] = GetOperationSingleTagRule()
	rules[OperationTagDefined] = GetGlobalOperationTagsRule()
	rules[PathParamsRule] = GetPathParamsRule()
	rules[ContactProperties] = GetContactPropertiesRule()
	rules[InfoContact] = GetInfoContactRule()
	rules[InfoDescription] = GetInfoDescriptionRule()
	rules[InfoLicense] = GetInfoLicenseRule()
	rules[LicenseUrl] = GetInfoLicenseUrlRule()
	rules[OpenAPITagsAlphabetical] = GetOpenApiTagsAlphabeticalRule()
	rules[OpenAPITags] = GetOpenApiTagsRule()
	rules[OperationTags] = GetOperationTagsRule()
	rules[OperationDescription] = GetOperationDescriptionRule()
	rules[ComponentDescription] = GetComponentDescriptionsRule()
	rules[OperationOperationIdValidInUrl] = GetOperationIdValidInUrlRule()
	rules[PathDeclarationsMustExist] = GetPathDeclarationsMustExistRule()
	rules[PathKeysNoTrailingSlash] = GetPathNoTrailingSlashRule()
	rules[PathNotIncludeQuery] = GetPathNotIncludeQueryRule()
	rules[TagDescription] = GetTagDescriptionRequiredRule()
	rules[NoRefSiblings] = GetNoRefSiblingsRule()
	rules[Oas3UnusedComponent] = GetOAS3UnusedComponentRule()
	rules[Oas2UnusedDefinition] = GetOAS2UnusedComponentRule()
	rules[Oas2APIHost] = GetOAS2APIHostRule()
	rules[Oas2APISchemes] = GetOAS2APISchemesRule()
	rules[Oas2Discriminator] = GetOAS2DiscriminatorRule()
	rules[Oas2HostNotExample] = GetOAS2HostNotExampleRule()
	rules[Oas3HostNotExample] = GetOAS3HostNotExampleRule()
	rules[Oas2HostTrailingSlash] = GetOAS2HostTrailingSlashRule()
	rules[Oas3HostTrailingSlash] = GetOAS3HostTrailingSlashRule()
	rules[Oas2ParameterDescription] = GetOAS2ParameterDescriptionRule()
	rules[Oas3ParameterDescription] = GetOAS3ParameterDescriptionRule()
	rules[Oas3OperationSecurityDefined] = GetOAS3SecurityDefinedRule()
	rules[Oas2OperationSecurityDefined] = GetOAS2SecurityDefinedRule()
	rules[TypedEnum] = GetTypedEnumRule()
	rules[DuplicatedEntryInEnum] = GetDuplicatedEntryInEnumRule()
	rules[NoEvalInMarkdown] = GetNoEvalInMarkdownRule()
	rules[NoScriptTagsInMarkdown] = GetNoScriptTagsInMarkdownRule()
	rules[DescriptionDuplication] = GetDescriptionDuplicationRule()
	rules[Oas3APIServers] = GetAPIServersRule()
	rules[Oas2OperationFormDataConsumeCheck] = GetOAS2FormDataConsumesRule()
	rules[Oas2AnyOf] = GetOAS2PolymorphicAnyOfRule()
	rules[Oas2OneOf] = GetOAS2PolymorphicOneOfRule()
	rules[Oas3ValidSchemaExample] = GetOAS3ExamplesRule()
	rules[Oas2ValidSchemaExample] = GetOAS2ExamplesRule()
	rules[NoAmbiguousPathsRule] = NoAmbiguousPaths()
	rules[NoVerbsInPath] = GetNoVerbsInPathRule()
	rules[PathsKebabCase] = GetPathsKebabCaseRule()
	rules[OperationErrorResponse] = GetOperationErrorResponseRule()
	rules[Oas2Schema] = GetOAS2SchemaRule()
	rules[Oas3Schema] = GetOAS3SchemaRule()
	return rules
}

// GetAllOWASPRules returns a map of all the OWASP rules available, ready to be used in a RuleSet.
func GetAllOWASPRules() map[string]*model.Rule {
	rules := make(map[string]*model.Rule)

	rules[OwaspNoNumericIDs] = GetOWASPNoNumericIDsRule()
	rules[OwaspNoHttpBasic] = GetOWASPNoHttpBasicRule()
	rules[OwaspNoAPIKeysInURL] = GetOWASPNoAPIKeysInURLRule()
	rules[OwaspNoCredentialsInURL] = GetOWASPNoCredentialsInURLRule()
	rules[OwaspAuthInsecureSchemes] = GetOWASPAuthInsecureSchemesRule()
	rules[OwaspJWTBestPractices] = GetOWASPJWTBestPracticesRule()
	rules[OwaspProtectionGlobalUnsafe] = GetOWASPProtectionGlobalUnsafeRule()
	rules[OwaspProtectionGlobalUnsafeStrict] = GetOWASPProtectionGlobalUnsafeStrictRule()
	rules[OwaspProtectionGlobalSafe] = GetOWASPProtectionGlobalSafeRule()
	rules[OwaspDefineErrorValidation] = GetOWASPDefineErrorValidationRule()
	rules[OwaspDefineErrorResponses401] = GetOWASPDefineErrorResponses401Rule()
	rules[OwaspDefineErrorResponses500] = GetOWASPDefineErrorResponses500Rule()
	rules[OwaspRateLimit] = GetOWASPRateLimitRule()
	rules[OwaspRateLimitRetryAfter] = GetOWASPRateLimitRetryAfterRule()
	rules[OwaspDefineErrorResponses429] = GetOWASPDefineErrorResponses429Rule()
	rules[OwaspArrayLimit] = GetOWASPArrayLimitRule()
	rules[OwaspStringLimit] = GetOWASPStringLimitRule()
	rules[OwaspStringRestricted] = GetOWASPStringRestrictedRule()
	rules[OwaspIntegerLimit] = GetOWASPIntegerLimitRule()
	rules[OwaspIntegerLimitLegacy] = GetOWASPIntegerLimitLegacyRule()
	rules[OwaspIntegerFormat] = GetOWASPIntegerFormatRule()
	rules[OwaspNoAdditionalProperties] = GetOWASPNoAdditionalPropertiesRule()
	rules[OwaspConstrainedAdditionalProperties] = GetOWASPConstrainedAdditionalPropertiesRule()
	rules[OwaspSecurityHostsHttpsOAS2] = GetOWASPSecurityHostsHttpsOAS2Rule()
	rules[OwaspSecurityHostsHttpsOAS3] = GetOWASPSecurityHostsHttpsOAS3Rule()
	return rules
}

// GenerateDefaultOpenAPIRuleSet generates a default ruleset for OpenAPI. All the built in rules, ready to go.
func GenerateDefaultOpenAPIRuleSet() *RuleSet {
	set := &RuleSet{
		DocumentationURI: "https://quobix.com/vacuum/rulesets/all",
		Rules:            GetAllBuiltInRules(),
		Description:      "Every single rule that is built-in to vacuum. The full monty",
	}
	return set
}

// RuleSet represents a collection of Rule definitions.
type RuleSet struct {
	Description      string                 `json:"description,omitempty" yaml:"description,omitempty"`
	DocumentationURI string                 `json:"documentationUrl,omitempty" yaml:"documentationUrl,omitempty"`
	Formats          []string               `json:"formats,omitempty" yaml:"formats,omitempty"`
	RuleDefinitions  map[string]interface{} `json:"rules" yaml:"rules"` // this can be either a string, or an entire rule (super annoying, stoplight).
	Rules            map[string]*model.Rule `json:"-" yaml:"-"`
	Extends          interface{}            `json:"extends,omitempty" yaml:"extends,omitempty"` // can be string or tuple (again... why stoplight?)
	extendsMeta      map[string]string
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

	compiler := jsonschema.NewCompiler()
	_ = compiler.AddResource("schema.json", strings.NewReader(rulesetSchema))
	jsch, _ := compiler.Compile("schema.json")

	var data map[string]interface{}
	_ = json.Unmarshal(jsonData, &data)

	// 4. validate the object against the schema
	scErrs := jsch.Validate(data)

	if scErrs != nil {
		jk := scErrs.(*jsonschema.ValidationError)
		var buf strings.Builder
		// flatten the validationErrors
		schFlatErrs := jk.BasicOutput().Errors
		for q := range schFlatErrs {
			buf.WriteString(schFlatErrs[q].Error)
			if q+1 < len(schFlatErrs) {
				buf.WriteString(", ")
			}
		}
		return nil, fmt.Errorf("rules not valid: %s", buf.String())
	}

	// unmarshal JSON into new RuleSet
	rs := &RuleSet{}
	uErr := json.Unmarshal(jsonData, rs)
	if uErr != nil {
		return nil, uErr
	}

	// raw rules are unpacked, lets copy them over
	rs.Rules = make(map[string]*model.Rule)
	for k, v := range rs.RuleDefinitions {
		if b, ok := v.(map[string]interface{}); ok {
			var rule model.Rule
			dErr := mapstructure.Decode(b, &rule)
			if dErr != nil {
				return nil, dErr
			}
			rs.Rules[k] = &rule
			rule.Resolved = true // default resolved.
		}

		if b, ok := v.(model.Rule); ok {
			rs.Rules[k] = &b
			b.Resolved = true // default resolved
		}
	}
	return rs, nil
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
