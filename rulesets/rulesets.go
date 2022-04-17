// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package rulesets

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"sync"
)

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
)

var AllOperationsPath = fmt.Sprintf("%s%s", allPaths, allOperations)

type ruleSetsModel struct {
	openAPIRuleSet *model.RuleSet
}

// RuleSets is used to generate default RuleSets built into vacuum
type RuleSets interface {

	// GenerateOpenAPIDefaultRuleSet generates a ready to run pointer to a model.RuleSet containing all
	// OpenAPI rules supported by vacuum. Passing all these rules would be considered a good quality specification.
	GenerateOpenAPIDefaultRuleSet() *model.RuleSet
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

func (rsm ruleSetsModel) GenerateOpenAPIDefaultRuleSet() *model.RuleSet {
	return rsm.openAPIRuleSet
}

func generateDefaultOpenAPIRuleSet() *model.RuleSet {

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

	set := &model.RuleSet{
		DocumentationURI: "https://quobix.com/vacuum/rules/openapi",
		Rules:            rules,
	}

	return set

}
