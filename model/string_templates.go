package model

import (
	"fmt"
	"strconv"
	"strings"
)

// StringTemplates provides efficient string building for common patterns in vacuum.
// This replaces fmt.Sprintf usage to reduce allocation overhead.
type StringTemplates struct {
	// messageCache caches frequently used message patterns
	messageCache map[string]string
}

// clearCache clears the message cache (used for testing)
func (st *StringTemplates) clearCache() {
	if st.messageCache != nil {
		for k := range st.messageCache {
			delete(st.messageCache, k)
		}
	}
}

// getCachedMessage retrieves a cached message or builds and caches it
func (st *StringTemplates) getCachedMessage(key string, builder func() string) string {
	if st.messageCache == nil {
		st.messageCache = make(map[string]string, 50) // reasonable initial capacity
	}

	if msg, exists := st.messageCache[key]; exists {
		return msg
	}

	// Build and cache the message
	msg := builder()
	st.messageCache[key] = msg
	return msg
}

// NewStringTemplates creates a new StringTemplates instance
func NewStringTemplates() *StringTemplates {
	return &StringTemplates{}
}

// Global instance for reuse
var templates = NewStringTemplates()

// GetStringTemplates returns the global StringTemplates instance
func GetStringTemplates() *StringTemplates {
	return templates
}

// BuildFieldValidationMessage builds a validation error message for a field
// Pattern: "ruleMessage: `field` must be condition"
func (st *StringTemplates) BuildFieldValidationMessage(ruleMessage, field, condition string) string {
	var builder strings.Builder
	builder.Grow(len(ruleMessage) + len(field) + len(condition) + 10) // message + field + condition + formatting
	builder.WriteString(ruleMessage)
	builder.WriteString(": `")
	builder.WriteString(field)
	builder.WriteString("` must be ")
	builder.WriteString(condition)
	return builder.String()
}

// BuildFieldMustNotMessage builds a "must not" validation error message
// Pattern: "ruleMessage: `field` must not be condition"
func (st *StringTemplates) BuildFieldMustNotMessage(ruleMessage, field, condition string) string {
	var builder strings.Builder
	builder.Grow(len(ruleMessage) + len(field) + len(condition) + 14) // message + field + condition + " must not be "
	builder.WriteString(ruleMessage)
	builder.WriteString(": `")
	builder.WriteString(field)
	builder.WriteString("` must not be ")
	builder.WriteString(condition)
	return builder.String()
}

// BuildFieldMessage builds a general field message
// Pattern: "ruleMessage: `field` message"
func (st *StringTemplates) BuildFieldMessage(ruleMessage, field, message string) string {
	var builder strings.Builder
	builder.Grow(len(ruleMessage) + len(field) + len(message) + 6) // message + field + message + ": `" + "` "
	builder.WriteString(ruleMessage)
	builder.WriteString(": `")
	builder.WriteString(field)
	builder.WriteString("` ")
	builder.WriteString(message)
	return builder.String()
}

// BuildJSONPath builds a JSON path string
// Pattern: "base.field"
func (st *StringTemplates) BuildJSONPath(base, field string) string {
	var builder strings.Builder
	builder.Grow(len(base) + len(field) + 1) // base + field + "."
	builder.WriteString(base)
	builder.WriteByte('.')
	builder.WriteString(field)
	return builder.String()
}

// BuildArrayPath builds an array access path
// Pattern: "base[index]"
func (st *StringTemplates) BuildArrayPath(base string, index int) string {
	var builder strings.Builder
	indexStr := strconv.Itoa(index)
	builder.Grow(len(base) + len(indexStr) + 2) // base + index + "[]"
	builder.WriteString(base)
	builder.WriteByte('[')
	builder.WriteString(indexStr)
	builder.WriteByte(']')
	return builder.String()
}

// BuildQuotedPath builds a quoted field path
// Pattern: "base['field']"
func (st *StringTemplates) BuildQuotedPath(base, field string) string {
	var builder strings.Builder
	builder.Grow(len(base) + len(field) + 4) // base + field + "['']"
	builder.WriteString(base)
	builder.WriteString("['")
	builder.WriteString(field)
	builder.WriteString("']")
	return builder.String()
}

// BuildMissingRequiredMessage builds a missing required field message
// Pattern: "missing response code `code` for `method`"
func (st *StringTemplates) BuildMissingRequiredMessage(itemType, item, context string) string {
	var builder strings.Builder
	builder.Grow(len(itemType) + len(item) + len(context) + 15) // "missing " + itemType + " `" + item + "` for `" + context + "`"
	builder.WriteString("missing ")
	builder.WriteString(itemType)
	builder.WriteString(" `")
	builder.WriteString(item)
	builder.WriteString("` for `")
	builder.WriteString(context)
	builder.WriteByte('`')
	return builder.String()
}

// BuildBothDefinedMessage builds a message for XOR validation
// Pattern: "ruleMessage: `field1` and `field2` must not be both defined or undefined"
func (st *StringTemplates) BuildBothDefinedMessage(ruleMessage, field1, field2 string) string {
	var builder strings.Builder
	builder.Grow(len(ruleMessage) + len(field1) + len(field2) + 50) // rough estimate
	builder.WriteString(ruleMessage)
	builder.WriteString(": `")
	builder.WriteString(field1)
	builder.WriteString("` and `")
	builder.WriteString(field2)
	builder.WriteString("` must not be both defined or undefined")
	return builder.String()
}

// BuildAlphabeticalMessage builds an alphabetical ordering error message
// Pattern: "ruleMessage: `item1` must be placed before `item2` (alphabetical)"
func (st *StringTemplates) BuildAlphabeticalMessage(ruleMessage, item1, item2 string) string {
	var builder strings.Builder
	builder.Grow(len(ruleMessage) + len(item1) + len(item2) + 35) // rough estimate
	builder.WriteString(ruleMessage)
	builder.WriteString(": `")
	builder.WriteString(item1)
	builder.WriteString("` must be placed before `")
	builder.WriteString(item2)
	builder.WriteString("` (alphabetical)")
	return builder.String()
}

// BuildEnumValidationMessage builds an enum validation error message
// Pattern: "ruleMessage: `value` must equal to one of: [values]"
func (st *StringTemplates) BuildEnumValidationMessage(ruleMessage, value string, allowedValues interface{}) string {
	var builder strings.Builder
	valuesStr := fmt.Sprintf("%v", allowedValues)
	builder.Grow(len(ruleMessage) + len(value) + len(valuesStr) + 25) // rough estimate
	builder.WriteString(ruleMessage)
	builder.WriteString(": `")
	builder.WriteString(value)
	builder.WriteString("` must equal to one of: ")
	builder.WriteString(valuesStr)
	return builder.String()
}

// BuildPatternMessage builds a pattern validation error message
// Pattern: "ruleMessage: `value` does not match the expression `pattern`"
func (st *StringTemplates) BuildPatternMessage(ruleMessage, value, pattern string) string {
	var builder strings.Builder
	builder.Grow(len(ruleMessage) + len(value) + len(pattern) + 40) // rough estimate
	builder.WriteString(ruleMessage)
	builder.WriteString(": `")
	builder.WriteString(value)
	builder.WriteString("` does not match the expression `")
	builder.WriteString(pattern)
	builder.WriteByte('`')
	return builder.String()
}

// BuildRegexCompileErrorMessage builds a regex compilation error message
// Pattern: "ruleMessage: `pattern` cannot be compiled into a regular expression [`error`]"
func (st *StringTemplates) BuildRegexCompileErrorMessage(ruleMessage, pattern, errorMsg string) string {
	var builder strings.Builder
	builder.Grow(len(ruleMessage) + len(pattern) + len(errorMsg) + 60) // rough estimate
	builder.WriteString(ruleMessage)
	builder.WriteString(": `")
	builder.WriteString(pattern)
	builder.WriteString("` cannot be compiled into a regular expression [`")
	builder.WriteString(errorMsg)
	builder.WriteString("`]")
	return builder.String()
}

// BuildPatternMatchMessage builds a pattern match error message
// Pattern: "ruleMessage: matches the expression `pattern`"
func (st *StringTemplates) BuildPatternMatchMessage(ruleMessage, pattern string) string {
	var builder strings.Builder
	builder.Grow(len(ruleMessage) + len(pattern) + 25) // rough estimate
	builder.WriteString(ruleMessage)
	builder.WriteString(": matches the expression `")
	builder.WriteString(pattern)
	builder.WriteByte('`')
	return builder.String()
}

// BuildTypeErrorMessage builds a type validation error message
// Pattern: "ruleMessage: `value` is a type. errorMessage"
func (st *StringTemplates) BuildTypeErrorMessage(ruleMessage, value, typeDesc, errorMessage string) string {
	var builder strings.Builder
	builder.Grow(len(ruleMessage) + len(value) + len(typeDesc) + len(errorMessage) + 10) // rough estimate
	builder.WriteString(ruleMessage)
	builder.WriteString(": `")
	builder.WriteString(value)
	builder.WriteString("` is a ")
	builder.WriteString(typeDesc)
	builder.WriteString(". ")
	builder.WriteString(errorMessage)
	return builder.String()
}

// BuildNumericalOrderingMessage builds a numerical ordering error message
// Pattern: "ruleMessage: `value1` is less than `value2`, they need to be swapped (numerical ordering)"
func (st *StringTemplates) BuildNumericalOrderingMessage(ruleMessage, value1, value2 string) string {
	var builder strings.Builder
	builder.Grow(len(ruleMessage) + len(value1) + len(value2) + 60) // rough estimate
	builder.WriteString(ruleMessage)
	builder.WriteString(": `")
	builder.WriteString(value1)
	builder.WriteString("` is less than `")
	builder.WriteString(value2)
	builder.WriteString("`, they need to be swapped (numerical ordering)")
	return builder.String()
}

// BuildKebabCaseMessage builds a kebab-case validation message
// Pattern: "path segments `segments` do not use kebab-case"
func (st *StringTemplates) BuildKebabCaseMessage(segments string) string {
	var builder strings.Builder
	builder.Grow(len(segments) + 40) // rough estimate
	builder.WriteString("path segments `")
	builder.WriteString(segments)
	builder.WriteString("` do not use kebab-case")
	return builder.String()
}

// BuildUnknownSchemaTypeMessage builds an unknown schema type error message
// Pattern: "unknown schema type: `type`"
func (st *StringTemplates) BuildUnknownSchemaTypeMessage(schemaType string) string {
	var builder strings.Builder
	builder.Grow(len(schemaType) + 22) // "unknown schema type: `" + type + "`"
	builder.WriteString("unknown schema type: `")
	builder.WriteString(schemaType)
	builder.WriteByte('`')
	return builder.String()
}

// BuildHTTPVerbInPathMessage builds an HTTP verb in path error message
// Pattern: "path `path` contains an HTTP Verb `verb`"
func (st *StringTemplates) BuildHTTPVerbInPathMessage(path, verb string) string {
	var builder strings.Builder
	builder.Grow(len(path) + len(verb) + 35) // rough estimate
	builder.WriteString("path `")
	builder.WriteString(path)
	builder.WriteString("` contains an HTTP Verb `")
	builder.WriteString(verb)
	builder.WriteByte('`')
	return builder.String()
}

// BuildMissingExampleMessage builds a missing example error message
// Pattern: "media type schema property `propName` is missing `examples` or `example`"
func (st *StringTemplates) BuildMissingExampleMessage(propName string) string {
	var builder strings.Builder
	builder.Grow(len(propName) + 55) // rough estimate
	builder.WriteString("media type schema property `")
	builder.WriteString(propName)
	builder.WriteString("` is missing `examples` or `example`")
	return builder.String()
}

// BuildPropertyArrayPath builds a property array path
// Pattern: "base.property[index]"
func (st *StringTemplates) BuildPropertyArrayPath(base, property string, index int) string {
	var builder strings.Builder
	indexStr := strconv.Itoa(index)
	builder.Grow(len(base) + len(property) + len(indexStr) + 3) // base + "." + property + "[" + index + "]"
	builder.WriteString(base)
	builder.WriteByte('.')
	builder.WriteString(property)
	builder.WriteByte('[')
	builder.WriteString(indexStr)
	builder.WriteByte(']')
	return builder.String()
}

// BuildRequiredFieldMessage builds a required field error message
// Pattern: "`required` field `field` is not defined in `properties`"
func (st *StringTemplates) BuildRequiredFieldMessage(field string) string {
	var builder strings.Builder
	builder.Grow(len(field) + 45) // rough estimate
	builder.WriteString("`required` field `")
	builder.WriteString(field)
	builder.WriteString("` is not defined in `properties`")
	return builder.String()
}

// BuildOWASPResponseMessage builds an OWASP response validation message
// Pattern: "response with code `code`, must contain one of the defined headers: `headers`"
func (st *StringTemplates) BuildOWASPResponseMessage(code, headers string) string {
	var builder strings.Builder
	builder.Grow(len(code) + len(headers) + 65) // rough estimate
	builder.WriteString("response with code `")
	builder.WriteString(code)
	builder.WriteString("`, must contain one of the defined headers: `")
	builder.WriteString(headers)
	builder.WriteByte('`')
	return builder.String()
}

// BuildAPIKeyMessage builds an API key security message
// Pattern: "API keys must not be passed via URL parameters (`key`)"
func (st *StringTemplates) BuildAPIKeyMessage(key string) string {
	var builder strings.Builder
	builder.Grow(len(key) + 50) // rough estimate
	builder.WriteString("API keys must not be passed via URL parameters (`")
	builder.WriteString(key)
	builder.WriteString("`)")
	return builder.String()
}

// BuildCredentialsMessage builds a credentials security message
// Pattern: "URL parameters must not contain credentials, passwords, or secrets (`param`)"
func (st *StringTemplates) BuildCredentialsMessage(param string) string {
	var builder strings.Builder
	builder.Grow(len(param) + 70) // rough estimate
	builder.WriteString("URL parameters must not contain credentials, passwords, or secrets (`")
	builder.WriteString(param)
	builder.WriteString("`)")
	return builder.String()
}

// BuildSecurityDefinedMessage builds a security defined message
// Pattern: "`security` was not defined for path `path` in method `method`"
func (st *StringTemplates) BuildSecurityDefinedMessage(path, method string) string {
	var builder strings.Builder
	builder.Grow(len(path) + len(method) + 50) // rough estimate
	builder.WriteString("`security` was not defined for path `")
	builder.WriteString(path)
	builder.WriteString("` in method `")
	builder.WriteString(method)
	builder.WriteByte('`')
	return builder.String()
}

// BuildSecurityEmptyMessage builds a security empty message
// Pattern: "`security` is empty for path `path` in method `method`"
func (st *StringTemplates) BuildSecurityEmptyMessage(path, method string) string {
	var builder strings.Builder
	builder.Grow(len(path) + len(method) + 45) // rough estimate
	builder.WriteString("`security` is empty for path `")
	builder.WriteString(path)
	builder.WriteString("` in method `")
	builder.WriteString(method)
	builder.WriteByte('`')
	return builder.String()
}

// BuildSecurityNullElementsMessage builds a security null elements message
// Pattern: "`security` has null elements for path `path` in method `method`"
func (st *StringTemplates) BuildSecurityNullElementsMessage(path, method string) string {
	var builder strings.Builder
	builder.Grow(len(path) + len(method) + 55) // rough estimate
	builder.WriteString("`security` has null elements for path `")
	builder.WriteString(path)
	builder.WriteString("` in method `")
	builder.WriteString(method)
	builder.WriteByte('`')
	return builder.String()
}

// BuildCachedFieldValidationMessage builds a cached field validation message
// This caches common validation patterns to avoid rebuilding the same messages
func (st *StringTemplates) BuildCachedFieldValidationMessage(ruleMessage, field, condition string) string {
	// Create cache key for common patterns
	key := "field_validation:" + field + ":" + condition

	return st.getCachedMessage(key, func() string {
		return st.BuildFieldValidationMessage(ruleMessage, field, condition)
	})
}

// BuildCachedPatternMessage builds a cached pattern validation message
func (st *StringTemplates) BuildCachedPatternMessage(ruleMessage, value, pattern string) string {
	// Create cache key for common patterns
	key := "pattern:" + pattern

	return st.getCachedMessage(key, func() string {
		return st.BuildPatternMessage(ruleMessage, value, pattern)
	})
}
