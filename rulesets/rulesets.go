// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package rulesets

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/daveshanley/vacuum/model"
	"github.com/mitchellh/mapstructure"
	"github.com/pb33f/libopenapi/utils"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed schemas/ruleset.schema.json
var RulesetSchema string

//go:embed schemas/rule.schema.json
var RuleSchema string

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
	InfoLicenseSPDX                      = "info-license-spdx"
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
	Oas3NoRefSiblings                    = "oas3-no-$ref-siblings"
	Oas3UnusedComponent                  = "oas3-unused-component"
	Oas2UnusedDefinition                 = "oas2-unused-definition"
	Oas2APIHost                          = "oas2-api-host"
	Oas2APISchemes                       = "oas2-api-schemes"
	Oas2Discriminator                    = "oas2-discriminator"
	Oas2HostNotExample                   = "oas2-host-not-example"
	Oas3HostNotExample                   = "oas3-host-not-example"
	Oas2HostTrailingSlash                = "oas2-host-trailing-slash"
	Oas3HostTrailingSlash                = "oas3-server-trailing-slash"
	Oas2ParameterDescription             = "oas2-parameter-description"
	Oas3ParameterDescription             = "oas3-parameter-description"
	Oas3OperationSecurityDefined         = "oas3-operation-security-defined"
	Oas2OperationSecurityDefined         = "oas2-operation-security-defined"
	Oas3ValidSchemaExample               = "oas3-valid-schema-example"
	Oas3ExampleMissingCheck              = "oas3-missing-example"
	Oas3ExampleExternalCheck             = "oas3-example-external-check"
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
	OasSchemaCheck                       = "oas-schema-check"
	PathItemReferences                   = "path-item-refs"
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
	OwaspIntegerFormat                   = "owasp-integer-format"
	OwaspNoAdditionalProperties          = "owasp-no-additionalProperties"
	OwaspConstrainedAdditionalProperties = "owasp-constrained-additionalProperties"
	OwaspSecurityHostsHttpsOAS3          = "owasp-security-hosts-https-oas3"
	PostResponseSuccess                  = "post-response-success"
	NoRequestBody                        = "no-request-body"
	VacuumOpenAPI                        = "vacuum:oas"
	SpectralOpenAPI                      = "spectral:oas"
	SpectralOwasp                        = "spectral:owasp"
	VacuumOwasp                          = "vacuum:owasp"
	VacuumAllRulesets                    = "vacuum:all"  // Combined OpenAPI + OWASP rules
	VacuumRecommended                    = "recommended"
	VacuumAll                            = "all"
	VacuumOff                            = "off"
	SpectralRecommended                  = VacuumRecommended
	SpectralAll                          = VacuumAll
	SpectralOff                          = SpectralAll
)

type ruleSetsModel struct {
	openAPIRuleSet *RuleSet
	logger         *slog.Logger
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
	
	// GenerateRuleSetFromSuppliedRuleSetWithHTTPClient will generate a ready to run ruleset based on a supplied configuration. This
	// will look for any extensions and apply all rules turned on, turned off and any custom rules.
	// It accepts an HTTP client for downloading remote rulesets with certificate authentication.
	GenerateRuleSetFromSuppliedRuleSetWithHTTPClient(config *RuleSet, httpClient *http.Client) *RuleSet
}

//var rulesetsSingleton *ruleSetsModel

func BuildDefaultRuleSets() RuleSets {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
	return BuildDefaultRuleSetsWithLogger(logger)
}

func BuildDefaultRuleSetsWithLogger(logger *slog.Logger) RuleSets {
	rulesetsSingleton := &ruleSetsModel{
		openAPIRuleSet: GenerateDefaultOpenAPIRuleSet(),
		logger:         logger,
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
	return rsm.GenerateRuleSetFromSuppliedRuleSetWithHTTPClient(ruleset, nil)
}

func (rsm ruleSetsModel) GenerateRuleSetFromSuppliedRuleSetWithHTTPClient(ruleset *RuleSet, httpClient *http.Client) *RuleSet {

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
	if extends[VacuumOpenAPI] == VacuumRecommended || extends[VacuumOpenAPI] == VacuumOpenAPI {
		rs = rsm.GenerateOpenAPIRecommendedRuleSet()
	}

	// default and explicitly recommended
	if extends[SpectralOpenAPI] == VacuumRecommended || extends[SpectralOpenAPI] == SpectralOpenAPI {
		rs = rsm.GenerateOpenAPIRecommendedRuleSet()
	}

	// all rules
	if extends[SpectralOpenAPI] == VacuumAll || extends[VacuumOpenAPI] == VacuumAll {
		rs = rsm.openAPIRuleSet
	}

	// vacuum:all - combines both OpenAPI and OWASP rules
	if extends[VacuumAllRulesets] == VacuumAll || extends[VacuumAllRulesets] == VacuumAllRulesets {
		// Start with OpenAPI rules
		rs = rsm.openAPIRuleSet
		// Add all OWASP rules
		for ruleName, rule := range GetAllOWASPRules() {
			rs.Rules[ruleName] = rule
		}
		rs.DocumentationURI = "https://quobix.com/vacuum/rulesets/all-combined"
		rs.Description = "All OpenAPI and OWASP rules combined"
	}

	// vacuum:all with off - start with empty ruleset
	if extends[VacuumAllRulesets] == VacuumOff {
		if rs.DocumentationURI == "" {
			rs.DocumentationURI = "https://quobix.com/vacuum/rulesets/no-rules"
		}
		rs.Rules = make(map[string]*model.Rule)
		rs.Description = fmt.Sprintf("All disabled ruleset, processing %d supplied rules", len(rs.RuleDefinitions))
	}

	// no rules!
	if extends[SpectralOpenAPI] == VacuumOff || extends[VacuumOpenAPI] == VacuumOff {
		if rs.DocumentationURI == "" {
			rs.DocumentationURI = "https://quobix.com/vacuum/rulesets/no-rules"
		}
		rs.Rules = make(map[string]*model.Rule)
		rs.Description = fmt.Sprintf("All disabled ruleset, processing %d supplied rules", len(rs.RuleDefinitions))
	}

	if ruleset.DocumentationURI == "" {
		ruleset.DocumentationURI = "https://quobix.com/vacuum/rulesets/understanding"
	}

	// make sure the map is never nil.
	if rs.Rules == nil {
		rs.Rules = make(map[string]*model.Rule)
	}

	// owasp rules with spectral and vacuum namespace
	if extends[SpectralOwasp] == VacuumAll || extends[VacuumOwasp] == VacuumAll {
		for ruleName, rule := range GetAllOWASPRules() {
			rs.Rules[ruleName] = rule
		}
	}

	// owasp rules with spectral and vacuum namespace (recommended)
	if extends[SpectralOwasp] == VacuumRecommended || extends[VacuumOwasp] == VacuumRecommended {
		for ruleName, rule := range GetRecommendedOWASPRules() {
			rs.Rules[ruleName] = rule
		}
	}

	// add definitions.
	rs.RuleDefinitions = ruleset.RuleDefinitions

	if rs.RuleDefinitions == nil {
		rs.RuleDefinitions = make(map[string]any)
	}

	// download remote rulesets
	if CheckForRemoteExtends(extends) || CheckForLocalExtends(extends) {

		doneChan := make(chan bool)

		// give it a fair wait, 5 seconds is long enough.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		total := 0

		go func() {

			for k := range extends {
				if strings.HasPrefix(k, "http") ||
					filepath.Ext(k) == ".yml" ||
					filepath.Ext(k) == ".yaml" ||
					filepath.Ext(k) == ".json" {
					total++
					remote := false
					if strings.HasPrefix(k, "http") {
						remote = true
					}
					SniffOutAllExternalRules(ctx, &rsm, k, nil, rs, remote, httpClient)
				}
			}
			doneChan <- true
		}()

		select {
		case <-ctx.Done():
			rsm.logger.Error("external ruleset fetch timed out after 5 seconds")
			break
		case <-doneChan:
			break
		}

	}

	// now all the base rules are in, let's run through the raw definitions and decide
	// what we need to add, enable, disable, replace or change severity on.
	rs.mutex.Lock()
	for k, v := range rs.RuleDefinitions {

		// let's try to cast to a string first (enable/disable/severity)
		if evalStr, ok := v.(string); ok {

			// let's check to see if this rule exists
			if rs.Rules[k] == nil {

				rsm.logger.Warn("Rule does not exist, ignoring it", "rule", k)

				// we don't know anything about this rule, so skip it.
				continue
			}

			switch evalStr {
			case model.SeverityError, model.SeverityWarn, model.SeverityInfo, model.SeverityHint:
				rs.Rules[k].Severity = evalStr
			case VacuumOff:
				delete(rs.Rules, k) // remove it completely
			}
		}

		// let's try to cast to a bool, this means we want to enable a rule.
		// otherwise it means delete it
		if eval, ok := v.(bool); ok {
			if eval {
				// First check if it's in the OpenAPI ruleset
				if rsm.openAPIRuleSet.Rules[k] != nil {
					rs.Rules[k] = rsm.openAPIRuleSet.Rules[k]
				} else {
					// Check if it's an OWASP rule when vacuum:all is used
					if extends[VacuumAllRulesets] == VacuumOff || extends[VacuumAllRulesets] == VacuumAll || extends[VacuumAllRulesets] == VacuumAllRulesets {
						allOWASPRules := GetAllOWASPRules()
						if allOWASPRules[k] != nil {
							rs.Rules[k] = allOWASPRules[k]
						} else {
							rsm.logger.Warn("Rule does not exist, ignoring it", "rule", k)
							continue
						}
					} else {
						rsm.logger.Warn("Rule does not exist, ignoring it", "rule", k)
						continue
					}
				}
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
				rsm.logger.Error("Unable to decode rule", "error", dErr.Error())
			}
			dErr = mapstructure.Decode(newRule["category"], &rc)
			if dErr != nil {
				rsm.logger.Error("Unable to decode rule category", "error", dErr.Error())
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

			if nr.RuleCategory == nil && rs.Rules[k].RuleCategory != nil {
				nr.RuleCategory = rs.Rules[k].RuleCategory
			}

			// default new rule to be resolved if not supplied.
			if newRule["resolved"] == nil {
				nr.Resolved = true
			}

			rs.Rules[k] = &nr
		}
	}
	rs.mutex.Unlock()
	return rs
}

// CreateRuleSetFromRuleMap creates a RuleSet from a map of rules. Built-in rules can can be exposed by using
// the GetAllBuiltInRules() function.
func CreateRuleSetFromRuleMap(rules map[string]*model.Rule) *RuleSet {
	rs := &RuleSet{
		DocumentationURI: "https://quobix.com/vacuum/rulesets/understanding",
		Formats:          []string{"oas2", "oas3"},
		Extends:          map[string]string{VacuumOpenAPI: VacuumOff},
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
	rules[InfoLicenseSPDX] = GetInfoLicenseSPDXRule()
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
	rules[Oas3NoRefSiblings] = GetOAS3NoRefSiblingsRule()
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
	rules[NoAmbiguousPathsRule] = NoAmbiguousPaths()
	rules[NoVerbsInPath] = GetNoVerbsInPathRule()
	rules[PathsKebabCase] = GetPathsKebabCaseRule()
	rules[OperationErrorResponse] = GetOperationErrorResponseRule()
	rules[Oas2Schema] = GetOAS2SchemaRule()
	rules[Oas3Schema] = GetOAS3SchemaRule()
	rules[Oas3ValidSchemaExample] = GetOAS3ExamplesRule()
	rules[Oas3ExampleMissingCheck] = GetOAS3ExamplesMissingRule()
	rules[Oas3ExampleExternalCheck] = GetOAS3ExamplesExternalCheck()
	rules[OasSchemaCheck] = GetSchemaTypeCheckRule()
	rules[PostResponseSuccess] = GetPostSuccessResponseRule()
	rules[NoRequestBody] = GetNoRequestBodyRule()
	rules[PathItemReferences] = GetPathItemReferencesRule()

	// dead.
	//rules[Oas2ValidSchemaExample] = GetOAS2ExamplesRule()

	return rules
}

// GetAllOWASPRules returns a map of all the OWASP rules available, ready to be used in a RuleSet.
func GetAllOWASPRules() map[string]*model.Rule {
	rules := make(map[string]*model.Rule)

	rules[OwaspProtectionGlobalUnsafe] = GetOWASPProtectionGlobalUnsafeRule()
	rules[OwaspProtectionGlobalUnsafeStrict] = GetOWASPProtectionGlobalUnsafeStrictRule()
	rules[OwaspProtectionGlobalSafe] = GetOWASPProtectionGlobalSafeRule()
	rules[OwaspDefineErrorResponses401] = GetOWASPDefineErrorResponses401Rule()
	rules[OwaspDefineErrorResponses500] = GetOWASPDefineErrorResponses500Rule()
	rules[OwaspRateLimit] = GetOWASPRateLimitRule()
	rules[OwaspRateLimitRetryAfter] = GetOWASPRateLimitRetryAfterRule()
	rules[OwaspDefineErrorResponses429] = GetOWASPDefineErrorResponses429Rule()
	rules[OwaspArrayLimit] = GetOWASPArrayLimitRule()
	rules[OwaspJWTBestPractices] = GetOWASPJWTBestPracticesRule()
	rules[OwaspAuthInsecureSchemes] = GetOWASPAuthInsecureSchemesRule()
	rules[OwaspNoNumericIDs] = GetOWASPNoNumericIDsRule()
	rules[OwaspNoHttpBasic] = GetOWASPNoHttpBasicRule()
	rules[OwaspDefineErrorValidation] = GetOWASPDefineErrorValidationRule()
	rules[OwaspNoAPIKeysInURL] = GetOWASPNoAPIKeysInURLRule()
	rules[OwaspNoCredentialsInURL] = GetOWASPNoCredentialsInURLRule()
	rules[OwaspStringLimit] = GetOWASPStringLimitRule()
	rules[OwaspStringRestricted] = GetOWASPStringRestrictedRule()
	rules[OwaspIntegerFormat] = GetOWASPIntegerFormatRule()
	rules[OwaspIntegerLimit] = GetOWASPIntegerLimitRule()
	rules[OwaspNoAdditionalProperties] = GetOWASPNoAdditionalPropertiesRule()
	rules[OwaspConstrainedAdditionalProperties] = GetOWASPConstrainedAdditionalPropertiesRule()
	rules[OwaspSecurityHostsHttpsOAS3] = GetOWASPSecurityHostsHttpsOAS3Rule()

	return rules
}

// GetRecommendedOWASPRules returns a map of all the OWASP rules available, ready to be used in a RuleSet.
func GetRecommendedOWASPRules() map[string]*model.Rule {
	return GetAllOWASPRules() // change if we need to customize this in the future.
}

// GenerateDefaultOpenAPIRuleSet generates a default ruleset for OpenAPI. All the built-in rules, ready to go.
func GenerateDefaultOpenAPIRuleSet() *RuleSet {
	set := &RuleSet{
		DocumentationURI: "https://quobix.com/vacuum/rulesets/all",
		Rules:            GetAllBuiltInRules(),
		Description:      "Every single rule that is built-in to vacuum. The full monty",
	}
	return set
}

// GenerateOWASPOpenAPIRuleSet generates our OWASP ruleset for OpenAPI. Hard mode engage!
func GenerateOWASPOpenAPIRuleSet() *RuleSet {
	set := &RuleSet{
		DocumentationURI: "https://quobix.com/vacuum/rulesets/owasp",
		Rules:            GetAllOWASPRules(),
		Description:      "All OWASP Rules, or 'hard mode' as we call it.",
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
	mutex            sync.Mutex
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

	// schema needs to be parsed first
	var parsed map[string]interface{}
	_ = json.Unmarshal([]byte(RulesetSchema), &parsed)
	_ = compiler.AddResource("schema.json", parsed)
	jsch, _ := compiler.Compile("schema.json")

	var data map[string]interface{}
	_ = json.Unmarshal(jsonData, &data)

	scErrs := jsch.Validate(data)

	if scErrs != nil {
		jk := scErrs.(*jsonschema.ValidationError)
		var buf strings.Builder
		// flatten the validationErrors
		schFlatErrs := jk.BasicOutput().Errors
		for q := range schFlatErrs {
			buf.WriteString(schFlatErrs[q].Error.Kind.LocalizedString(message.NewPrinter(language.Tag{})))
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
			// check if rule has category.
			if b["category"] != nil {
				var cat model.RuleCategory
				dErr = mapstructure.Decode(b["category"], &cat)
				if dErr == nil {
					rule.RuleCategory = &cat
				}
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
