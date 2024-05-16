package openapi

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"gopkg.in/yaml.v3"
)

// PolymorphicOneOf checks that there is no polymorphism used, in particular 'anyOf'
type PolymorphicOneOf struct {
}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the PolymorphicOneOf rule.
func (pm PolymorphicOneOf) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "polymorphic_oneOf",
	}
}

// GetCategory returns the category of the PolymorphicOneOf rule.
func (pm PolymorphicOneOf) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the PolymorphicOneOf rule, based on supplied context and a supplied []*yaml.Node slice.
func (pm PolymorphicOneOf) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	// no need to search! the index already has what we need.
	refs := context.Index.GetPolyOneOfReferences()

	for _, ref := range refs {
		results = append(results, model.RuleFunctionResult{
			Message:   fmt.Sprintf("`oneOf` polymorphic reference: %s", context.Rule.Description),
			StartNode: ref.Node,
			EndNode:   vacuumUtils.BuildEndNode(ref.Node),
			Path:      ref.Path,
			Rule:      context.Rule,
		})
	}

	return results
}
