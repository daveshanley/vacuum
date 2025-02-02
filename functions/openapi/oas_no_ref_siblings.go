package openapi

import (
	"strings"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
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

func notAllowedKeys(node *yaml.Node) []string {
	var keys []string

	for i := 0; i < len(node.Content); i += 2 {
		key := node.Content[i].Value
		switch key {
		case "$ref", "summary", "description":
			continue
		default:
			keys = append(keys, key)
		}
	}
	return keys
}

// RunRule will execute the OASNoRefSiblings rule, based on supplied context and a supplied []*yaml.Node slice.
func (nrs OASNoRefSiblings) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {

	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult
	siblings := context.Index.GetReferencesWithSiblings()
	for _, ref := range siblings {
		notAllowedKeys := notAllowedKeys(ref.Node)
		if len(notAllowedKeys) != 0 {
			key, val := utils.FindKeyNode("$ref", ref.Node.Content)
			results = append(results, model.RuleFunctionResult{
				Message:   "a `$ref` can only be placed next to `summary` and `description` but got:" + strings.Join(notAllowedKeys, " ,"),
				StartNode: key,
				EndNode:   vacuumUtils.BuildEndNode(val),
				Path:      ref.Path,
				Rule:      context.Rule,
			})
		}
	}
	return results
}
