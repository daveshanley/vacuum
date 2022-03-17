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

	///*
	// add success response
	rules["operation-success-response"] = GetOperationSuccessResponseRule()

	// add unique operation ID rule
	rules["operation-operationId"] = GetOperationIdUniqueRule()

	// add operation params rule
	rules["operation-parameters"] = GetOperationParametersRule()

	// add operation tag defined rule
	rules["operation-tag-defined"] = GetGlobalOperationTagsRule()

	// add operation tag defined rule
	rules["path-params"] = GetPathParamsRule()

	// contact-properties
	rules["contact-properties"] = GetContactPropertiesRule()

	// info object: contains contact
	rules["info-contact"] = GetInfoContactRule()

	// info object: contains description
	rules["info-description"] = GetInfoDescriptionRule()

	// info object: contains a license
	rules["info-license"] = GetInfoLicenseRule()

	// info object: contains a license url
	rules["license-url"] = GetInfoLicenseUrlRule()

	// check no eval statements in markdown descriptions.
	rules["no-eval-in-markdown"] = GetNoEvalInMarkdownRule()

	// check no script statements in markdown descriptions.
	rules["no-script-tags-in-markdown"] = GetNoScriptTagsInMarkdownRule()

	// check tags are in alphabetical order
	rules["openapi-tags-alphabetical"] = GetOpenApiTagsAlphabeticalRule()

	// check tags exist correctly
	rules["openapi-tags"] = GetOpenApiTagsRule()

	// check operation tags exist correctly
	rules["operation-tags"] = GetOperationTagsRule()

	// check all operations have a description, and they match a set length.
	rules["operation-description"] = GetOperationDescriptionRule()

	// check all components have a description, and they match a set length.
	rules["component-description"] = GetComponentDescriptionsRule()

	// check for description and summary duplication
	rules["description-duplication"] = GetDescriptionDuplicationRule()

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

	// check enums respect specified types
	rules["typed-enum"] = GetTypedEnumRule()

	// check for duplication in enums
	rules["duplicated-entry-in-enum"] = GetDuplicatedEntryInEnumRule()

	// add no $ref siblings
	rules["no-$ref-siblings"] = GetNoRefSiblingsRule()

	// add unused component rule for openapi3
	rules["oas3-unused-component"] = GetOAS3UnusedComponentRule()

	// TODO: build in spec types so we don't run this twice :)
	//rules["oas2-unused-definition"] = unusedComponentRule

	// security for versions 2 and 3.
	rules["oas3-operation-security-defined"] = GetOAS3SecurityDefinedRule()
	rules["oas2-operation-security-defined"] = GetOAS2SecurityDefinedRule()

	// TODO: Examples is not efficient at all, needs cleaning up significantly,
	// should be broken down into sub rules most likely.

	// check all examples
	rules["oas-3valid-schema-example"] = GetExamplesRule()
	//*/

	set := &model.RuleSet{
		DocumentationURI: "https://quobix.com/vacuum/rules/openapi",
		Rules:            rules,
	}

	return set

}
