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
	error         = "error"
	info          = "info"
	hint          = "hint"
	style         = "style"
	validation    = "validation"
	allPaths      = "$.paths[*]"
	allOperations = "[?(@.get || @.post || @.put || @.patch || @.delete || @.trace || @.options || @.head)]"
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

	// add success response
	rules["operation-success-response"] = GetOperationSuccessResponseRule()

	// add unique operation ID rule
	rules["operation-operationId-unique"] = GetOperationIdUniqueRule()

	// add operation ID rule
	rules["operation-operationId"] = GetOperationIdRule()

	// add operation params rule
	rules["operation-parameters"] = GetOperationParametersRule()

	// add operation single tag rule
	rules["operation-singular-tag"] = GetOperationSingleTagRule()

	// add operation tag defined rule
	rules["operation-tag-defined"] = GetGlobalOperationTagsRule()

	// add operation tag defined rule
	rules["path-params"] = GetPathParamsRule()

	//// contact-properties
	rules["contact-properties"] = GetContactPropertiesRule()

	// info object: contains contact
	rules["info-contact"] = GetInfoContactRule()

	// info object: contains description
	rules["info-description"] = GetInfoDescriptionRule()

	// info object: contains a license
	rules["info-license"] = GetInfoLicenseRule()

	// info object: contains a license url
	rules["license-url"] = GetInfoLicenseUrlRule()

	// check tags are in alphabetical order
	rules["openapi-tags-alphabetical"] = GetOpenApiTagsAlphabeticalRule()

	// check tags exist correctly
	rules["openapi-tags"] = GetOpenApiTagsRule()

	// check operation tags exist correctly
	rules["operation-tags"] = GetOperationTagsRule()

	//check all operations have a description, and they match a set length.
	rules["operation-description"] = GetOperationDescriptionRule()

	// check all components have a description, and they match a set length.
	rules["component-description"] = GetComponentDescriptionsRule()

	// check operationId does not contain characters that would be invalid in a URL
	rules["operation-operationId-valid-in-url"] = GetOperationIdValidInUrlRule()

	// check paths don't have any empty declarations
	rules["path-declarations-must-exist"] = GetPathDeclarationsMustExistRule()

	// check paths don't end with a slash
	rules["path-keys-no-trailing-slash"] = GetPathNoTrailingSlashRule()

	// check paths don't contain a query string
	rules["path-not-include-query"] = GetPathNotIncludeQueryRule()

	// check tags have a description defined
	rules["tag-description"] = GetTagDescriptionRequiredRule()

	//add no $ref siblings
	rules["no-$ref-siblings"] = GetNoRefSiblingsRule()

	// add unused component rule
	rules["oas3-unused-component"] = GetOAS3UnusedComponentRule()
	rules["oas2-unused-definition"] = GetOAS2UnusedComponentRule()

	// oas2 check for host value
	rules["oas2-api-host"] = GetOAS2APIHostRule()

	// oas2 check for schemes
	rules["oas2-api-schemes"] = GetOAS2APISchemesRule()

	// oas2 check for discriminator
	rules["oas2-discriminator"] = GetOAS2DiscriminatorRule()

	// oas2 check for example.com being used
	rules["oas2-host-not-example"] = GetOAS2HostNotExampleRule()

	// oas3 check for example.com being used
	rules["oas3-host-not-example.com"] = GetOAS3HostNotExampleRule()

	// oas2 check host does not have a trailing slash
	rules["oas2-host-trailing-slash"] = GetOAS2HostTrailingSlashRule()

	// oas3 check host does not have a trailing slash
	rules["oas3-host-trailing-slash"] = GetOAS2HostTrailingSlashRule()

	// oas2 parameter description check
	rules["oas2-parameter-description"] = GetOAS2ParameterDescriptionRule()

	// oas3 parameter description check
	rules["oas3-parameter-description"] = GetOAS3ParameterDescriptionRule()

	// security for versions 2 and 3.
	rules["oas3-operation-security-defined"] = GetOAS3SecurityDefinedRule()
	rules["oas2-operation-security-defined"] = GetOAS2SecurityDefinedRule()

	// check all examples
	rules["oas3-valid-schema-example"] = GetOAS3ExamplesRule()

	// check all examples
	rules["oas2-valid-schema-example"] = GetOAS2ExamplesRule()

	// check enums respect specified types
	rules["typed-enum"] = GetTypedEnumRule()

	// check for duplication in enums
	rules["duplicated-entry-in-enum"] = GetDuplicatedEntryInEnumRule()

	// check no eval statements in markdown descriptions.
	rules["no-eval-in-markdown"] = GetNoEvalInMarkdownRule()

	// check no script statements in markdown descriptions.
	rules["no-script-tags-in-markdown"] = GetNoScriptTagsInMarkdownRule()

	// check for description and summary duplication
	rules["description-duplication"] = GetDescriptionDuplicationRule()

	// check for valid API server definitions
	rules["oas3-api-servers"] = GetAPIServersRule()

	// check for correct 'consumes' type used with parameters and in: formData
	rules["oas2-operation-formData-consume-check"] = GetOAS2FormDataConsumesRule()

	// check that no 'anyOf' polymorphism has been used in 2.0 spec.
	rules["oas2-anyOf"] = GetOAS2PolymorphicAnyOfRule()

	// check that no 'oneOf' polymorphism has been used in 2.0 spec.
	rules["oas2-oneOf"] = GetOAS2PolymorphicOneOfRule()

	// TODO: These schema rules are super slow, due to the nature of the library we're using.
	// so only run these rules on hard mode.

	// check that the schema is even valid if it's a swagger doc
	//rules["oas2-schema"] = GetOAS2SchemaRule()

	// check that the schema is even valid, if it's an OpenAPI doc.
	//rules["oas3-schema"] = GetOAS3SchemaRule()

	// TODO: need to map all spectral rules that don't map specifically to vacuum (like examples).
	//oas2-valid-schema-example
	//oas2-valid-media-example

	set := &model.RuleSet{
		DocumentationURI: "https://quobix.com/vacuum/rules/openapi",
		Rules:            rules,
	}

	return set

}
