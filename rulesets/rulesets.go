// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package rulesets

import (
    _ "embed"
    "encoding/json"
    "errors"
    "fmt"
    "github.com/daveshanley/vacuum/model"
    "github.com/mitchellh/mapstructure"
    "github.com/pb33f/libopenapi/utils"
    "github.com/santhosh-tekuri/jsonschema/v5"
    "go.uber.org/zap"
    "strings"
)

//go:embed schemas/ruleset.schema.json
var rulesetSchema string

const (
    style                             = "style"
    validation                        = "validation"
    noVerbsInPath                     = "no-http-verbs-in-path"
    pathsKebabCase                    = "paths-kebab-case"
    noAmbiguousPaths                  = "no-ambiguous-paths"
    operationErrorResponse            = "operation-4xx-response"
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
    oas3HostTrailingSlash             = "oas3-host-trailing-slash"
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
        openAPIRuleSet: generateDefaultOpenAPIRuleSet(),
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

    if ruleset.DocumentationURI == "" {
        ruleset.DocumentationURI = "https://quobix.com/vacuum/rulesets/understanding"
    }

    // make sure the map is never nil.
    if rs.Rules == nil {
        rs.Rules = make(map[string]*model.Rule)
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
    rules[oas3HostTrailingSlash] = GetOAS3HostTrailingSlashRule()
    rules[oas2ParameterDescription] = GetOAS2ParameterDescriptionRule()
    rules[oas3ParameterDescription] = GetOAS3ParameterDescriptionRule()
    rules[oas3OperationSecurityDefined] = GetOAS3SecurityDefinedRule()
    rules[oas2OperationSecurityDefined] = GetOAS2SecurityDefinedRule()
    rules[typedEnum] = GetTypedEnumRule()
    rules[duplicatedEntryInEnum] = GetDuplicatedEntryInEnumRule()
    rules[noEvalInMarkdown] = GetNoEvalInMarkdownRule()
    rules[noScriptTagsInMarkdown] = GetNoScriptTagsInMarkdownRule()
    rules[descriptionDuplication] = GetDescriptionDuplicationRule()
    rules[oas3APIServers] = GetAPIServersRule()
    rules[oas2OperationFormDataConsumeCheck] = GetOAS2FormDataConsumesRule()
    rules[oas2AnyOf] = GetOAS2PolymorphicAnyOfRule()
    rules[oas2OneOf] = GetOAS2PolymorphicOneOfRule()
    rules[oas3ValidSchemaExample] = GetOAS3ExamplesRule()
    rules[oas2ValidSchemaExample] = GetOAS2ExamplesRule()
    rules[noAmbiguousPaths] = NoAmbiguousPaths()
    rules[noVerbsInPath] = GetNoVerbsInPathRule()
    rules[pathsKebabCase] = GetPathsKebabCaseRule()
    rules[operationErrorResponse] = GetOperationErrorResponseRule()
    rules[oas2Schema] = GetOAS2SchemaRule()
    rules[oas3Schema] = GetOAS3SchemaRule()

    set := &RuleSet{
        DocumentationURI: "https://quobix.com/vacuum/rulesets/all",
        Rules:            rules,
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
            buf.WriteString(fmt.Sprintf("%s", schFlatErrs[q].Error))
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
