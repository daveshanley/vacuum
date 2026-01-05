package model

import (
	_ "embed"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v4"
)

const (
	OAS2  = "oas2"
	OAS3  = "oas3"   // family format - matches all 3.x versions
	OAS30 = "oas3_0" // exact 3.0 only - does not match 3.1 or 3.2
	OAS31 = "oas3_1"
	OAS32 = "oas3_2"
)

var OAS3_1Format = []string{OAS31}
var OAS3_2Format = []string{OAS32}
var AllExceptOAS3_1 = []string{OAS2, OAS30} // uses OAS30 to avoid matching 3.1
var OAS3Format = []string{OAS3}
var OAS3AllFormat = []string{OAS3, OAS31, OAS32}
var OAS2Format = []string{OAS2}
var AllFormats = []string{OAS3, OAS31, OAS32, OAS2}

// FormatMatches checks if a rule format matches a spec format.
// The oas3 format is treated as a "family" that covers oas3, oas3_1, and oas3_2.
// This allows rules with `formats: [oas3]` to match OpenAPI 3.0, 3.1, and 3.2 specs,
// which matches Spectral's behavior.
// The oas3_0 format is an exact match for OpenAPI 3.0 only (does not match 3.1 or 3.2).
func FormatMatches(ruleFormat, specFormat string) bool {
	if ruleFormat == specFormat {
		return true
	}
	// oas3 is a family format that matches all 3.x versions
	if ruleFormat == OAS3 && (specFormat == OAS31 || specFormat == OAS32) {
		return true
	}
	// oas3_0 (exact 3.0) matches when spec is detected as oas3 (which means 3.0)
	if ruleFormat == OAS30 && specFormat == OAS3 {
		return true
	}
	return false
}

const WebsiteUrl = "https://quobix.com/vacuum"
const GithubUrl = "https://github.com/daveshanley/vacuum"

// buildResultMessage efficiently builds a result message without fmt.Sprintf
func buildResultMessage(key, message string, value interface{}) string {
	var builder strings.Builder
	// Pre-allocate reasonable capacity to avoid reallocations
	builder.Grow(len(key) + len(message) + 20) // key + message + quotes + value estimate
	builder.WriteByte('\'')
	builder.WriteString(key)
	builder.WriteString("' ")
	builder.WriteString(message)
	builder.WriteString(" '")
	builder.WriteString(fmt.Sprintf("%v", value)) // Keep fmt for interface{} conversion
	builder.WriteByte('\'')
	return builder.String()
}

// Simple JSONPath builder for basic path construction
type JSONPathBuilder struct {
	segments []string
}

// GetJSONPathBuilder returns a simple JSONPath builder
func GetJSONPathBuilder() *JSONPathBuilder {
	return &JSONPathBuilder{
		segments: make([]string, 0, 10),
	}
}

// Reset clears the builder
func (b *JSONPathBuilder) Reset() *JSONPathBuilder {
	b.segments = b.segments[:0]
	return b
}

// Root starts a JSONPath
func (b *JSONPathBuilder) Root() *JSONPathBuilder {
	b.segments = append(b.segments, "$")
	return b
}

// Field adds a field to the path
func (b *JSONPathBuilder) Field(field string) *JSONPathBuilder {
	b.segments = append(b.segments, ".", field)
	return b
}

// Key adds a key to the path
func (b *JSONPathBuilder) Key(key string) *JSONPathBuilder {
	b.segments = append(b.segments, "['", key, "']")
	return b
}

// Index adds an index to the path
func (b *JSONPathBuilder) Index(index int) *JSONPathBuilder {
	b.segments = append(b.segments, "[", strconv.Itoa(index), "]")
	return b
}

// Build constructs the JSONPath
func (b *JSONPathBuilder) Build() string {
	var builder strings.Builder
	for _, segment := range b.segments {
		builder.WriteString(segment)
	}
	return builder.String()
}

// BuildOperationFieldPath builds a path for operation fields
func BuildOperationFieldPath(path, method, field string) string {
	return fmt.Sprintf("$.paths['%s'].%s.%s", path, method, field)
}

// BuildResponsePath builds a path for responses
func BuildResponsePath(path, method, code string) string {
	return fmt.Sprintf("$.paths['%s'].%s.responses['%s']", path, method, code)
}

// buildResultMessageWithDescription efficiently builds a result message with description
func buildResultMessageWithDescription(desc, key, message string, value interface{}) string {
	var builder strings.Builder
	// Pre-allocate reasonable capacity
	builder.Grow(len(desc) + len(key) + len(message) + 25) // desc + key + message + separators + value estimate
	builder.WriteString(desc)
	builder.WriteString(": '")
	builder.WriteString(key)
	builder.WriteString("' ")
	builder.WriteString(message)
	builder.WriteString(" '")
	builder.WriteString(fmt.Sprintf("%v", value)) // Keep fmt for interface{} conversion
	builder.WriteByte('\'')
	return builder.String()
}

// BuildFunctionResult will create a RuleFunctionResult from a key, message and value.
// Deprecated: use BuildFunctionResultWithDescription instead.
func BuildFunctionResult(key, message string, value interface{}) RuleFunctionResult {
	return RuleFunctionResult{
		Message: buildResultMessage(key, message, value),
	}
}

// BuildPooledFunctionResult will create a RuleFunctionResult from the pool for better performance.
// The caller is responsible for returning the result to the pool when done.
func BuildPooledFunctionResult(key, message string, value interface{}) *RuleFunctionResult {
	result := GetPooledRuleFunctionResult()
	result.Message = buildResultMessage(key, message, value)
	return result
}

// BuildFunctionResultWithDescription will create a RuleFunctionResult from a description, key, message and value.
func BuildFunctionResultWithDescription(desc, key, message string, value interface{}) RuleFunctionResult {
	return RuleFunctionResult{
		Message: buildResultMessageWithDescription(desc, key, message, value),
	}
}

// BuildPooledFunctionResultWithDescription will create a RuleFunctionResult from the pool for better performance.
// The caller is responsible for returning the result to the pool when done.
func BuildPooledFunctionResultWithDescription(desc, key, message string, value interface{}) *RuleFunctionResult {
	result := GetPooledRuleFunctionResult()
	result.Message = buildResultMessageWithDescription(desc, key, message, value)
	return result
}

// BuildFunctionResultString will create a RuleFunctionResult from a string already parsed into a message.
func BuildFunctionResultString(message string) RuleFunctionResult {
	return RuleFunctionResult{
		Message: message,
	}
}

// extractOptionKeys extracts top-level keys from various option types.
// supports map[string]interface{}, map[string]string, and []interface{} containing maps.
func extractOptionKeys(options interface{}) []string {
	switch opts := options.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(opts))
		for k := range opts {
			keys = append(keys, k)
		}
		return keys
	case map[string]string:
		keys := make([]string, 0, len(opts))
		for k := range opts {
			keys = append(keys, k)
		}
		return keys
	case []interface{}:
		var keys []string
		for _, v := range opts {
			if m, ok := v.(map[string]interface{}); ok {
				for k := range m {
					keys = append(keys, k)
				}
			}
		}
		return keys
	default:
		return nil
	}
}

// optionKeyMatchesProperty checks if an option key matches a schema property.
// supports exact matches and prefix matches for dot-notation properties.
// e.g., "schema" matches "schema", "separator" matches "separator.char"
func optionKeyMatchesProperty(optionKey, propName string) bool {
	if optionKey == propName {
		return true
	}
	// option key is parent of dot-notation property
	return strings.HasPrefix(propName, optionKey+".")
}

// findInvalidOptionKeys returns option keys that don't match any schema property.
func findInvalidOptionKeys(keys []string, properties []RuleFunctionProperty) []string {
	var invalid []string
	for _, k := range keys {
		found := false
		for _, prop := range properties {
			if optionKeyMatchesProperty(k, prop.Name) {
				found = true
				break
			}
		}
		if !found {
			invalid = append(invalid, k)
		}
	}
	return invalid
}

// ValidateRuleFunctionContextAgainstSchema will perform run-time validation against a rule to ensure that
// options being passed in are acceptable and meet the needs of the Rule schema.
// This validates top-level option keys only, allowing nested objects (like JSON schemas) as values.
func ValidateRuleFunctionContextAgainstSchema(ruleFunction RuleFunction, ctx RuleFunctionContext) (bool, []string) {

	valid := true
	var errs []string
	schema := ruleFunction.GetSchema()
	numProps := 0

	if options, ok := ctx.Options.(map[string]interface{}); ok {
		numProps = countPropsInterface(options, numProps)
	}

	if options, ok := ctx.Options.(map[string]string); ok {
		numProps = countPropsString(options, numProps)
	}

	if options, ok := ctx.Options.(map[string][]string); ok {
		numProps = len(options)
	}

	if schema.MinProperties > 0 && numProps < schema.MinProperties {
		valid = false
		errs = append(errs, fmt.Sprintf("%s: minimum property number not met (%v)",
			schema.ErrorMessage, schema.MinProperties))
	}

	if schema.MaxProperties > 0 && numProps > schema.MaxProperties {
		valid = false
		errs = append(errs, fmt.Sprintf("%s: maximum number (%v) of properties exceeded. '%v' provided.",
			schema.ErrorMessage, schema.MaxProperties, numProps))
	}

	// check if the function requires a field or not, and check if it's been set
	if schema.RequiresField && ctx.RuleAction.Field == "" {
		errs = append(errs, fmt.Sprintf("'%s' requires a 'field' value to be set", schema.Name))
	}

	// check if this schema has required properties.
	// we check the original options structure (not flattened) because some functions
	// like 'schema' expect complex nested objects as option values. Flattening is only used
	// at runtime via GetOptionsStringMap() for functions that need flat key-value pairs.
	if len(schema.Required) > 0 {
		optionKeys := extractOptionKeys(ctx.Options)
		var missingProps []string
		for _, req := range schema.Required {
			found := false
			for _, k := range optionKeys {
				if k == req {
					found = true
					break
				}
			}
			if !found {
				missingProps = append(missingProps, req)
			}
		}
		if len(missingProps) > 0 {
			valid = false
			for _, mProp := range missingProps {
				errs = append(errs, fmt.Sprintf("%s: missing required property: %s (%s)",
					schema.ErrorMessage, mProp, schema.GetPropertyDescription(mProp)))
			}
		}
	}

	// check if the values submitted exist as properties.
	// we check top-level keys and support two patterns:
	// 1. key matches a property exactly (e.g., "schema" matches property "schema")
	// 2. key is a prefix of a dot-notation property (e.g., "separator" matches "separator.char")
	// this allows both nested objects as values and Spectral's nested YAML format.
	if len(schema.Properties) > 0 {
		optionKeys := extractOptionKeys(ctx.Options)
		invalidKeys := findInvalidOptionKeys(optionKeys, schema.Properties)
		for _, k := range invalidKeys {
			valid = false
			errs = append(errs, fmt.Sprintf("%s: property '%s' is not a valid property for '%s'",
				schema.ErrorMessage, k, schema.Name))
		}
	}

	return valid, errs
}

func countPropsInterface(options map[string]interface{}, numProps int) int {
	for _, v := range options {
		if stringVal, ok := v.(string); ok {
			if strings.Contains(stringVal, ",") {
				split := strings.Split(stringVal, ",")
				numProps += len(split)
			} else {
				numProps++
			}
		}
		if _, ok := v.(int); ok {
			numProps++
		}
		if _, ok := v.(float64); ok {
			numProps++
		}
		if _, ok := v.(bool); ok {
			numProps++
		}
		if arr, ok := v.([]interface{}); ok {
			numProps += len(arr)
		}
	}
	return numProps
}

func countPropsString(options map[string]string, numProps int) int {
	for _, v := range options {
		if strings.Contains(v, ",") {
			split := strings.Split(v, ",")
			numProps += len(split)
		} else {
			numProps++
		}
	}
	return numProps
}

// CastToRuleAction is a utility function to cast an unknown structure into a RuleAction.
// useful for when building rules or testing out concepts.
func CastToRuleAction(action interface{}) *RuleAction {
	if action == nil {
		return nil
	}
	if ra, ok := action.(*RuleAction); ok {
		return ra
	}
	return nil
}

// MapPathAndNodesToResults will map the same start/end nodes with the same path.
func MapPathAndNodesToResults(path string, startNode, endNode *yaml.Node, results []RuleFunctionResult) []RuleFunctionResult {
	var mapped []RuleFunctionResult
	for _, result := range results {
		result.Path = path
		result.StartNode = startNode
		result.EndNode = endNode
		mapped = append(mapped, result)
	}

	return mapped
}

// CompileRegex attempts to compile the provided `Pattern` from the ruleset. If it fails, returns nil
// and adds an error to the result set. Any rule using this should then return the results if there is no *Regexp
// returned.
func CompileRegex(context RuleFunctionContext, pattern string, results *[]RuleFunctionResult) *regexp.Regexp {
	compiledRegex, err := regexp.Compile(pattern)
	if err != nil {
		*results = append(*results, RuleFunctionResult{
			Message: fmt.Sprintf("Error: cannot run rule, pattern `%s` cannot be compiled", pattern),
			Rule:    context.Rule,
		})
		return nil
	}
	return compiledRegex
}
