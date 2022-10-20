// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package rulesets

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/parser"
	"github.com/pb33f/libopenapi/utils"
	"regexp"
)

// GetContactPropertiesRule will return a rule configured to look at contact properties of a spec.
// it uses the in-built 'truthy' function
func GetContactPropertiesRule() *model.Rule {
	return &model.Rule{
		Name:         "Check contact properties: name, URL, email",
		Id:           contactProperties,
		Formats:      model.AllFormats,
		Description:  "Contact details are incomplete",
		Given:        "$.info.contact",
		Resolved:     true,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  false,
		Type:         validation,
		Severity:     model.SeverityInfo,
		Then: []model.RuleAction{
			{
				Field:    "name",
				Function: "truthy",
			},
			{
				Field:    "url",
				Function: "truthy",
			},
			{
				Field:    "email",
				Function: "truthy",
			},
		},
		HowToFix: contactPropertiesFix,
	}
}

// GetInfoContactRule Will return a rule that uses the truthy function to check if the
// info object contains a contact object
func GetInfoContactRule() *model.Rule {
	return &model.Rule{
		Name:         "Check for specification contact details",
		Id:           infoContact,
		Formats:      model.AllFormats,
		Description:  "Info section is missing contact details",
		Given:        "$.info",
		Resolved:     true,
		Recommended:  false,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Type:         validation,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Field:    "contact",
			Function: "truthy",
		},
		HowToFix: contactFix,
	}
}

// GetInfoDescriptionRule Will return a rule that uses the truthy function to check if the
// info object contains a description
func GetInfoDescriptionRule() *model.Rule {
	return &model.Rule{
		Name:         "Check for a specification description",
		Id:           infoDescription,
		Formats:      model.AllFormats,
		Description:  "Info section is missing a description",
		Given:        "$.info",
		Resolved:     true,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Field:    "description",
			Function: "truthy",
		},
		HowToFix: infoDescriptionFix,
	}
}

// GetInfoLicenseRule will return a rule that uses the truthy function to check if the
// info object contains a license
func GetInfoLicenseRule() *model.Rule {
	return &model.Rule{
		Name:         "Check for a license definition",
		Id:           infoLicense,
		Formats:      model.AllFormats,
		Description:  "Info section should contain a license",
		Given:        "$.info",
		Resolved:     true,
		Recommended:  false,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Type:         style,
		Severity:     model.SeverityInfo,
		Then: model.RuleAction{
			Field:    "license",
			Function: "truthy",
		},
		HowToFix: infoLicenseFix,
	}
}

// GetInfoLicenseUrlRule will return a rule that uses the truthy function to check if the
// info object contains a license with an url that is set.
func GetInfoLicenseUrlRule() *model.Rule {
	return &model.Rule{
		Name:         "Check if license is missing an URL",
		Id:           licenseUrl,
		Formats:      model.AllFormats,
		Description:  "License should contain an url",
		Given:        "$.info.license",
		Resolved:     true,
		Recommended:  false,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Type:         style,
		Severity:     model.SeverityInfo,
		Then: model.RuleAction{
			Field:    "url",
			Function: "truthy",
		},
		HowToFix: infoLicenseUrlFix,
	}
}

// GetNoEvalInMarkdownRule will return a rule that uses the pattern function to check if
// there is no eval statements markdown used in descriptions
func GetNoEvalInMarkdownRule() *model.Rule {

	fo := make(map[string]string)
	fo["pattern"] = "eval\\("
	comp, _ := regexp.Compile(fo["pattern"])

	return &model.Rule{
		Name:         "Check descriptions for  'eval()' statements",
		Id:           noEvalInMarkdown,
		Formats:      model.AllFormats,
		Description:  "Markdown descriptions must not have `eval()` statements'",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryValidation],
		Type:         validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function:        "noEvalDescription",
			FunctionOptions: fo,
		},
		PrecompiledPattern: comp,
		HowToFix:           noEvalInMarkDownFix,
	}
}

// GetNoScriptTagsInMarkdownRule will return a rule that uses the pattern function to check if
// there is no script tags used in descriptions and the title.
func GetNoScriptTagsInMarkdownRule() *model.Rule {

	fo := make(map[string]string)
	fo["pattern"] = "<script"
	comp, _ := regexp.Compile(fo["pattern"])

	return &model.Rule{
		Name:         "Check descriptions for '<script>' tags",
		Id:           noScriptTagsInMarkdown,
		Formats:      model.AllFormats,
		Description:  "Markdown descriptions must not have `<script>` tags'",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryValidation],
		Type:         validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function:        "noEvalDescription",
			FunctionOptions: fo,
		},
		PrecompiledPattern: comp,
		HowToFix:           noScriptTagsInMarkdownFix,
	}
}

// GetOpenApiTagsAlphabeticalRule will return a rule that uses the alphabetical function to check if
// tags are in alphabetical order
func GetOpenApiTagsAlphabeticalRule() *model.Rule {

	fo := make(map[string]string)
	fo["keyedBy"] = "name"

	return &model.Rule{
		Name:         "Check tags are ordered alphabetically",
		Id:           openAPITagsAlphabetical,
		Formats:      model.AllFormats,
		Description:  "Tags must be in alphabetical order",
		Given:        "$.tags",
		Resolved:     true,
		Recommended:  false,
		RuleCategory: model.RuleCategories[model.CategoryTags],
		Type:         style,
		Severity:     model.SeverityInfo,
		Then: model.RuleAction{
			Function:        "alphabetical",
			FunctionOptions: fo,
		},
		HowToFix: openAPITagsAlphabeticalFix,
	}
}

// GetOpenApiTagsRule uses the schema function to check if there tags exist and that
// it's an array with at least one item.
func GetOpenApiTagsRule() *model.Rule {
	items := 1

	// create a schema to match against.
	opts := make(map[string]interface{})
	opts["schema"] = parser.Schema{
		Type: &utils.ArrayLabel,
		Items: &parser.Schema{
			Type:     &utils.ObjectLabel,
			MinItems: &items,
		},
		UniqueItems: true,
	}
	opts["forceValidation"] = true // this will be picked up by the schema function to force validation.
	//opts["unpack"] = true          // unpack will correctly unpack this data so the schema method can use it.

	return &model.Rule{
		Name:         "Check global tags are defined",
		Id:           openAPITags,
		Formats:      model.AllFormats,
		Description:  "Top level spec `tags` must not be empty, and must be an array",
		Given:        "$",
		Resolved:     true,
		RuleCategory: model.RuleCategories[model.CategoryTags],
		Recommended:  false,
		Type:         validation,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Field:           "tags",
			Function:        "schema",
			FunctionOptions: opts,
		},
		HowToFix: openAPITagsFix,
	}
}

// GetOAS2APISchemesRule uses the schema function to check if swagger has schemes and that
// it's an array with at least one item.
func GetOAS2APISchemesRule() *model.Rule {
	items := 1

	// create a schema to match against.
	opts := make(map[string]interface{})
	opts["schema"] = parser.Schema{
		Type: &utils.ArrayLabel,
		Items: &parser.Schema{
			Type:     &utils.StringLabel,
			MinItems: &items,
		},
		UniqueItems: true,
	}
	opts["forceValidation"] = true // this will be picked up by the schema function to force validation.

	return &model.Rule{
		Name:         "Check host schemes are defined",
		Id:           oas2APISchemes,
		Formats:      model.OAS2Format,
		Description:  "OpenAPI host `schemes` must be present and non-empty array",
		Given:        "$",
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         validation,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Field:           "schemes",
			Function:        "schema",
			FunctionOptions: opts,
		},
		HowToFix: oas2APISchemesFix,
	}
}

// GetOAS2HostNotExampleRule checks to make sure that example.com is not being used as a host.
// TODO: how common is this? should we keep it? change it?
func GetOAS2HostNotExampleRule() *model.Rule {
	opts := make(map[string]interface{})
	opts["notMatch"] = "example\\.com"
	comp, _ := regexp.Compile(opts["notMatch"].(string))
	return &model.Rule{
		Name:         "Check server URLs for example.com",
		Id:           oas2HostNotExample,
		Formats:      model.OAS2Format,
		Description:  "Host URL should not point at example.com",
		Given:        "$.host",
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         style,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Function:        "pattern",
			FunctionOptions: opts,
		},
		PrecompiledPattern: comp,
		HowToFix:           oas2HostNotExampleFix,
	}
}

// GetOAS3HostNotExampleRule checks to make sure that example.com is not being used as a host.
// TODO: how common is this? should we keep it? change it?
func GetOAS3HostNotExampleRule() *model.Rule {
	opts := make(map[string]interface{})
	opts["notMatch"] = "example\\.com"
	comp, _ := regexp.Compile(opts["notMatch"].(string))
	return &model.Rule{
		Name:         "Check server URLs for example.com",
		Id:           oas3HostNotExample,
		Formats:      model.OAS3AllFormat,
		Description:  "Server URL should not point at example.com",
		Given:        "$.servers[*].url",
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  false,
		Type:         style,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Function:        "pattern",
			FunctionOptions: opts,
		},
		PrecompiledPattern: comp,
		HowToFix:           oas3HostNotExampleFix,
	}
}

// GetOAS2HostTrailingSlashRule checks to make sure there is no trailing slash on the host
func GetOAS2HostTrailingSlashRule() *model.Rule {
	opts := make(map[string]interface{})
	opts["notMatch"] = "/$"
	comp, _ := regexp.Compile(opts["notMatch"].(string))
	return &model.Rule{
		Name:         "Check host for trailing slash",
		Id:           oas2HostTrailingSlash,
		Formats:      model.OAS2Format,
		Description:  "Host URL should not contain a trailing slash",
		Given:        "$.host",
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         style,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Function:        "pattern",
			FunctionOptions: opts,
		},
		PrecompiledPattern: comp,
		HowToFix:           oas2HostTrailingSlashFix,
	}
}

// GetOAS3HostTrailingSlashRule checks to make sure there is no trailing slash on the host
func GetOAS3HostTrailingSlashRule() *model.Rule {
	opts := make(map[string]interface{})
	opts["notMatch"] = "/$"
	comp, _ := regexp.Compile(opts["notMatch"].(string))
	return &model.Rule{
		Name:         "Check server url for trailing slash",
		Id:           oas3HostTrailingSlash,
		Formats:      model.OAS3Format,
		Description:  "server URL should not contain a trailing slash",
		Given:        "$.servers[*]",
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  false,
		Type:         style,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Field:           "url",
			Function:        "pattern",
			FunctionOptions: opts,
		},
		PrecompiledPattern: comp,
		HowToFix:           oas3HostTrailingSlashFix,
	}
}

// GetOperationDescriptionRule will return a rule that uses the truthy function to check if an operation
// has defined a description or not, or does not meet the required length
func GetOperationDescriptionRule() *model.Rule {
	opts := make(map[string]interface{})
	opts["minWords"] = "1" // there must be at least a single word.
	return &model.Rule{
		Name:         "Check operation description",
		Id:           operationDescription,
		Formats:      model.AllFormats,
		Description:  "Operation description checks",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryDescriptions],
		Type:         validation,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Function:        "oasDescriptions",
			FunctionOptions: opts,
		},
		HowToFix: operationDescriptionFix,
	}
}

// GetOAS2ParameterDescriptionRule will check specs to make sure parameters have a description.
func GetOAS2ParameterDescriptionRule() *model.Rule {
	return &model.Rule{
		Name:         "Check parameter description",
		Id:           oas2ParameterDescription,
		Formats:      model.OAS2Format,
		Description:  "Parameter description checks",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryDescriptions],
		Type:         style,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Function: "oasParamDescriptions",
		},
		HowToFix: oasParameterDescriptionFix,
	}
}

// GetOAS3ParameterDescriptionRule will check specs to make sure parameters have a description.
func GetOAS3ParameterDescriptionRule() *model.Rule {
	return &model.Rule{
		Name:         "Check parameter description",
		Id:           oas3ParameterDescription,
		Formats:      model.OAS3AllFormat,
		Description:  "Parameter description checks",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryDescriptions],
		Type:         style,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Function: "oasParamDescriptions",
		},
		HowToFix: oasParameterDescriptionFix,
	}
}

// GetDescriptionDuplicationRule will check if any descriptions have been copy/pasted or duplicated.
// all descriptions should be unique, otherwise what is the point?
func GetDescriptionDuplicationRule() *model.Rule {
	return &model.Rule{
		Name:         "Check descriptions for duplicates",
		Id:           descriptionDuplication,
		Formats:      model.AllFormats,
		Description:  "Description duplication check",
		Given:        "$",
		Resolved:     false,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryDescriptions],
		Type:         validation,
		Severity:     model.SeverityInfo,
		Then: model.RuleAction{
			Function: "oasDescriptionDuplication",
		},
		HowToFix: descriptionDuplicationFix,
	}
}

// GetComponentDescriptionsRule will check all components for description problems.
func GetComponentDescriptionsRule() *model.Rule {
	return &model.Rule{
		Name:         "Check component description",
		Formats:      model.OAS3AllFormat,
		Id:           componentDescription,
		Description:  "Component description check",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryDescriptions],
		Type:         validation,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Function: "oasComponentDescriptions",
		},
		HowToFix: componentDescriptionFix,
	}
}

// GetAPIServersRule checks to make sure there is a valid 'servers' definition in the document.
func GetAPIServersRule() *model.Rule {
	return &model.Rule{
		Name:         "Validate API server definitions",
		Id:           oas3APIServers,
		Formats:      model.OAS3Format,
		Description:  "Check for valid API servers definition",
		Given:        "$",
		Resolved:     false,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryValidation],
		Type:         validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "oasAPIServers",
		},
		HowToFix: oasServersFix,
	}
}

// GetOperationIdValidInUrlRule will check id an operationId will be valid when used in an url.
func GetOperationIdValidInUrlRule() *model.Rule {
	// TODO: re-build this the path is useless.
	opts := make(map[string]interface{})
	opts["match"] = "^[A-Za-z0-9-._~:/?#\\[\\]@!\\$&'()*+,;=]*$"
	comp, _ := regexp.Compile(opts["match"].(string))
	return &model.Rule{
		Name:         "Check operationId is URL friendly",
		Id:           operationOperationIdValidInUrl,
		Formats:      model.AllFormats,
		Description:  "OperationId must use URL friendly characters",
		Given:        "$.paths[*][*]",
		Resolved:     true,
		RuleCategory: model.RuleCategories[model.CategoryOperations],
		Recommended:  true,
		Type:         validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Field:           "operationId",
			Function:        "pattern",
			FunctionOptions: opts,
		},
		PrecompiledPattern: comp,
		HowToFix:           operationIdValidInUrlFix,
	}
}

// GetOperationTagsRule uses the schema function to check if there tags exist and that
// it's an array with at least one item.
func GetOperationTagsRule() *model.Rule {
	return &model.Rule{
		Name:         "Check operation tags are used",
		Id:           operationTags,
		Formats:      model.AllFormats,
		Description:  "Operation `tags` are missing/empty",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryTags],
		Type:         validation,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Function: "oasOperationTags",
		},
		HowToFix: operationTagsFix,
	}
}

// GetPathDeclarationsMustExistRule will check to make sure there are no empty path variables
func GetPathDeclarationsMustExistRule() *model.Rule {
	opts := make(map[string]interface{})
	opts["notMatch"] = "{}"
	comp, _ := regexp.Compile(opts["notMatch"].(string))
	return &model.Rule{
		Name:         "Check path parameter declarations",
		Id:           pathDeclarationsMustExist,
		Formats:      model.AllFormats,
		Description:  "Path parameter declarations must not be empty ex. `/api/{}` is invalid",
		Given:        "$.paths",
		Resolved:     true,
		RuleCategory: model.RuleCategories[model.CategoryOperations],
		Recommended:  true,
		Type:         validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function:        "pattern",
			FunctionOptions: opts,
		},
		PrecompiledPattern: comp,
		HowToFix:           pathDeclarationsMustExistFix,
	}
}

// GetPathNoTrailingSlashRule will make sure that paths don't have trailing slashes
func GetPathNoTrailingSlashRule() *model.Rule {
	opts := make(map[string]interface{})
	opts["notMatch"] = ".+\\/$"
	comp, _ := regexp.Compile(opts["notMatch"].(string))
	return &model.Rule{
		Name:         "Check path for any trailing slashes",
		Id:           pathKeysNoTrailingSlash,
		Formats:      model.AllFormats,
		Description:  "Path must not end with a slash",
		Given:        "$.paths",
		Resolved:     true,
		RuleCategory: model.RuleCategories[model.CategoryOperations],
		Recommended:  true,
		Type:         validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function:        "pattern",
			FunctionOptions: opts,
		},
		PrecompiledPattern: comp,
		HowToFix:           pathNoTrailingSlashFix,
	}
}

// GetPathNotIncludeQueryRule checks to ensure paths are not including any query parameters.
func GetPathNotIncludeQueryRule() *model.Rule {
	opts := make(map[string]interface{})
	opts["notMatch"] = "\\?"
	comp, _ := regexp.Compile(opts["notMatch"].(string))
	return &model.Rule{
		Name:         "Check path excludes query string",
		Id:           pathNotIncludeQuery,
		Formats:      model.AllFormats,
		Description:  "Path must not include query string",
		Given:        "$.paths",
		Resolved:     true,
		RuleCategory: model.RuleCategories[model.CategoryOperations],
		Recommended:  true,
		Type:         validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function:        "pattern",
			FunctionOptions: opts,
		},
		PrecompiledPattern: comp,
		HowToFix:           pathNotIncludeQueryFix,
	}
}

// GetTagDescriptionRequiredRule checks to ensure tags defined have been given a description
func GetTagDescriptionRequiredRule() *model.Rule {
	return &model.Rule{
		Name:         "Check tag description",
		Id:           tagDescription,
		Formats:      model.AllFormats,
		Description:  "Tag must have a description defined",
		Given:        "$.tags",
		Resolved:     true,
		Recommended:  false,
		RuleCategory: model.RuleCategories[model.CategoryTags],
		Type:         validation,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Field:    "description",
			Function: "truthy",
		},
		HowToFix: tagDescriptionRequiredFix,
	}
}

// GetTypedEnumRule checks to ensure enums are of the specified type
func GetTypedEnumRule() *model.Rule {
	return &model.Rule{
		Name:         "Check enum types",
		Id:           typedEnum,
		Formats:      model.AllFormats,
		Description:  "Enum values must respect the specified type",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategorySchemas],
		Type:         validation,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Function: "typedEnum",
		},
		HowToFix: typedEnumFix,
	}
}

// GetPathParamsRule checks if path params are valid and defined.
func GetPathParamsRule() *model.Rule {
	// add operation tag defined rule
	return &model.Rule{
		Name:         "Check path validity and definition",
		Id:           pathParamsRule,
		Formats:      model.AllFormats,
		Description:  "Path parameters must be defined and valid.",
		Given:        "$",
		Resolved:     true,
		RuleCategory: model.RuleCategories[model.CategoryOperations],
		Recommended:  true,
		Type:         validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "oasPathParam",
		},
		HowToFix: pathParamsFix,
	}
}

// GetGlobalOperationTagsRule will check that an operation tag exists in top level tags
func GetGlobalOperationTagsRule() *model.Rule {
	return &model.Rule{
		Name:         "Check operation tags exist globally",
		Id:           operationTagDefined,
		Formats:      model.AllFormats,
		Description:  "Operation tags must be defined in global tags.",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryTags],
		Type:         validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "oasTagDefined",
		},
		HowToFix: globalOperationTagsFix,
	}
}

// GetOperationParametersRule will check that an operation has valid parameters defined
func GetOperationParametersRule() *model.Rule {
	return &model.Rule{
		Name:         "Check operation parameters",
		Id:           operationParameters,
		Formats:      model.AllFormats,
		Description:  "Operation parameters are unique and non-repeating.",
		Given:        "$.paths",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryOperations],
		Type:         validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "oasOpParams",
		},
		HowToFix: operationParametersFix,
	}
}

// GetOAS2FormDataConsumesRule will check that an "application/x-www-form-urlencoded" or "multipart/form-data"
// is defined in the 'consumes' node for in any parameters that use in formData.
func GetOAS2FormDataConsumesRule() *model.Rule {
	return &model.Rule{
		Name:    "Check operation parameter 'formData' definition",
		Id:      oas2OperationFormDataConsumeCheck,
		Formats: model.OAS2Format,
		Description: "Operations with `in: formData` parameter must include `application/x-www-form-urlencoded` or" +
			" `multipart/form-data` in their `consumes` property.",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryOperations],
		Type:         validation,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Function: "oasOpFormDataConsumeCheck",
		},
		HowToFix: formDataConsumesFix,
	}
}

// GetOAS2PolymorphicAnyOfRule will check that 'anyOf' has not been used in a swagger spec (introduced in 3.0)
func GetOAS2PolymorphicAnyOfRule() *model.Rule {
	return &model.Rule{
		Name:         "Check for invalid use of 'anyOf'",
		Id:           oas2AnyOf,
		Formats:      model.OAS2Format,
		Description:  "`anyOf` was introduced in OpenAPI 3.0, cannot be used un OpenAPI 2 specs",
		Given:        "$",
		Resolved:     false,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategorySchemas],
		Type:         validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "oasPolymorphicAnyOf",
		},
		HowToFix: oas2AnyOfFix,
	}
}

// GetOAS2PolymorphicOneOfRule will check that 'oneOf' has not been used in a swagger spec (introduced in 3.0)
func GetOAS2PolymorphicOneOfRule() *model.Rule {
	return &model.Rule{
		Name:         "Check for invalid use of 'oneOf'",
		Id:           oas2OneOf,
		Formats:      model.OAS2Format,
		Description:  "`oneOf` was introduced in OpenAPI 3.0, cannot be used un OpenAPI 2 specs",
		Given:        "$",
		Resolved:     false,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategorySchemas],
		Type:         validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "oasPolymorphicOneOf",
		},
		HowToFix: oas2OneOfFix,
	}
}

// GetOAS2SchemaRule will check that the schema is valid for swagger docs.
func GetOAS2SchemaRule() *model.Rule {
	return &model.Rule{
		Name:         "Check schema is valid OpenAPI 2",
		Id:           oas2Schema,
		Formats:      model.OAS2Format,
		Description:  "OpenAPI 2 specification is invalid",
		Given:        "$",
		Resolved:     false,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryValidation],
		Type:         validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "oasDocumentSchema",
		},
		HowToFix: oas2SchemaCheckFix,
	}
}

// GetOAS3SchemaRule will check that the schema is valid for openapi 3+ docs.
func GetOAS3SchemaRule() *model.Rule {
	return &model.Rule{
		Name:         "Check spec is valid OpenAPI 3",
		Id:           oas3Schema,
		Formats:      model.OAS3Format,
		Description:  "OpenAPI 3 specification is invalid",
		Given:        "$",
		Resolved:     false,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategorySchemas],
		Type:         validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "oasDocumentSchema",
		},
		HowToFix: oas3SchemaCheckFix,
	}
}

// GetOperationIdUniqueRule will check to make sure that operationIds are all unique and non-repeating
func GetOperationIdUniqueRule() *model.Rule {
	return &model.Rule{
		Name:         "Check operations for unique operationId",
		Id:           operationOperationIdUnique,
		Formats:      model.AllFormats,
		Description:  "Every operation must have unique `operationId`.",
		Given:        "$.paths",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryOperations],
		Type:         validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "oasOpIdUnique",
		},
		HowToFix: operationIdUniqueFix,
	}
}

// GetOperationSingleTagRule will check to see if an operation has more than a single tag
func GetOperationSingleTagRule() *model.Rule {
	return &model.Rule{
		Name:         "Check operations for multiple tags",
		Id:           operationSingularTag,
		Formats:      model.AllFormats,
		Description:  "Operation cannot have more than a single tag defined",
		Given:        "$",
		Resolved:     false,
		Recommended:  false,
		RuleCategory: model.RuleCategories[model.CategoryTags],
		Type:         validation,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Function: "oasOpSingleTag",
		},
		HowToFix: operationSingleTagFix,
	}
}

// GetOAS2APIHostRule will check swagger specs for the host property being set.
func GetOAS2APIHostRule() *model.Rule {
	return &model.Rule{
		Name:         "Check spec for 'host' value",
		Id:           oas2APIHost,
		Formats:      model.OAS2Format,
		Description:  "OpenAPI `host` must be present and a non-empty string",
		Given:        "$",
		Resolved:     false,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Type:         style,
		Severity:     model.SeverityInfo,
		Then: model.RuleAction{
			Field:    "host",
			Function: "truthy",
		},
		HowToFix: oas2APIHostFix,
	}
}

// GetOperationIdRule will check to make sure that operationIds  exist on all operations
func GetOperationIdRule() *model.Rule {
	return &model.Rule{
		Name:         "Check operations for an operationId",
		Id:           operationOperationId,
		Formats:      model.AllFormats,
		Description:  "Every operation must contain an `operationId`.",
		Given:        "$",
		Resolved:     false,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryOperations],
		Type:         validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "oasOpId",
		},
		HowToFix: operationIdExistsFix,
	}
}

// GetOperationSuccessResponseRule will check that every operation has a success response defined.
func GetOperationSuccessResponseRule() *model.Rule {
	return &model.Rule{
		Name:         "Check operations for success response",
		Id:           operationSuccessResponse,
		Formats:      model.AllFormats,
		Description:  "Operation must have at least one `2xx` or a `3xx` response.",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryOperations],
		Type:         style,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Field:    "responses",
			Function: "oasOpSuccessResponse",
		},
		HowToFix: operationSuccessResponseFix,
	}
}

// GetDuplicatedEntryInEnumRule will check that enums used are not duplicates
func GetDuplicatedEntryInEnumRule() *model.Rule {
	return &model.Rule{
		Name:         "Check for duplicate enum entries",
		Id:           duplicatedEntryInEnum,
		Formats:      model.AllFormats,
		Description:  "Enum values must not have duplicate entry",
		Given:        "$",
		Resolved:     false,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategorySchemas],
		Type:         validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "duplicatedEnum",
		},
		HowToFix: duplicatedEntryInEnumFix,
	}
}

// GetNoRefSiblingsRule will check that there are no sibling nodes next to a $ref (which is technically invalid)
func GetNoRefSiblingsRule() *model.Rule {
	return &model.Rule{
		Name:         "Check for siblings to $ref values",
		Id:           noRefSiblings,
		Formats:      model.AllFormats,
		Description:  "$ref values cannot be placed next to other properties (like a description)",
		Given:        "$",
		Resolved:     false,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategorySchemas],
		Type:         validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "refSiblings",
		},
		HowToFix: noRefSiblingsFix,
	}
}

// GetOAS3UnusedComponentRule will check that there aren't any components anywhere that haven't been used.
func GetOAS3UnusedComponentRule() *model.Rule {
	return &model.Rule{
		Name:         "Check for unused components",
		Id:           oas3UnusedComponent,
		Formats:      model.OAS3AllFormat,
		Description:  "Check for unused components and bad references",
		Given:        "$",
		Resolved:     false,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategorySchemas],
		Type:         validation,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Function: "oasUnusedComponent",
		},
		HowToFix: unusedComponentFix,
	}
}

// GetOAS2UnusedComponentRule will check that there aren't any components anywhere that haven't been used.
func GetOAS2UnusedComponentRule() *model.Rule {
	return &model.Rule{
		Name:         "Check for unused definitions",
		Id:           oas2UnusedDefinition,
		Formats:      model.OAS2Format,
		Description:  "Check for unused definitions and bad references",
		Given:        "$",
		Resolved:     false,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategorySchemas],
		Type:         validation,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Function: "oasUnusedComponent",
		},
		HowToFix: oas3UnusedComponentFix,
	}
}

// GetOAS3SecurityDefinedRule will check that security definitions exist and validate for OpenAPI 3
func GetOAS3SecurityDefinedRule() *model.Rule {
	oasSecurityPath := make(map[string]string)
	oasSecurityPath["schemesPath"] = "$.components.securitySchemes"

	return &model.Rule{
		Name:         "Check operation security",
		Id:           oas3OperationSecurityDefined,
		Formats:      model.OAS3AllFormat,
		Description:  "`security` values must match a scheme defined in components.securitySchemes",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategorySecurity],
		Type:         validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function:        "oasOpSecurityDefined",
			FunctionOptions: oasSecurityPath,
		},
		HowToFix: oas3SecurityDefinedFix,
	}
}

// GetOAS2SecurityDefinedRule will check that security definitions exist and validate for OpenAPI 2
func GetOAS2SecurityDefinedRule() *model.Rule {
	return &model.Rule{
		Name:         "Check operation security",
		Id:           oas2OperationSecurityDefined,
		Formats:      model.OAS2Format,
		Description:  "`security` values must match a scheme defined in securityDefinitions",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategorySecurity],
		Type:         validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "oas2OpSecurityDefined",
		},
		HowToFix: oas2SecurityDefinedFix,
	}
}

// GetOAS2DiscriminatorRule will check swagger schemas to ensure they are using discriminations correctly.
func GetOAS2DiscriminatorRule() *model.Rule {
	return &model.Rule{
		Name:         "Discriminator check",
		Id:           oas2Discriminator,
		Formats:      model.OAS2Format,
		Description:  "discriminators are used correctly in schemas",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategorySchemas],
		Type:         validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "oasDiscriminator",
		},
		HowToFix: oas2DiscriminatorFix,
	}
}

// GetOAS3ExamplesRule will check the entire spec for correct example use.
func GetOAS3ExamplesRule() *model.Rule {
	return &model.Rule{
		Name:         "Check all examples",
		Id:           oas3ValidSchemaExample,
		Formats:      model.OAS3AllFormat,
		Description:  "Examples must be present and valid for operations and components",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryExamples],
		Type:         validation,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Function: "oasExample",
		},
		HowToFix: oas3ExamplesFix,
	}
}

// GetOAS2ExamplesRule will check the entire spec for correct example use.
func GetOAS2ExamplesRule() *model.Rule {
	return &model.Rule{
		Name:         "Check all examples",
		Id:           oas2ValidSchemaExample,
		Formats:      model.OAS2Format,
		Description:  "Examples must be present and valid for operations and components",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryExamples],
		Type:         validation,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Function: "oasExample",
		},
		HowToFix: oas3ExamplesFix,
	}
}

// NoAmbiguousPaths will check for paths that are ambiguous with one another
func NoAmbiguousPaths() *model.Rule {
	return &model.Rule{
		Name:         "No ambiguous paths, each path must resolve unambiguously",
		Id:           noAmbiguousPaths,
		Formats:      model.AllFormats,
		Description:  "Paths need to resolve unambiguously from one another",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryOperations],
		Type:         validation,
		Severity:     model.SeverityError,
		Then: model.RuleAction{
			Function: "ambiguousPaths",
		},
		HowToFix: ambiguousPathsFix,
	}
}

// GetNoVerbsInPathRule will check all paths to make sure not HTTP verbs have been used as a segment.
func GetNoVerbsInPathRule() *model.Rule {
	return &model.Rule{
		Name:         "Paths cannot contain HTTP verbs as segments",
		Id:           noVerbsInPath,
		Formats:      model.AllFormats,
		Description:  "Path segments must not contain an HTTP verb",
		Given:        "$",
		Resolved:     false,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryOperations],
		Type:         style,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Function: "noVerbsInPath",
		},
		HowToFix: noVerbsInPathFix,
	}
}

// GetPathsKebabCaseRule will check that each path segment is kebab-case
func GetPathsKebabCaseRule() *model.Rule {
	return &model.Rule{
		Name:         "Path segments must be kebab-case only",
		Id:           pathsKebabCase,
		Formats:      model.AllFormats,
		Description:  "Path segments must only use kebab-case (no underscores or uppercase)",
		Given:        "$",
		Resolved:     false,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryOperations],
		Type:         validation,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Function: "pathsKebabCase",
		},
		HowToFix: pathsKebabCaseFix,
	}
}

// GetOperationErrorResponseRule will return the rule for checking for a 4xx response defined in operations.
func GetOperationErrorResponseRule() *model.Rule {
	return &model.Rule{
		Name:         "Operations must return at least 4xx user error response",
		Id:           operationErrorResponse,
		Formats:      model.AllFormats,
		Description:  "Make sure operations return at least one `4xx` error response to help with bad requests",
		Given:        "$.paths",
		Resolved:     true,
		Recommended:  false,
		RuleCategory: model.RuleCategories[model.CategoryOperations],
		Type:         validation,
		Severity:     model.SeverityWarn,
		Then: model.RuleAction{
			Function: "oasOpErrorResponse",
		},
		HowToFix: operationsErrorResponseFix,
	}
}
