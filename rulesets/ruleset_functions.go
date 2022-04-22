// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package rulesets

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/parser"
	"github.com/daveshanley/vacuum/utils"
	"regexp"
)

// GetContactPropertiesRule will return a rule configured to look at contact properties of a spec.
// it uses the in-built 'truthy' function
func GetContactPropertiesRule() *model.Rule {
	return &model.Rule{
		Id:           contactProperties,
		Formats:      model.AllFormats,
		Description:  "Contact details are incomplete",
		Given:        "$.info.contact",
		Resolved:     true,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         validation,
		Severity:     info,
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
	}
}

// GetInfoContactRule Will return a rule that uses the truthy function to check if the
// info object contains a contact object
func GetInfoContactRule() *model.Rule {
	return &model.Rule{
		Id:           infoContact,
		Formats:      model.AllFormats,
		Description:  "Info section is missing contact details",
		Given:        "$.info",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Type:         validation,
		Severity:     warn,
		Then: model.RuleAction{
			Field:    "contact",
			Function: "truthy",
		},
	}
}

// GetInfoDescriptionRule Will return a rule that uses the truthy function to check if the
// info object contains a description
func GetInfoDescriptionRule() *model.Rule {
	return &model.Rule{
		Id:           infoDescription,
		Formats:      model.AllFormats,
		Description:  "Info section is missing a description",
		Given:        "$.info",
		Resolved:     true,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         validation,
		Severity:     err,
		Then: model.RuleAction{
			Field:    "description",
			Function: "truthy",
		},
	}
}

// GetInfoLicenseRule will return a rule that uses the truthy function to check if the
// info object contains a license
func GetInfoLicenseRule() *model.Rule {
	return &model.Rule{
		Id:           infoLicense,
		Formats:      model.AllFormats,
		Description:  "Info section should contain a license",
		Given:        "$.info",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Type:         style,
		Severity:     info,
		Then: model.RuleAction{
			Field:    "license",
			Function: "truthy",
		},
	}
}

// GetInfoLicenseUrlRule will return a rule that uses the truthy function to check if the
// info object contains a license with an url that is set.
func GetInfoLicenseUrlRule() *model.Rule {
	return &model.Rule{
		Id:           licenseUrl,
		Formats:      model.AllFormats,
		Description:  "License should contain an url",
		Given:        "$.info.license",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Type:         style,
		Severity:     info,
		Then: model.RuleAction{
			Field:    "url",
			Function: "truthy",
		},
	}
}

// GetNoEvalInMarkdownRule will return a rule that uses the pattern function to check if
// there is no eval statements markdown used in descriptions
func GetNoEvalInMarkdownRule() *model.Rule {

	fo := make(map[string]string)
	fo["pattern"] = "eval\\("
	comp, _ := regexp.Compile(fo["pattern"])

	return &model.Rule{
		Id:           noEvalInMarkdown,
		Formats:      model.AllFormats,
		Description:  "Markdown descriptions must not have 'eval()' statements'",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryValidation],
		Type:         validation,
		Severity:     err,
		Then: model.RuleAction{
			Function:        "noEvalDescription",
			FunctionOptions: fo,
		},
		PrecomiledPattern: comp,
	}
}

// GetNoScriptTagsInMarkdownRule will return a rule that uses the pattern function to check if
// there is no script tags used in descriptions and the title.
func GetNoScriptTagsInMarkdownRule() *model.Rule {

	fo := make(map[string]string)
	fo["pattern"] = "<script"
	comp, _ := regexp.Compile(fo["pattern"])

	return &model.Rule{
		Id:           noScriptTagsInMarkdown,
		Formats:      model.AllFormats,
		Description:  "Markdown descriptions must not have '<script>' tags'",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryValidation],
		Type:         validation,
		Severity:     err,
		Then: model.RuleAction{
			Function:        "noEvalDescription",
			FunctionOptions: fo,
		},
		PrecomiledPattern: comp,
	}
}

// GetOpenApiTagsAlphabeticalRule will return a rule that uses the alphabetical function to check if
// tags are in alphabetical order
func GetOpenApiTagsAlphabeticalRule() *model.Rule {

	fo := make(map[string]string)
	fo["keyedBy"] = "name"

	return &model.Rule{
		Id:           openAPITagsAlphabetical,
		Formats:      model.AllFormats,
		Description:  "Tags must be in alphabetical order",
		Given:        "$.tags",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryTags],
		Type:         style,
		Severity:     info,
		Then: model.RuleAction{
			Function:        "alphabetical",
			FunctionOptions: fo,
		},
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
		Id:           openAPITags,
		Formats:      model.AllFormats,
		Description:  "Top level spec 'tags' must not be empty, and must be an array",
		Given:        "$",
		Resolved:     true,
		RuleCategory: model.RuleCategories[model.CategoryTags],
		Recommended:  true,
		Type:         validation,
		Severity:     warn,
		Then: model.RuleAction{
			Field:           "tags",
			Function:        "schema",
			FunctionOptions: opts,
		},
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
		Id:           oas2APISchemes,
		Formats:      model.AllFormats,
		Description:  "OpenAPI host 'schemes' must be present and non-empty array.",
		Given:        "$",
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         validation,
		Severity:     warn,
		Then: model.RuleAction{
			Field:           "schemes",
			Function:        "schema",
			FunctionOptions: opts,
		},
	}
}

// GetOAS2HostNotExampleRule checks to make sure that example.com is not being used as a host.
// TODO: how common is this? should we keep it? change it?
func GetOAS2HostNotExampleRule() *model.Rule {
	opts := make(map[string]interface{})
	opts["notMatch"] = "example\\.com"
	comp, _ := regexp.Compile(opts["notMatch"].(string))
	return &model.Rule{
		Id:           oas2HostNotExample,
		Formats:      model.OAS2Format,
		Description:  "Host URL should not point at example.com",
		Given:        "$.host",
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         style,
		Severity:     warn,
		Then: model.RuleAction{
			Field:           "host",
			Function:        "pattern",
			FunctionOptions: opts,
		},
		PrecomiledPattern: comp,
	}
}

// GetOAS3HostNotExampleRule checks to make sure that example.com is not being used as a host.
// TODO: how common is this? should we keep it? change it?
func GetOAS3HostNotExampleRule() *model.Rule {
	opts := make(map[string]interface{})
	opts["notMatch"] = "example\\.com"
	comp, _ := regexp.Compile(opts["notMatch"].(string))
	return &model.Rule{
		Id:           oas3HostNotExample,
		Formats:      model.OAS3AllFormat,
		Description:  "Server URL should not point at example.com",
		Given:        "$.servers[*].url",
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         style,
		Severity:     warn,
		Then: model.RuleAction{
			Function:        "pattern",
			FunctionOptions: opts,
		},
		PrecomiledPattern: comp,
	}
}

// GetOAS2HostTrailingSlashRule checks to make sure there is no trailing slash on the host
func GetOAS2HostTrailingSlashRule() *model.Rule {
	opts := make(map[string]interface{})
	opts["notMatch"] = "/$"
	comp, _ := regexp.Compile(opts["notMatch"].(string))
	return &model.Rule{
		Id:           oas2HostTrailingSlash,
		Formats:      model.OAS2Format,
		Description:  "Host URL should not contain a trailing slash",
		Given:        "$.host",
		Resolved:     false,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Recommended:  true,
		Type:         style,
		Severity:     warn,
		Then: model.RuleAction{
			Field:           "host",
			Function:        "pattern",
			FunctionOptions: opts,
		},
		PrecomiledPattern: comp,
	}
}

// GetOperationDescriptionRule will return a rule that uses the truthy function to check if an operation
// has defined a description or not, or does not meet the required length
func GetOperationDescriptionRule() *model.Rule {
	opts := make(map[string]interface{})
	opts["minWords"] = "1" // there must be at least a single word.
	return &model.Rule{
		Id:           operationDescription,
		Formats:      model.AllFormats,
		Description:  "Operation description checks",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryDescriptions],
		Type:         validation,
		Severity:     warn,
		Then: model.RuleAction{
			Function:        "oasDescriptions",
			FunctionOptions: opts,
		},
	}
}

// GetOAS2ParameterDescriptionRule will check specs to make sure parameters have a description.
func GetOAS2ParameterDescriptionRule() *model.Rule {
	return &model.Rule{
		Id:           oas2ParameterDescription,
		Formats:      model.OAS2Format,
		Description:  "Parameter description checks",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryDescriptions],
		Type:         style,
		Severity:     warn,
		Then: model.RuleAction{
			Function: "oasParamDescriptions",
		},
	}
}

// GetOAS3ParameterDescriptionRule will check specs to make sure parameters have a description.
func GetOAS3ParameterDescriptionRule() *model.Rule {
	return &model.Rule{
		Id:           oas3ParameterDescription,
		Formats:      model.OAS3AllFormat,
		Description:  "Parameter description checks",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryDescriptions],
		Type:         style,
		Severity:     warn,
		Then: model.RuleAction{
			Function: "oasParamDescriptions",
		},
	}
}

// GetDescriptionDuplicationRule will check if any descriptions have been copy/pasted or duplicated.
// all descriptions should be unique, otherwise what is the point?
func GetDescriptionDuplicationRule() *model.Rule {
	return &model.Rule{
		Id:           descriptionDuplication,
		Formats:      model.AllFormats,
		Description:  "Description duplication check",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryDescriptions],
		Type:         validation,
		Severity:     info,
		Then: model.RuleAction{
			Function: "oasDescriptionDuplication",
		},
	}
}

// GetComponentDescriptionsRule will check all components for description problems.
func GetComponentDescriptionsRule() *model.Rule {
	return &model.Rule{
		Formats:      model.AllFormats,
		Description:  "Component description check",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryDescriptions],
		Type:         validation,
		Severity:     warn,
		Then: model.RuleAction{
			Function: "oasComponentDescriptions",
		},
	}
}

// GetAPIServersRule checks to make sure there is a valid 'servers' definition in the document.
func GetAPIServersRule() *model.Rule {
	return &model.Rule{
		Id:           oas3APIServers,
		Formats:      model.OAS3Format,
		Description:  "Check for valid API servers definition",
		Given:        "$",
		Resolved:     false,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryValidation],
		Type:         validation,
		Severity:     err,
		Then: model.RuleAction{
			Function: "oasAPIServers",
		},
	}
}

// GetOperationIdValidInUrlRule will check id an operationId will be valid when used in an url.
func GetOperationIdValidInUrlRule() *model.Rule {
	// TODO: re-build this the path is useless.
	opts := make(map[string]interface{})
	opts["match"] = "^[A-Za-z0-9-._~:/?#\\[\\]@!\\$&'()*+,;=]*$"
	comp, _ := regexp.Compile(opts["match"].(string))
	return &model.Rule{
		Id:           operationOperationIdValidInUrl,
		Formats:      model.AllFormats,
		Description:  "OperationId must use URL friendly characters",
		Given:        AllOperationsPath,
		Resolved:     true,
		RuleCategory: model.RuleCategories[model.CategoryOperations],
		Recommended:  true,
		Type:         validation,
		Severity:     err,
		Then: model.RuleAction{
			Field:           "operationId",
			Function:        "pattern",
			FunctionOptions: opts,
		},
		PrecomiledPattern: comp,
	}
}

// GetOperationTagsRule uses the schema function to check if there tags exist and that
// it's an array with at least one item.
func GetOperationTagsRule() *model.Rule {
	return &model.Rule{
		Id:           operationTags,
		Formats:      model.AllFormats,
		Description:  "Operation 'tags' are missing/empty",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryTags],
		Type:         validation,
		Severity:     warn,
		Then: model.RuleAction{
			Function: "oasOperationTags",
		},
	}
}

// GetPathDeclarationsMustExistRule will check to make sure there are no empty path variables
func GetPathDeclarationsMustExistRule() *model.Rule {
	opts := make(map[string]interface{})
	opts["notMatch"] = "{}"
	comp, _ := regexp.Compile(opts["notMatch"].(string))
	return &model.Rule{
		Id:           pathDeclarationsMustExist,
		Formats:      model.AllFormats,
		Description:  "Path parameter declarations must not be empty ex. '/api/{}' is invalid",
		Given:        "$.paths",
		Resolved:     true,
		RuleCategory: model.RuleCategories[model.CategoryOperations],
		Recommended:  true,
		Type:         validation,
		Severity:     err,
		Then: model.RuleAction{
			Function:        "pattern",
			FunctionOptions: opts,
		},
		PrecomiledPattern: comp,
	}
}

// GetPathNoTrailingSlashRule will make sure that paths don't have trailing slashes
func GetPathNoTrailingSlashRule() *model.Rule {
	opts := make(map[string]interface{})
	opts["notMatch"] = ".+\\/$"
	comp, _ := regexp.Compile(opts["notMatch"].(string))
	return &model.Rule{
		Id:           pathKeysNoTrailingSlash,
		Formats:      model.AllFormats,
		Description:  "Path must not end with a slash",
		Given:        "$.paths",
		Resolved:     true,
		RuleCategory: model.RuleCategories[model.CategoryOperations],
		Recommended:  true,
		Type:         validation,
		Severity:     err,
		Then: model.RuleAction{
			Function:        "pattern",
			FunctionOptions: opts,
		},
		PrecomiledPattern: comp,
	}
}

// GetPathNotIncludeQueryRule checks to ensure paths are not including any query parameters.
func GetPathNotIncludeQueryRule() *model.Rule {
	opts := make(map[string]interface{})
	opts["notMatch"] = "\\?"
	comp, _ := regexp.Compile(opts["notMatch"].(string))
	return &model.Rule{
		Id:           pathNotIncludeQuery,
		Formats:      model.AllFormats,
		Description:  "Path must not include query string",
		Given:        "$.paths",
		Resolved:     true,
		RuleCategory: model.RuleCategories[model.CategoryOperations],
		Recommended:  true,
		Type:         validation,
		Severity:     err,
		Then: model.RuleAction{
			Function:        "pattern",
			FunctionOptions: opts,
		},
		PrecomiledPattern: comp,
	}
}

// GetTagDescriptionRequiredRule checks to ensure tags defined have been given a description
func GetTagDescriptionRequiredRule() *model.Rule {
	return &model.Rule{
		Id:           tagDescription,
		Formats:      model.AllFormats,
		Description:  "Tag must have a description defined",
		Given:        "$.tags",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryTags],
		Type:         validation,
		Severity:     warn,
		Then: model.RuleAction{
			Field:    "description",
			Function: "truthy",
		},
	}
}

// GetTypedEnumRule checks to ensure enums are of the specified type
func GetTypedEnumRule() *model.Rule {
	return &model.Rule{
		Id:           typedEnum,
		Formats:      model.AllFormats,
		Description:  "Enum values must respect the specified type",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategorySchemas],
		Type:         validation,
		Severity:     warn,
		Then: model.RuleAction{
			Function: "typedEnum",
		},
	}
}

// GetPathParamsRule checks if path params are valid and defined.
func GetPathParamsRule() *model.Rule {
	// add operation tag defined rule
	return &model.Rule{
		Id:           pathParamsRule,
		Formats:      model.AllFormats,
		Description:  "Path parameters must be defined and valid.",
		Given:        "$",
		Resolved:     true,
		RuleCategory: model.RuleCategories[model.CategoryOperations],
		Recommended:  true,
		Type:         validation,
		Severity:     err,
		Then: model.RuleAction{
			Function: "oasPathParam",
		},
	}
}

// GetGlobalOperationTagsRule will check that an operation tag exists in top level tags
func GetGlobalOperationTagsRule() *model.Rule {
	return &model.Rule{
		Id:           operationTagDefined,
		Formats:      model.AllFormats,
		Description:  "Operation tags must be defined in global tags.",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryTags],
		Type:         validation,
		Severity:     err,
		Then: model.RuleAction{
			Function: "oasTagDefined",
		},
	}
}

// GetOperationParametersRule will check that an operation has valid parameters defined
func GetOperationParametersRule() *model.Rule {
	return &model.Rule{
		Id:           operationParameters,
		Formats:      model.AllFormats,
		Description:  "Operation parameters are unique and non-repeating.",
		Given:        "$.paths",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryOperations],
		Type:         validation,
		Severity:     err,
		Then: model.RuleAction{
			Function: "oasOpParams",
		},
	}
}

// GetOAS2FormDataConsumesRule will check that an "application/x-www-form-urlencoded" or "multipart/form-data"
// is defined in the 'consumes' node for in any parameters that use in formData.
func GetOAS2FormDataConsumesRule() *model.Rule {
	return &model.Rule{
		Id:      oas2OperationFormDataConsumeCheck,
		Formats: model.OAS2Format,
		Description: "Operations with 'in: formData' parameter must include 'application/x-www-form-urlencoded' or" +
			" 'multipart/form-data' in their 'consumes' property.",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryOperations],
		Type:         validation,
		Severity:     warn,
		Then: model.RuleAction{
			Function: "oasOpFormDataConsumeCheck",
		},
	}
}

// GetOAS2PolymorphicAnyOfRule will check that 'anyOf' has not been used in a swagger spec (introduced in 3.0)
func GetOAS2PolymorphicAnyOfRule() *model.Rule {
	return &model.Rule{
		Id:           oas2AnyOf,
		Formats:      model.OAS2Format,
		Description:  "'anyOf' was introduced in OpenAPI 3.0, cannot be used un OpenAPI 2 specs",
		Given:        "$",
		Resolved:     false,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategorySchemas],
		Type:         validation,
		Severity:     err,
		Then: model.RuleAction{
			Function: "oasPolymorphicAnyOf",
		},
	}
}

// GetOAS2PolymorphicOneOfRule will check that 'oneOf' has not been used in a swagger spec (introduced in 3.0)
func GetOAS2PolymorphicOneOfRule() *model.Rule {
	return &model.Rule{
		Id:           oas2OneOf,
		Formats:      model.OAS2Format,
		Description:  "'oneOf' was introduced in OpenAPI 3.0, cannot be used un OpenAPI 2 specs",
		Given:        "$",
		Resolved:     false,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategorySchemas],
		Type:         validation,
		Severity:     err,
		Then: model.RuleAction{
			Function: "oasPolymorphicOneOf",
		},
	}
}

// GetOAS2SchemaRule will check that the schema is valid for swagger docs.
func GetOAS2SchemaRule() *model.Rule {
	return &model.Rule{
		Id:           oas2Schema,
		Formats:      model.OAS2Format,
		Description:  "OpenAPI 2 specification is invalid",
		Given:        "$",
		Resolved:     false,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryValidation],
		Type:         validation,
		Severity:     err,
		Then: model.RuleAction{
			Function: "oasDocumentSchema",
		},
	}
}

// GetOAS3SchemaRule will check that the schema is valid for openapi 3+ docs.
func GetOAS3SchemaRule() *model.Rule {
	return &model.Rule{
		Id:           oas3Schema,
		Formats:      model.OAS3Format,
		Description:  "OpenAPI 3 specification is invalid",
		Given:        "$",
		Resolved:     false,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryValidation],
		Type:         validation,
		Severity:     err,
		Then: model.RuleAction{
			Function: "oasDocumentSchema",
		},
	}
}

// GetOperationIdUniqueRule will check to make sure that operationIds are all unique and non-repeating
func GetOperationIdUniqueRule() *model.Rule {
	return &model.Rule{
		Id:           operationOperationIdUnique,
		Formats:      model.AllFormats,
		Description:  "Every operation must have unique \"operationId\".",
		Given:        "$.paths",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryOperations],
		Type:         validation,
		Severity:     err,
		Then: model.RuleAction{
			Function: "oasOpIdUnique",
		},
	}
}

// GetOperationSingleTagRule will check to see if an operation has more than a single tag
func GetOperationSingleTagRule() *model.Rule {
	return &model.Rule{
		Id:           operationSingularTag,
		Formats:      model.AllFormats,
		Description:  "Operation cannot have more than a single tag defined",
		Given:        "$",
		Resolved:     false,
		Recommended:  false,
		RuleCategory: model.RuleCategories[model.CategoryOperations],
		Type:         validation,
		Severity:     warn,
		Then: model.RuleAction{
			Function: "oasOpSingleTag",
		},
	}
}

// GetOAS2APIHostRule will check swagger specs for the host property being set.
func GetOAS2APIHostRule() *model.Rule {
	return &model.Rule{
		Id:           oas2APIHost,
		Formats:      model.OAS2Format,
		Description:  "OpenAPI 'host' must be present and a non-empty string.",
		Given:        "$",
		Resolved:     false,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryInfo],
		Type:         style,
		Severity:     info,
		Then: model.RuleAction{
			Field:    "host",
			Function: "truthy",
		},
	}
}

// GetOperationIdRule will check to make sure that operationIds  exist on all operations
func GetOperationIdRule() *model.Rule {
	return &model.Rule{
		Id:           operationOperationId,
		Formats:      model.AllFormats,
		Description:  "Every operation must contain an \"operationId\".",
		Given:        "$",
		Resolved:     false,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryOperations],
		Type:         validation,
		Severity:     err,
		Then: model.RuleAction{
			Function: "oasOpId",
		},
	}
}

// GetOperationSuccessResponseRule will check that every operation has a success response defined.
func GetOperationSuccessResponseRule() *model.Rule {
	return &model.Rule{
		Id:           operationSuccessResponse,
		Formats:      model.AllFormats,
		Description:  "Operation must have at least one 2xx or a 3xx response.",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryOperations],
		Type:         style,
		Severity:     warn,
		Then: model.RuleAction{
			Field:    "responses",
			Function: "oasOpSuccessResponse",
		},
	}
}

// GetDuplicatedEntryInEnumRule will check that enums used are not duplicates
func GetDuplicatedEntryInEnumRule() *model.Rule {
	return &model.Rule{
		Id:           duplicatedEntryInEnum,
		Formats:      model.AllFormats,
		Description:  "Enum values must not have duplicate entry",
		Given:        "$",
		Resolved:     false,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategorySchemas],
		Type:         validation,
		Severity:     err,
		Then: model.RuleAction{
			Function: "duplicatedEnum",
		},
	}
}

// GetNoRefSiblingsRule will check that there are no sibling nodes next to a $ref (which is technically invalid)
func GetNoRefSiblingsRule() *model.Rule {
	return &model.Rule{
		Id:           noRefSiblings,
		Formats:      model.AllFormats,
		Description:  "$ref values cannot be placed next to other properties (like a description)",
		Given:        "$",
		Resolved:     false,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategorySchemas],
		Type:         validation,
		Severity:     err,
		Then: model.RuleAction{
			Function: "refSiblings",
		},
	}
}

// GetOAS3UnusedComponentRule will check that there aren't any components anywhere that haven't been used.
func GetOAS3UnusedComponentRule() *model.Rule {
	return &model.Rule{
		Id:           oas3UnusedComponent,
		Formats:      model.OAS3AllFormat,
		Description:  "Check for unused components and bad references",
		Given:        "$",
		Resolved:     false,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategorySchemas],
		Type:         validation,
		Severity:     warn,
		Then: model.RuleAction{
			Function: "oasUnusedComponent",
		},
	}
}

// GetOAS2UnusedComponentRule will check that there aren't any components anywhere that haven't been used.
func GetOAS2UnusedComponentRule() *model.Rule {
	return &model.Rule{
		Id:           oas2UnusedDefinition,
		Formats:      model.OAS2Format,
		Description:  "Check for unused components and bad references",
		Given:        "$",
		Resolved:     false,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategorySchemas],
		Type:         validation,
		Severity:     warn,
		Then: model.RuleAction{
			Function: "oasUnusedComponent",
		},
	}
}

// GetOAS3SecurityDefinedRule will check that security definitions exist and validate for OpenAPI 3
func GetOAS3SecurityDefinedRule() *model.Rule {
	oasSecurityPath := make(map[string]string)
	oasSecurityPath["schemesPath"] = "$.components.securitySchemes"

	return &model.Rule{
		Id:           oas3OperationSecurityDefined,
		Formats:      model.OAS3AllFormat,
		Description:  "'security' values must match a scheme defined in components.securitySchemes",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategorySecurity],
		Type:         validation,
		Severity:     err,
		Then: model.RuleAction{
			Function:        "oasOpSecurityDefined",
			FunctionOptions: oasSecurityPath,
		},
	}
}

// GetOAS2SecurityDefinedRule will check that security definitions exist and validate for OpenAPI 2
func GetOAS2SecurityDefinedRule() *model.Rule {
	return &model.Rule{
		Id:           oas2OperationSecurityDefined,
		Formats:      model.OAS2Format,
		Description:  "'security' values must match a scheme defined in securityDefinitions",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategorySecurity],
		Type:         validation,
		Severity:     err,
		Then: model.RuleAction{
			Function: "oas2OpSecurityDefined",
		},
	}
}

// GetOAS2DiscriminatorRule will check swagger schemas to ensure they are using discriminations correctly.
func GetOAS2DiscriminatorRule() *model.Rule {
	return &model.Rule{
		Id:           oas2Discriminator,
		Formats:      model.OAS2Format,
		Description:  "discriminators are used correctly in schemas",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategorySchemas],
		Type:         validation,
		Severity:     err,
		Then: model.RuleAction{
			Function: "oasDiscriminator",
		},
	}
}

// GetOAS3ExamplesRule will check the entire spec for correct example use.
func GetOAS3ExamplesRule() *model.Rule {
	return &model.Rule{
		Id:           oas3ValidSchemaExample,
		Formats:      model.OAS3AllFormat,
		Description:  "Examples must be present and valid for operations and components",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryExamples],
		Type:         validation,
		Severity:     warn,
		Then: model.RuleAction{
			Function: "oasExample",
		},
	}
}

// GetOAS2ExamplesRule will check the entire spec for correct example use.
func GetOAS2ExamplesRule() *model.Rule {
	return &model.Rule{
		Id:           oas2ValidSchemaExample,
		Formats:      model.OAS2Format,
		Description:  "Examples must be present and valid for operations and components",
		Given:        "$",
		Resolved:     true,
		Recommended:  true,
		RuleCategory: model.RuleCategories[model.CategoryExamples],
		Type:         validation,
		Severity:     warn,
		Then: model.RuleAction{
			Function: "oasExample",
		},
	}
}
