// Copyright 2022-2023 Dave Shanley / Quobix
// Princess Beef Heavy Industries LLC
// SPDX-License-Identifier: MIT

package openapi

import (
	"crypto/sha256"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
	"sync"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/libopenapi-validator/config"
	"github.com/pb33f/libopenapi-validator/errors"
	"github.com/pb33f/libopenapi-validator/helpers"
	"github.com/pb33f/libopenapi-validator/schema_validation"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/utils"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"go.yaml.in/yaml/v4"
)

// ErrorClassification represents the semantic category of a validation error
type ErrorClassification int

const (
	ErrorClassNoise        ErrorClassification = iota // Filter out - noise from oneOf/anyOf branches
	ErrorClassLowPriority                             // Show if no high priority errors exist
	ErrorClassHighPriority                            // Always show - represents actual validation issues
)

// ErrorContext captures the semantic context needed to interpret validation errors
type ErrorContext struct {
	SchemaURL        string
	InstanceLocation []string
	ErrorKind        jsonschema.ErrorKind
	SchemaKeyword    string // Extracted from SchemaURL: "unevaluatedProperties", "additionalProperties", etc.
}

// InstancePath returns the instance location as a slash-delimited path (computed lazily)
func (ec ErrorContext) InstancePath() string {
	return "/" + strings.Join(ec.InstanceLocation, "/")
}

// NewErrorContext extracts semantic context from a ValidationError
func NewErrorContext(err *jsonschema.ValidationError) ErrorContext {
	ctx := ErrorContext{
		SchemaURL:        err.SchemaURL,
		InstanceLocation: err.InstanceLocation,
		ErrorKind:        err.ErrorKind,
	}
	ctx.SchemaKeyword = extractSchemaKeyword(err.SchemaURL)
	return ctx
}

// extractSchemaKeyword extracts the last path component from a schema URL
// e.g., "file:///.../schema#/$defs/paths/unevaluatedProperties" -> "unevaluatedProperties"
func extractSchemaKeyword(schemaURL string) string {
	if idx := strings.LastIndex(schemaURL, "/"); idx != -1 && idx < len(schemaURL)-1 {
		return schemaURL[idx+1:]
	}
	return ""
}

// String returns a string representation of the classification for debugging
func (ec ErrorClassification) String() string {
	switch ec {
	case ErrorClassNoise:
		return "Noise"
	case ErrorClassLowPriority:
		return "LowPriority"
	case ErrorClassHighPriority:
		return "HighPriority"
	default:
		return "Unknown"
	}
}

// isConstraintViolation checks if an ErrorKind represents a concrete constraint violation
func isConstraintViolation(ek jsonschema.ErrorKind) bool {
	switch ek.(type) {
	case *kind.Type, *kind.Pattern, *kind.Format, *kind.Const,
		*kind.AdditionalProperties,
		*kind.MinLength, *kind.MaxLength, *kind.Minimum, *kind.Maximum,
		*kind.MinItems, *kind.MaxItems, *kind.MinProperties, *kind.MaxProperties,
		*kind.UniqueItems, *kind.PropertyNames, *kind.MultipleOf:
		return true
	}
	return false
}

// ClassifyError determines how to handle an error based on full context
func ClassifyError(ctx ErrorContext) ErrorClassification {
	if ctx.ErrorKind == nil {
		return ErrorClassNoise
	}

	switch k := ctx.ErrorKind.(type) {
	case *kind.FalseSchema:
		return classifyFalseSchema(ctx)
	case *kind.Required:
		return classifyRequired(k)
	case *kind.Enum:
		return ErrorClassLowPriority
	default:
		if isConstraintViolation(ctx.ErrorKind) {
			return ErrorClassHighPriority
		}
		return ErrorClassLowPriority
	}
}

// classifyFalseSchema determines if a FalseSchema error is signal or noise
func classifyFalseSchema(ctx ErrorContext) ErrorClassification {
	// FalseSchema from constraint keywords = real error (property doesn't match allowed patterns)
	switch ctx.SchemaKeyword {
	case "unevaluatedProperties", "additionalProperties":
		return ErrorClassHighPriority
	default:
		// FalseSchema from oneOf/anyOf branches = noise
		return ErrorClassNoise
	}
}

// classifyRequired determines if a Required error is signal or noise
func classifyRequired(k *kind.Required) ErrorClassification {
	// "missing properties: [$ref]" is noise from oneOf branches in OpenAPI schemas
	for _, missing := range k.Missing {
		if missing == "$ref" {
			return ErrorClassNoise
		}
	}
	return ErrorClassHighPriority
}

// ErrorFormatter generates human-readable messages for validation errors
type ErrorFormatter interface {
	Format(ctx ErrorContext) string
}

// DefaultErrorFormatter provides generic jsonschema error formatting
type DefaultErrorFormatter struct{}

// Format converts ErrorKind to a readable string
func (f DefaultErrorFormatter) Format(ctx ErrorContext) string {
	if ctx.ErrorKind == nil {
		return ""
	}
	switch k := ctx.ErrorKind.(type) {
	case *kind.Required:
		return fmt.Sprintf("missing properties: `%v`", k.Missing)
	case *kind.AdditionalProperties:
		return fmt.Sprintf("additional properties not allowed: `%v`", k.Properties)
	case *kind.Type:
		return fmt.Sprintf("expected type `%v`, got `%v`", k.Want, k.Got)
	case *kind.Enum:
		return fmt.Sprintf("value must be one of: `%v`", k.Want)
	case *kind.FalseSchema:
		return "property not allowed"
	case *kind.Pattern:
		return fmt.Sprintf("does not match pattern `%s`", k.Want)
	case *kind.MinLength:
		return fmt.Sprintf("length must be >= `%d`", k.Want)
	case *kind.MaxLength:
		return fmt.Sprintf("length must be <= `%d`", k.Want)
	case *kind.Minimum:
		return fmt.Sprintf("must be >= `%v`", k.Want)
	case *kind.Maximum:
		return fmt.Sprintf("must be <= `%v`", k.Want)
	case *kind.MinItems:
		return fmt.Sprintf("must have >= `%d` items", k.Want)
	case *kind.MaxItems:
		return fmt.Sprintf("must have <= `%d` items", k.Want)
	case *kind.MinProperties:
		return fmt.Sprintf("must have >= `%d` properties", k.Want)
	case *kind.MaxProperties:
		return fmt.Sprintf("must have <= `%d` properties", k.Want)
	case *kind.Const:
		return fmt.Sprintf("must be `%v`", k.Want)
	case *kind.Format:
		return fmt.Sprintf("invalid format: expected `%s`", k.Want)
	default:
		return fmt.Sprintf("%v", ctx.ErrorKind)
	}
}

// OpenAPIErrorFormatter provides OpenAPI-aware error formatting with context-specific messages
type OpenAPIErrorFormatter struct {
	DefaultErrorFormatter
}

// Format generates a context-aware error message with OpenAPI-specific handling
func (f OpenAPIErrorFormatter) Format(ctx ErrorContext) string {
	if _, ok := ctx.ErrorKind.(*kind.FalseSchema); ok {
		return f.formatFalseSchema(ctx)
	}
	return f.DefaultErrorFormatter.Format(ctx)
}

// formatFalseSchema generates specific messages for FalseSchema errors based on context
func (f OpenAPIErrorFormatter) formatFalseSchema(ctx ErrorContext) string {
	if len(ctx.InstanceLocation) >= 2 && ctx.InstanceLocation[0] == "paths" {
		pathKey := ctx.InstanceLocation[1]
		if !strings.HasPrefix(pathKey, "/") && !strings.HasPrefix(pathKey, "x-") {
			return fmt.Sprintf("path `%s` is invalid: paths must begin with `/` or `x-`", pathKey)
		}
		return fmt.Sprintf("path `%s` is not allowed", pathKey)
	}

	if ctx.SchemaKeyword == "unevaluatedProperties" && len(ctx.InstanceLocation) > 0 {
		property := ctx.InstanceLocation[len(ctx.InstanceLocation)-1]
		return fmt.Sprintf("property `%s` is not allowed", property)
	}

	return f.DefaultErrorFormatter.Format(ctx)
}

// oasSchemaCache caches compiled OAS JSON Schemas keyed by version number.
// There are at most 4 entries (2.0, 3.0, 3.1, 3.2) so a sync.Map is ideal.
var oasSchemaCache sync.Map

func getOrCompileOASSchema(apiSchema string, version float32) (*jsonschema.Schema, error) {
	if cached, ok := oasSchemaCache.Load(version); ok {
		return cached.(*jsonschema.Schema), nil
	}
	options := config.NewValidationOptions()
	compiled, err := helpers.NewCompiledSchemaWithVersion("schema", []byte(apiSchema), options, version)
	if err != nil {
		return nil, err
	}
	oasSchemaCache.Store(version, compiled)
	return compiled, nil
}

// OASSchema  will check that the document is a valid OpenAPI schema.
type OASSchema struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the OASSchema rule.
func (os OASSchema) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "oasSchema",
	}
}

// GetCategory returns the category of the OASSchema rule.
func (os OASSchema) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the OASSchema rule, based on supplied context and a supplied []*yaml.Node slice.
func (os OASSchema) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult
	info := context.SpecInfo

	if info == nil {
		return results
	}
	if info.SpecType == "" {
		// spec type is un-known, there is no point in running this rule.
		return results
	}

	// Early return when no JSON representation is available (e.g. turbo mode with
	// SkipJSONConversion). Newer libopenapi builds the JSON view lazily; use its
	// accessors when present and fall back to the released eager fields otherwise.
	validationInfo := info
	if context.Document != nil && context.Document.GetSpecInfo() != nil {
		validationInfo = context.Document.GetSpecInfo()
	}

	// libopenapi-validator v0.13.x still reads the deprecated SpecInfo fields
	// directly, so hydrate the same SpecInfo instance it will validate.
	specJSON, specJSONBytes, specJSONErr := specJSONForRead(validationInfo)
	if specJSON == nil && (specJSONBytes == nil || len(*specJSONBytes) == 0) {
		// a nil JSON view means either conversion was intentionally skipped
		// (no error) or the document cannot be represented as JSON at all -
		// surface the failure instead of silently skipping schema validation.
		if specJSONErr != nil {
			n := &yaml.Node{Line: 1, Column: 0}
			if info.RootNode != nil {
				n = info.RootNode
			}
			results = append(results, model.RuleFunctionResult{
				Message:   fmt.Sprintf("schema invalid: document cannot be converted to JSON: %s", specJSONErr.Error()),
				StartNode: n,
				EndNode:   vacuumUtils.BuildEndNode(n),
				Path:      "$",
				Rule:      context.Rule,
			})
		}
		return results
	}

	// Use a cached compiled schema when possible to avoid recompiling the OAS schema every invocation
	compiledSchema, _ := getOrCompileOASSchema(info.APISchema, info.VersionNumeric)
	valid, validationErrors := schema_validation.ValidateOpenAPIDocumentWithPrecompiled(
		context.Document, compiledSchema)

	// For OpenAPI 3.1+, check for nullable keyword usage which is not allowed
	version := validationInfo.VersionNumeric
	if version >= 3.1 {
		nullableResults := checkForNullableKeyword(context)
		results = append(results, nullableResults...)
	}

	if valid {
		return results // Return any nullable violations even if document is otherwise valid
	}

	// validation errors can appear multiple times across different schema branches; deduplicate by hash
	seenDocumentResults := make(map[string]struct{})
	seenSchemaFailures := make(map[string]*errors.SchemaValidationFailure)
	for i := range validationErrors {
		if len(validationErrors[i].SchemaValidationErrors) == 0 {
			res := buildDocumentValidationResult(validationErrors[i], context.Rule, validationInfo.RootNode)
			hash := hashDocumentValidationResult(&res)
			if _, ok := seenDocumentResults[hash]; ok {
				continue
			}
			results = append(results, res)
			seenDocumentResults[hash] = struct{}{}
			addResultToModelByLine(&res, context.DrDocument, validationErrors[i].SpecLine+1)
			continue
		}
		for y := range validationErrors[i].SchemaValidationErrors {
			if _, ok := seenSchemaFailures[hashResult(validationErrors[i].SchemaValidationErrors[y])]; ok {
				continue
			}
			if validationErrors[i].SchemaValidationErrors[y].Reason == "if-else failed" {
				continue
			}
			if validationErrors[i].SchemaValidationErrors[y].Reason == "if-then failed" {
				continue
			}
			_, location := utils.ConvertComponentIdIntoFriendlyPathSearch(validationErrors[i].SchemaValidationErrors[y].KeywordLocation)
			n := &yaml.Node{
				Line:   validationErrors[i].SchemaValidationErrors[y].Line,
				Column: validationErrors[i].SchemaValidationErrors[y].Column,
			}
			var allPaths []string
			var modelByLine []v3.Foundational
			var modelErr error
			if context.DrDocument != nil {
				modelByLine, modelErr = context.DrDocument.LocateModelByLine(validationErrors[i].SchemaValidationErrors[y].Line + 1)
				if modelErr == nil {
					if modelByLine != nil && len(modelByLine) >= 1 {
						allPaths = append(allPaths, location)
						location = modelByLine[0].GenerateJSONPath()
						for j := 0; j < len(modelByLine); j++ {
							allPaths = append(allPaths, modelByLine[j].GenerateJSONPath())
						}
					}
				}
			}

			schemaErr := validationErrors[i].SchemaValidationErrors[y]
			var reason string

			// Always prefer leaf error extraction from OriginalError when available (issue #766)
			if schemaErr.OriginalJsonSchemaError != nil {
				leafErrors := extractLeafValidationErrors(schemaErr.OriginalJsonSchemaError)
				if len(leafErrors) > 0 {
					// Limit to last 3 leaf errors for readability
					if len(leafErrors) > 3 {
						leafErrors = leafErrors[len(leafErrors)-3:]
					}
					reason = strings.Join(leafErrors, "; ")
				}
			}

			if reason == "" {
				reason = schemaErr.Reason
			}

			// guard against empty reason from upstream validator
			if reason == "" {
				reason = "schema validation failed"
			}
			res := model.RuleFunctionResult{
				Message:   fmt.Sprintf("schema invalid: %v", reason),
				StartNode: n,
				EndNode:   vacuumUtils.BuildEndNode(n),
				Path:      location,
				Rule:      context.Rule,
			}
			if len(allPaths) > 1 {
				res.Paths = allPaths
			}
			results = append(results, res)
			if len(modelByLine) > 0 {
				if arr, ok := modelByLine[0].(v3.AcceptsRuleResults); ok {
					arr.AddRuleFunctionResult(v3.ConvertRuleResult(&res))
				}
			}
			seenSchemaFailures[hashResult(validationErrors[i].SchemaValidationErrors[y])] = validationErrors[i].SchemaValidationErrors[y]
		}
	}
	return results
}

type modelByLineLocator interface {
	LocateModelByLine(line int) ([]v3.Foundational, error)
}

func addResultToModelByLine(result *model.RuleFunctionResult, locator modelByLineLocator, line int) {
	if result == nil || locator == nil {
		return
	}
	if line < 1 {
		line = 1
	}
	modelByLine, err := locator.LocateModelByLine(line)
	if err != nil || len(modelByLine) == 0 {
		return
	}
	if arr, ok := modelByLine[0].(v3.AcceptsRuleResults); ok {
		arr.AddRuleFunctionResult(v3.ConvertRuleResult(result))
	}
}

func buildDocumentValidationResult(validationError *errors.ValidationError, rule *model.Rule, rootNode *yaml.Node) model.RuleFunctionResult {
	line := validationError.SpecLine
	if line == 0 {
		line = 1
	}
	n := &yaml.Node{
		Line:   line,
		Column: validationError.SpecCol,
	}

	reason := validationError.Reason
	if reason == "" {
		reason = validationError.Message
	}
	if reason == "" {
		reason = "schema validation failed"
	}

	return model.RuleFunctionResult{
		Message:   fmt.Sprintf("schema invalid: %v", reason),
		StartNode: n,
		EndNode:   vacuumUtils.BuildEndNode(n),
		Path:      validationErrorResultPath(validationError, rootNode),
		Rule:      rule,
	}
}

func validationErrorResultPath(validationError *errors.ValidationError, rootNode *yaml.Node) string {
	if validationError == nil {
		return "$"
	}
	if pointer, ok := validationError.Context.(string); ok {
		return jsonPointerToJSONPath(pointer, rootNode)
	}
	return "$"
}

func jsonPointerToJSONPath(pointer string, rootNode *yaml.Node) string {
	if pointer == "" || !strings.HasPrefix(pointer, "/") {
		return "$"
	}

	path := "$"
	node := documentContentNode(rootNode)
	for _, segment := range strings.Split(pointer[1:], "/") {
		segment = strings.ReplaceAll(strings.ReplaceAll(segment, "~1", "/"), "~0", "~")
		if node != nil && node.Kind == yaml.SequenceNode {
			if index, err := strconv.Atoi(segment); err == nil && index >= 0 {
				path += fmt.Sprintf("[%d]", index)
				if index < len(node.Content) {
					node = node.Content[index]
				} else {
					node = nil
				}
				continue
			}
		}
		if isJSONPathIdentifier(segment) {
			path += "." + segment
		} else {
			path += "['" + escapeJSONPathSegment(segment) + "']"
		}
		node = mappingValueNode(node, segment)
	}
	return path
}

func escapeJSONPathSegment(segment string) string {
	segment = strings.ReplaceAll(segment, "\\", "\\\\")
	return strings.ReplaceAll(segment, "'", "\\'")
}

func documentContentNode(node *yaml.Node) *yaml.Node {
	if node != nil && node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return node.Content[0]
	}
	return node
}

func mappingValueNode(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func isJSONPathIdentifier(segment string) bool {
	if segment == "" {
		return false
	}
	for i, r := range segment {
		valid := r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || i > 0 && r >= '0' && r <= '9'
		if !valid {
			return false
		}
	}
	return true
}

type specJSONReader interface {
	GetSpecJSON() *map[string]interface{}
	GetSpecJSONBytes() *[]byte
	GetSpecJSONError() error
}

func specJSONForRead(info *datamodel.SpecInfo) (*map[string]interface{}, *[]byte, error) {
	if info == nil {
		return nil, nil, nil
	}
	// TODO: remove this compatibility shim once libopenapi-validator reads
	// SpecInfo through the lazy JSON accessors directly.
	if reader, ok := any(info).(specJSONReader); ok {
		return reader.GetSpecJSON(), reader.GetSpecJSONBytes(), reader.GetSpecJSONError()
	}
	return info.SpecJSON, info.SpecJSONBytes, nil
}

func hashResult(sve *errors.SchemaValidationFailure) string {
	return fmt.Sprintf("%x",
		sha256.Sum256([]byte(fmt.Sprintf("%s:%d:%d:%s", sve.KeywordLocation, sve.Line, sve.Column, sve.Reason))))
}

func hashDocumentValidationResult(res *model.RuleFunctionResult) string {
	if res == nil {
		return ""
	}
	line := 0
	column := 0
	if res.StartNode != nil {
		line = res.StartNode.Line
		column = res.StartNode.Column
	}
	return fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%s:%d:%d:%s", res.Path, line, column, res.Message))))
}

// checkForNullableKeyword searches for nullable keyword usage in OpenAPI 3.1+ documents
func checkForNullableKeyword(context model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult

	if context.DrDocument == nil {
		return results
	}

	// Check all schemas for nullable keyword
	if context.DrDocument.Schemas != nil {
		for _, schema := range context.DrDocument.Schemas {
			if schema.Value != nil && schema.Value.Nullable != nil && *schema.Value.Nullable {
				// Found nullable keyword in OpenAPI 3.1+ schema
				result := model.RuleFunctionResult{
					Message:   "The `nullable` keyword is not supported in OpenAPI 3.1. Use `type: ['string', 'null']` instead.",
					StartNode: schema.Value.GoLow().Nullable.KeyNode,
					EndNode:   vacuumUtils.BuildEndNode(schema.Value.GoLow().Nullable.KeyNode),
					Path:      schema.GenerateJSONPath() + ".nullable",
					Rule:      context.Rule,
				}
				results = append(results, result)
			}
		}
	}

	return results
}

// defaultFormatter is a package-level singleton to avoid repeated allocations
var defaultFormatter = OpenAPIErrorFormatter{}

// hashInstanceLocation creates a fast hash of an instance location for deduplication
func hashInstanceLocation(s []string) uint64 {
	h := fnv.New64a()
	for _, str := range s {
		h.Write([]byte(str))
		h.Write([]byte{0})
	}
	return h.Sum64()
}

// addUniqueError appends a formatted error message if not already seen
func addUniqueError(errors *[]string, seen map[string]bool, ctx ErrorContext) {
	msg := defaultFormatter.Format(ctx)
	fullMsg := "`" + ctx.InstancePath() + "` " + msg
	if !seen[fullMsg] {
		seen[fullMsg] = true
		*errors = append(*errors, fullMsg)
	}
}

// extractLeafValidationErrors extracts meaningful validation errors from a jsonschema error tree.
// It recursively walks to leaf nodes, classifies them by priority, and filters noise.
// Returns high-priority errors if any exist, otherwise low-priority errors.
// When high-priority errors exist, ALL low-priority errors are suppressed to reduce noise.
// Addresses issues #524 (path validation), #766 (oneOf/anyOf noise).
func extractLeafValidationErrors(err *jsonschema.ValidationError) []string {
	highPriority := make([]string, 0, 10)
	lowPriority := make([]string, 0, 5)
	seen := make(map[string]bool, 100)
	seenPaths := make(map[uint64]bool, 50)
	// prevent stack overflow from deeply nested schemas
	const maxDepth = 50

	var extract func(e *jsonschema.ValidationError, depth int)
	extract = func(e *jsonschema.ValidationError, depth int) {
		if e == nil || depth > maxDepth {
			return
		}

		if len(e.Causes) == 0 {
			ctx := NewErrorContext(e)
			classification := ClassifyError(ctx)

			if classification == ErrorClassNoise {
				return
			}

			pathHash := hashInstanceLocation(ctx.InstanceLocation)

			// seenPaths prevents low-priority errors for paths that already have high-priority errors
			if classification == ErrorClassHighPriority {
				if !seenPaths[pathHash] {
					addUniqueError(&highPriority, seen, ctx)
					seenPaths[pathHash] = true
				}
			} else if !seenPaths[pathHash] {
				addUniqueError(&lowPriority, seen, ctx)
			}
		}

		for _, cause := range e.Causes {
			extract(cause, depth+1)
		}
	}

	extract(err, 0)

	if len(highPriority) > 0 {
		return highPriority
	}
	return lowPriority
}

// errorKindToString converts a jsonschema ErrorKind to a human-readable string.
// Deprecated: Use DefaultErrorFormatter.Format instead.
func errorKindToString(ek jsonschema.ErrorKind) string {
	return DefaultErrorFormatter{}.Format(ErrorContext{ErrorKind: ek})
}
