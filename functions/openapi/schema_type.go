// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package openapi

import (
	"github.com/daveshanley/vacuum/functions/schemachecks"
	"github.com/daveshanley/vacuum/model"
	"go.yaml.in/yaml/v4"
)

// SchemaTypeCheck determines if document schemas contain compatible type, constraint and value definitions.
type SchemaTypeCheck struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the SchemaTypeCheck rule.
func (st SchemaTypeCheck) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "schemaTypeCheck",
	}
}

// GetCategory returns the category of the SchemaTypeCheck rule.
func (st SchemaTypeCheck) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule executes the SchemaTypeCheck rule against Doctor schemas built from libopenapi.
func (st SchemaTypeCheck) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	if context.DrDocument == nil {
		return nil
	}

	return schemachecks.RunTypeChecks(context.DrDocument.Schemas, context, schemachecks.TypeCheckOptions{
		AllowOAS30Nullable:          true,
		ValidateDependentRequired:   true,
		ValidateDiscriminator:       true,
		ValidateEnumConstRedundancy: true,
		ValidatePatterns:            true,
		ValidateValueCompatibility:  true,
	})
}
