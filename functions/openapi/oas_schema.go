// Copyright 2022-2023 Dave Shanley / Quobix
// Princess Beef Heavy Industries LLC
// SPDX-License-Identifier: MIT

package openapi

import (
	"crypto/sha256"
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/helpers"
	"github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/libopenapi-validator/errors"
	"github.com/pb33f/libopenapi-validator/schema_validation"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"strings"
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

	// use libopenapi-validator
	valid, validationErrors := schema_validation.ValidateOpenAPIDocument(context.Document)
	if valid {
		return nil
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

			var reason = validationErrors[i].SchemaValidationErrors[y].Reason
			if reason == "validation failed" { // this is garbage, so let's look into the original error stack.

				var helpfulMessages []string

				// dive into the validation error and pull out something more meaningful!
				helpers.DiveIntoValidationError(validationErrors[i].SchemaValidationErrors[y].OriginalError, &helpfulMessages,
					strings.TrimPrefix(validationErrors[i].SchemaValidationErrors[y].Location, "/"))

				// run through the helpful messages and remove any duplicates
				seenMessages := make(map[string]string)
				for _, message := range helpfulMessages {
					h := helpers.HashString(message)
					if _, exists := seenMessages[h]; !exists {
						seenMessages[h] = message
					}
				}
				var cleanedMessages []string
				for _, message := range seenMessages {
					cleanedMessages = append(cleanedMessages, message)
				}

				// wrap the root cause in a string, with a welcoming tone.
				// define a list of join phrases
				joinPhrases := []string{
					". Also, ",
					", and ",
					". And also ",
					", as well as ",
				}
				// pick a random join phrase using a random index
				reasonBuf := strings.Builder{}
				r := 0
				for q, message := range cleanedMessages {
					if r > 3 {
						r = 0
					}
					reasonBuf.WriteString(message)
					if q < len(cleanedMessages)-1 {
						reasonBuf.WriteString(joinPhrases[r])
					}
					r++
				}
				reason = reasonBuf.String()
			}
			if reason == "" {
				reason = "multiple components failed validation"
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
