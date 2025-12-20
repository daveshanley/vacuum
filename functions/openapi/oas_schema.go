// Copyright 2022-2023 Dave Shanley / Quobix
// Princess Beef Heavy Industries LLC
// SPDX-License-Identifier: MIT

package openapi

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/libopenapi-validator/errors"
	"github.com/pb33f/libopenapi-validator/schema_validation"
	"github.com/pb33f/libopenapi/utils"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"go.yaml.in/yaml/v4"
)

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

	// grab the original bytes and the spec info from context.
	info := context.SpecInfo

	if info.SpecType == "" {
		// spec type is un-known, there is no point in running this rule.
		return results
	}

	// Always check if the document can be marshaled to JSON ourselves
	// Don't depend on info.SpecJSON as it may be nil or in an inconsistent state
	// This catches issues like maps with non-string keys that would cause validation to fail
	var marshalingIssues []vacuumUtils.MarshalingIssue

	if info.RootNode != nil && info.SpecBytes != nil {
		// Decode the YAML ourselves to check for marshaling issues
		var yamlData interface{}
		decoder := yaml.NewDecoder(strings.NewReader(string(*info.SpecBytes)))
		if err := decoder.Decode(&yamlData); err == nil {
			// Check if it can marshal to JSON
			marshalingIssues = vacuumUtils.CheckJSONMarshaling(yamlData, info.RootNode)
			// Clear the decoded data to free memory
			yamlData = nil
		} else {
			// If we can't decode the YAML, check the AST directly for issues
			marshalingIssues = vacuumUtils.FindMarshalingIssuesInYAML(info.RootNode)
		}
	}

	if len(marshalingIssues) > 0 {
		// Report all marshaling issues
		for _, issue := range marshalingIssues {
			n := &yaml.Node{
				Line:   issue.Line,
				Column: issue.Column,
			}

			result := model.RuleFunctionResult{
				Message:   fmt.Sprintf("schema invalid: cannot marshal: %s", issue.Reason),
				StartNode: n,
				EndNode:   vacuumUtils.BuildEndNode(n),
				Path:      issue.Path,
				Rule:      context.Rule,
			}
			results = append(results, result)
		}
		// Return marshaling errors without attempting schema validation
		// since it would fail with unhelpful "got null, want object" errors
		return results
	}

	// use libopenapi-validator
	valid, validationErrors := schema_validation.ValidateOpenAPIDocument(context.Document)

	// For OpenAPI 3.1+, check for nullable keyword usage which is not allowed
	version := context.Document.GetSpecInfo().VersionNumeric
	if version >= 3.1 {
		nullableResults := checkForNullableKeyword(context)
		results = append(results, nullableResults...)
	}

	if valid {
		return results // Return any nullable violations even if document is otherwise valid
	}

	// duplicates are possible, so we need to de-dupe them.
	seen := make(map[string]*errors.SchemaValidationFailure)
	for i := range validationErrors {
		for y := range validationErrors[i].SchemaValidationErrors {
			// skip, seen it.
			if _, ok := seen[hashResult(validationErrors[i].SchemaValidationErrors[y])]; ok {
				continue
			}
			if validationErrors[i].SchemaValidationErrors[y].Reason == "if-else failed" {
				continue
			}
			if validationErrors[i].SchemaValidationErrors[y].Reason == "if-then failed" {
				continue
			}
			_, location := utils.ConvertComponentIdIntoFriendlyPathSearch(validationErrors[i].SchemaValidationErrors[y].Location)
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
			if schemaErr.OriginalError != nil {
				leafErrors := extractLeafValidationErrors(schemaErr.OriginalError)
				if len(leafErrors) > 0 {
					// Limit to last 3 leaf errors for readability
					if len(leafErrors) > 3 {
						leafErrors = leafErrors[len(leafErrors)-3:]
					}
					reason = strings.Join(leafErrors, "; ")
				}
			}

			// Fallback to Reason if no leaf errors extracted
			if reason == "" {
				reason = schemaErr.Reason
			}

			// Final fallback
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
			seen[hashResult(validationErrors[i].SchemaValidationErrors[y])] = validationErrors[i].SchemaValidationErrors[y]
		}
	}
	return results
}

func hashResult(sve *errors.SchemaValidationFailure) string {
	return fmt.Sprintf("%x",
		sha256.Sum256([]byte(fmt.Sprintf("%s:%d:%d:%s", sve.Location, sve.Line, sve.Column, sve.Reason))))
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

// extractLeafValidationErrors extracts only meaningful leaf error messages from a jsonschema.ValidationError tree (issue #766)
// It filters out noise from oneOf/anyOf schema structures (like "missing properties: [$ref]")
func extractLeafValidationErrors(err *jsonschema.ValidationError) []string {
	var highPriority []string  // Type errors, pattern errors, format errors - the real issues
	var lowPriority []string   // Enum errors (often noise from oneOf branches)
	seen := make(map[string]bool)
	seenPaths := make(map[string]bool) // Track paths we've already reported errors for
	const maxDepth = 50

	var extract func(e *jsonschema.ValidationError, depth int)
	extract = func(e *jsonschema.ValidationError, depth int) {
		if e == nil || depth > maxDepth {
			return
		}

		// if this is a leaf node (no causes), extract the message
		if len(e.Causes) == 0 {
			path := "/" + strings.Join(e.InstanceLocation, "/")

			// Skip noise errors from oneOf/anyOf schema structures
			if isNoiseError(e.ErrorKind) {
				return
			}

			msg := errorKindToString(e.ErrorKind)
			if msg != "" {
				fullMsg := fmt.Sprintf("`%s` %s", path, msg)
				if !seen[fullMsg] {
					seen[fullMsg] = true

					// Categorize by priority
					if isHighPriorityError(e.ErrorKind) {
						highPriority = append(highPriority, fullMsg)
						seenPaths[path] = true
					} else if !seenPaths[path] {
						// Only add low priority if we haven't seen this path in high priority
						lowPriority = append(lowPriority, fullMsg)
					}
				}
			}
		}

		// recurse into causes
		for _, cause := range e.Causes {
			extract(cause, depth+1)
		}
	}

	extract(err, 0)

	// Return high priority errors first, then low priority
	if len(highPriority) > 0 {
		return highPriority
	}
	return lowPriority
}

// isNoiseError returns true for errors that are noise from oneOf/anyOf schema structures
func isNoiseError(ek jsonschema.ErrorKind) bool {
	if ek == nil {
		return true
	}
	switch k := ek.(type) {
	case *kind.Required:
		// "missing properties: [$ref]" is noise from oneOf branches in OpenAPI schemas
		for _, missing := range k.Missing {
			if missing == "$ref" {
				return true
			}
		}
	case *kind.FalseSchema:
		// "property not allowed" from false schema branches
		return true
	}
	return false
}

// isHighPriorityError returns true for errors that represent the actual root cause
func isHighPriorityError(ek jsonschema.ErrorKind) bool {
	if ek == nil {
		return false
	}
	switch ek.(type) {
	case *kind.Type:
		return true // Wrong type is usually the real issue
	case *kind.Pattern:
		return true // Pattern mismatch is specific
	case *kind.Format:
		return true // Format error is specific
	case *kind.Const:
		return true // Const mismatch is specific
	case *kind.AdditionalProperties:
		return true // Extra properties is specific
	case *kind.Required:
		return true // Missing required field (that isn't $ref)
	}
	return false
}

// errorKindToString converts a jsonschema ErrorKind to a human-readable string
func errorKindToString(ek jsonschema.ErrorKind) string {
	if ek == nil {
		return ""
	}
	switch k := ek.(type) {
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
		return fmt.Sprintf("%v", ek)
	}
}
