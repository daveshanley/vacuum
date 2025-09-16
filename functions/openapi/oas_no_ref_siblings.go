package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"go.yaml.in/yaml/v4"
)

// OASNoRefSiblings validates that no properties other than `description` and `summary` are added alongside a `$ref`.
// This rule helps ensure that only essential properties are attached to `$ref` nodes, preventing unnecessary and unused additions.
type OASNoRefSiblings struct {
}

// GetCategory returns the category of the OASNoRefSiblings rule.
func (nrs OASNoRefSiblings) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the OASNoRefSiblings rule.
func (nrs OASNoRefSiblings) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "oasRefSiblings",
	}
}

// RunRule will execute the OASNoRefSiblings rule, based on supplied context and a supplied []*yaml.Node slice.
func (nrs OASNoRefSiblings) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	// In OpenAPI 3.1+ this rule is no longer useful, as libopenapi now handles this internally by correctly handling $ref nodes siblings
	// any siblings are supported by the model and can be accessed via the model, so this rule is redundant.
	//
	// keeping it here for historical purposes, but it will always return no results.
	//
	// https://github.com/pb33f/libopenapi/issues/90
	// https://github.com/pb33f/libopenapi/blob/main/document_test.go#L1675
	return []model.RuleFunctionResult{}
}
