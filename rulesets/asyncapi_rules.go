// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package rulesets

import (
	"fmt"

	"github.com/daveshanley/vacuum/model"
)

const asyncAPILatestVersion = "3.1.0"

// GenerateDefaultAsyncAPIRuleSet returns all built-in AsyncAPI rules. The rules
// are scoped to AsyncAPI 3.x only; AsyncAPI 2.x is intentionally unsupported.
func GenerateDefaultAsyncAPIRuleSet() *RuleSet {
	return &RuleSet{
		DocumentationURI: "https://quobix.com/vacuum/rulesets/asyncapi",
		Formats:          model.AsyncAPI3AllFormats,
		Description:      "All built-in AsyncAPI rules supported by vacuum.",
		Rules:            GetAllAsyncAPIRules(),
		Extends:          map[string]string{VacuumAsyncAPI: VacuumAll},
	}
}

// GetAllAsyncAPIRules returns every built-in AsyncAPI rule.
func GetAllAsyncAPIRules() map[string]*model.Rule {
	rules := GetAsyncAPIRecommendedRules()
	rules[AsyncAPIInfoLicenseURL] = asyncAPITruthyRule(AsyncAPIInfoLicenseURL, "Check AsyncAPI license URL", "License object must include `url`.", "$", "info.license.url", model.SeverityInfo, false, model.CategoryInfo)
	rules[AsyncAPI3ServerNotExampleCom] = asyncAPIPatternRule(AsyncAPI3ServerNotExampleCom, "Check AsyncAPI server host is not example.com", "Server host must not point at example.com.", "$.servers.*", "host", "", "example\\.com", model.SeverityInfo, false, model.CategoryValidation)
	rules[AsyncAPI3TagDescription] = asyncAPITruthyRule(AsyncAPI3TagDescription, "Check AsyncAPI tag descriptions", "Tag object must have `description`.", "$.tags[*]", "description", model.SeverityInfo, false, model.CategoryTags)
	rules[AsyncAPI3TagsAlphabetical] = asyncAPIRule(AsyncAPI3TagsAlphabetical, "Check AsyncAPI tags are alphabetical", "AsyncAPI tags must be ordered alphabetically.", "$", "tags", "alphabetical", map[string]string{"keyedBy": "name"}, model.SeverityInfo, false, model.CategoryTags)
	return rules
}

// GetAsyncAPIRecommendedRules returns the recommended AsyncAPI 3.x rule set.
func GetAsyncAPIRecommendedRules() map[string]*model.Rule {
	return map[string]*model.Rule{
		AsyncAPI3DocumentResolved:          asyncAPIDocumentRule(AsyncAPI3DocumentResolved, "Check resolved AsyncAPI v3 document structure", true),
		AsyncAPI3DocumentUnresolved:        asyncAPIDocumentRule(AsyncAPI3DocumentUnresolved, "Check unresolved AsyncAPI v3 document structure", false),
		AsyncAPI3ChannelNoEmptyParameter:   asyncAPIPatternRule(AsyncAPI3ChannelNoEmptyParameter, "Check AsyncAPI channel address parameters are not empty", "Channel address must not have empty parameter substitution pattern.", "$.channels.*", "address", "", "\\{\\}", model.SeverityError, true, model.CategoryValidation),
		AsyncAPI3ChannelNoQueryNorFragment: asyncAPIPatternRule(AsyncAPI3ChannelNoQueryNorFragment, "Check AsyncAPI channel address has no query or fragment", "Channel address must not include query or fragment delimiters.", "$.channels.*", "address", "", "[\\?#]", model.SeverityError, true, model.CategoryValidation),
		AsyncAPI3ChannelNoTrailingSlash:    asyncAPIPatternRule(AsyncAPI3ChannelNoTrailingSlash, "Check AsyncAPI channel address has no trailing slash", "Channel address must not end with slash.", "$.channels.*", "address", "", ".+\\/$", model.SeverityError, true, model.CategoryValidation),
		AsyncAPIChannelParameters:          asyncAPICustomRule(AsyncAPIChannelParameters, "Check AsyncAPI channel parameters", "Channel parameters must be defined and there must be no redundant parameters.", []string{"$.channels.*", "$.components.channels.*"}, "asyncApiChannelParameters", nil, model.SeverityError, true, model.CategoryValidation),
		AsyncAPI3ChannelServers:            asyncAPICustomRule(AsyncAPI3ChannelServers, "Check AsyncAPI channel server references", "Channel servers must be defined in the `servers` object.", "$.channels.*", "asyncApiChannelServers", nil, model.SeverityError, true, model.CategoryValidation),
		AsyncAPI3HeadersSchemaTypeObject:   asyncAPIHeadersSchemaRule(),
		AsyncAPIInfoContactProperties:      asyncAPIContactPropertiesRule(),
		AsyncAPIInfoContact:                asyncAPITruthyRule(AsyncAPIInfoContact, "Check AsyncAPI contact object", "Info object must have `contact` object.", "$", "info.contact", model.SeverityError, true, model.CategoryInfo),
		AsyncAPIInfoDescription:            asyncAPITruthyRule(AsyncAPIInfoDescription, "Check AsyncAPI info description", "Info `description` must be present and non-empty.", "$", "info.description", model.SeverityError, true, model.CategoryInfo),
		AsyncAPIInfoLicense:                asyncAPITruthyRule(AsyncAPIInfoLicense, "Check AsyncAPI license object", "Info object must have `license` object.", "$", "info.license", model.SeverityError, true, model.CategoryInfo),
		AsyncAPILatestVersion:              asyncAPILatestVersionRule(),
		AsyncAPI3OperationDescription:      asyncAPITruthyRule(AsyncAPI3OperationDescription, "Check AsyncAPI operation descriptions", "Operation `description` must be present and non-empty.", "$.operations.*", "description", model.SeverityError, true, model.CategoryOperations),
		AsyncAPI3OperationSecurity:         asyncAPICustomRule(AsyncAPI3OperationSecurity, "Check AsyncAPI operation security", "Operation security must reference defined security schemes.", "$.operations.*.security.*", "asyncApiSecurity", map[string]string{"objectType": "Operation"}, model.SeverityError, true, model.CategorySecurity),
		AsyncAPIParameterDescription:       asyncAPITruthyRule(AsyncAPIParameterDescription, "Check AsyncAPI parameter descriptions", "Parameter objects must have `description`.", []string{"$.components.parameters.*", "$.channels.*.parameters.*"}, "description", model.SeverityWarn, true, model.CategoryDescriptions),
		AsyncAPI3PayloadUnsupportedFormat:  asyncAPIRule(AsyncAPI3PayloadUnsupportedFormat, "Check AsyncAPI payload schema formats", "Message schema validation is only supported with default unspecified `schemaFormat`.", []string{"$.components.messages.*", "$.channels.*.messages.*"}, "schemaFormat", "undefined", nil, model.SeverityInfo, true, model.CategorySchemas),
		AsyncAPI3ServerNoEmptyVariable:     asyncAPIPatternRule(AsyncAPI3ServerNoEmptyVariable, "Check AsyncAPI server variables are not empty", "Server host and pathname must not have empty variable substitution pattern.", []string{"$.servers.*.host", "$.servers.*.pathname", "$.components.servers.*.host", "$.components.servers.*.pathname"}, "", "", "\\{\\}", model.SeverityError, true, model.CategoryValidation),
		AsyncAPI3ServerNoTrailingSlash:     asyncAPIPatternRule(AsyncAPI3ServerNoTrailingSlash, "Check AsyncAPI server host or pathname has no trailing slash", "Server host and pathname must not end with slash.", []string{"$.servers.*.host", "$.servers.*.pathname", "$.components.servers.*.host", "$.components.servers.*.pathname"}, "", "", "\\/$", model.SeverityError, true, model.CategoryValidation),
		AsyncAPIServers:                    asyncAPIServersRule(),
		AsyncAPI3TagsUniqueness:            asyncAPICustomRule(AsyncAPI3TagsUniqueness, "Check AsyncAPI tag uniqueness", "Each tag must have a unique name.", []string{"$.tags", "$.servers.*.tags", "$.components.servers.*.tags", "$.operations.*.tags", "$.components.operations.*.tags", "$.components.operationTraits.*.tags", "$.channels.*.messages.*.tags", "$.components.channels.*.messages.*.tags", "$.components.messages.*.tags", "$.components.messageTraits.*.tags"}, "asyncApiTagsUnique", nil, model.SeverityError, true, model.CategoryTags),
		AsyncAPI3Tags:                      asyncAPITruthyRule(AsyncAPI3Tags, "Check AsyncAPI tags", "AsyncAPI document must have non-empty tags array.", "$", "tags", model.SeverityError, true, model.CategoryTags),
		AsyncAPIServerVariables:            asyncAPICustomRule(AsyncAPIServerVariables, "Check AsyncAPI server variables", "Server variables must be defined and there must be no redundant variables.", []string{"$.servers.*", "$.components.servers.*"}, "asyncApiServerVariables", nil, model.SeverityError, true, model.CategoryValidation),
		AsyncAPIServerSecurity:             asyncAPICustomRule(AsyncAPIServerSecurity, "Check AsyncAPI server security", "Server security must reference defined security schemes.", "$.servers.*.security.*", "asyncApiSecurity", map[string]string{"objectType": "Server"}, model.SeverityError, true, model.CategorySecurity),
		AsyncAPIOperationChannel:           asyncAPICustomRule(AsyncAPIOperationChannel, "Check AsyncAPI operation channels", "Operations must reference declared channels.", "$.operations.*", "asyncApiOperationChannel", nil, model.SeverityError, true, model.CategoryOperations),
		AsyncAPIOperationMessages:          asyncAPICustomRule(AsyncAPIOperationMessages, "Check AsyncAPI operation messages", "Operations must reference declared messages.", "$.operations.*", "asyncApiOperationMessages", nil, model.SeverityError, true, model.CategoryOperations),
		AsyncAPIOperationReply:             asyncAPICustomRule(AsyncAPIOperationReply, "Check AsyncAPI operation replies", "Operation replies must reference declared channels and messages.", "$.operations.*", "asyncApiOperationReply", nil, model.SeverityError, true, model.CategoryOperations),
		AsyncAPIMessageExamples:            asyncAPICustomRule(AsyncAPIMessageExamples, "Check AsyncAPI message examples", "Message examples should define payload or headers.", []string{"$.components.messages.*", "$.channels.*.messages.*", "$.components.messageTraits.*"}, "asyncApiMessageExamples", nil, model.SeverityWarn, true, model.CategoryExamples),
		AsyncAPIUnusedComponents:           asyncAPICustomRule(AsyncAPIUnusedComponents, "Check AsyncAPI unused components", "Reusable AsyncAPI components should be referenced.", "$", "asyncApiUnusedComponents", nil, model.SeverityWarn, true, model.CategorySchemas),
		AsyncAPIContentType:                asyncAPICustomRule(AsyncAPIContentType, "Check AsyncAPI content types", "Messages with payloads should define a content type or use document defaultContentType.", []string{"$.components.messages.*", "$.channels.*.messages.*"}, "asyncApiContentType", nil, model.SeverityWarn, true, model.CategoryValidation),
	}
}

func asyncAPIDocumentRule(id, name string, resolved bool) *model.Rule {
	rule := asyncAPICustomRule(id, name, "AsyncAPI v3 document structure must be valid.", "$", "asyncApiDocument", map[string]bool{"resolved": resolved}, model.SeverityError, true, model.CategoryValidation)
	rule.Resolved = resolved
	return rule
}

func asyncAPIHeadersSchemaRule() *model.Rule {
	return asyncAPIRule(AsyncAPI3HeadersSchemaTypeObject, "Check AsyncAPI headers schema type", "Headers schema type must be `object`.", []string{"$.components.messageTraits.*.headers", "$.components.messages.*.headers", "$.channels.*.messages.*.headers", "$.channels.*.messages.*.traits[*].headers"}, "", "schema", map[string]any{
		"allErrors": true,
		"schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"type": map[string]any{"enum": []string{"object"}},
			},
			"required": []string{"type"},
		},
	}, model.SeverityError, true, model.CategorySchemas)
}

func asyncAPIContactPropertiesRule() *model.Rule {
	return asyncAPIRule(AsyncAPIInfoContactProperties, "Check AsyncAPI contact properties", "Contact object must have `name`, `url` and `email`.", "$.info.contact", "", []model.RuleAction{
		{Field: "name", Function: "truthy"},
		{Field: "url", Function: "truthy"},
		{Field: "email", Function: "truthy"},
	}, nil, model.SeverityError, true, model.CategoryInfo)
}

func asyncAPILatestVersionRule() *model.Rule {
	return asyncAPIRule(AsyncAPILatestVersion, "Check AsyncAPI latest version", "The latest AsyncAPI version should be used.", "$.asyncapi", "", "schema", map[string]any{
		"schema": map[string]any{"const": asyncAPILatestVersion},
	}, model.SeverityInfo, true, model.CategoryValidation)
}

func asyncAPIServersRule() *model.Rule {
	return asyncAPIRule(AsyncAPIServers, "Check AsyncAPI servers", "AsyncAPI object must have non-empty `servers` object.", "$", "servers", "schema", map[string]any{
		"allErrors": true,
		"schema": map[string]any{
			"type":          "object",
			"minProperties": 1,
		},
	}, model.SeverityError, true, model.CategoryValidation)
}

func asyncAPITruthyRule(id, name, description string, given any, field string, severity string, recommended bool, category string) *model.Rule {
	return asyncAPIRule(id, name, description, given, field, "truthy", nil, severity, recommended, category)
}

func asyncAPIPatternRule(id, name, description string, given any, field, match, notMatch string, severity string, recommended bool, category string) *model.Rule {
	options := make(map[string]string)
	if match != "" {
		options["match"] = match
	}
	if notMatch != "" {
		options["notMatch"] = notMatch
	}
	return asyncAPIRule(id, name, description, given, field, "pattern", options, severity, recommended, category)
}

func asyncAPICustomRule(id, name, description string, given any, function string, options any, severity string, recommended bool, category string) *model.Rule {
	return asyncAPIRule(id, name, description, given, "", function, asyncAPIBatchOptions(options), severity, recommended, category)
}

func asyncAPIBatchOptions(options any) map[string]interface{} {
	merged := make(map[string]interface{})
	switch typed := options.(type) {
	case nil:
	case map[string]interface{}:
		for key, value := range typed {
			merged[key] = value
		}
	case map[string]string:
		for key, value := range typed {
			merged[key] = value
		}
	case map[string]bool:
		for key, value := range typed {
			merged[key] = value
		}
	case map[interface{}]interface{}:
		for key, value := range typed {
			merged[fmt.Sprint(key)] = value
		}
	default:
		merged["options"] = typed
	}
	merged["batch"] = true
	return merged
}

func asyncAPIRule(id, name, description string, given any, field string, function any, options any, severity string, recommended bool, category string) *model.Rule {
	then := function
	if fn, ok := function.(string); ok {
		then = model.RuleAction{
			Field:           field,
			Function:        fn,
			FunctionOptions: options,
		}
	}
	return &model.Rule{
		Name:         name,
		Id:           id,
		Formats:      model.AsyncAPI3AllFormats,
		Description:  description,
		Given:        given,
		Resolved:     true,
		Recommended:  recommended,
		RuleCategory: model.RuleCategories[category],
		Type:         Validation,
		Severity:     severity,
		Then:         then,
	}
}
