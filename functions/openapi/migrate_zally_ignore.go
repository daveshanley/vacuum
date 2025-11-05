package openapi

import (
	"fmt"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"go.yaml.in/yaml/v4"
)

// MigrateZallyIgnore will check for x-zally-ignore keys and suggest migration to x-lint-ignore
type MigrateZallyIgnore struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the MigrateZallyIngore rule.
func (m MigrateZallyIgnore) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "migrateZallyIgnore",
	}
}

// GetCategory returns the category of the MigrateZallyIngore rule.
func (m MigrateZallyIgnore) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule will execute the MigrateZallyIngore rule, based on supplied context and a supplied []*yaml.Node slice.
func (m MigrateZallyIgnore) RunRule(
	nodes []*yaml.Node,
	context model.RuleFunctionContext,
) []model.RuleFunctionResult {
	if len(nodes) <= 0 {
		return nil
	}

	var results []model.RuleFunctionResult

	for _, node := range nodes {
		m.checkNodeWithPath(node, "$", &results, context)
	}

	return results
}

func (m MigrateZallyIgnore) checkNodeWithPath(
	node *yaml.Node,
	currentPath string,
	results *[]model.RuleFunctionResult,
	context model.RuleFunctionContext,
) {
	if node == nil {
		return
	}

	switch node.Kind {
	case yaml.DocumentNode:
		for _, content := range node.Content {
			m.checkNodeWithPath(content, currentPath, results, context)
		}
	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			keyPath := currentPath + "." + keyNode.Value
			if currentPath == "$" {
				keyPath = "$." + keyNode.Value
			}

			if keyNode.Value == "x-zally-ignore" {
				*results = append(*results, model.RuleFunctionResult{
					Message:   "Convert ignore rules to use x-lint-ignore",
					StartNode: keyNode,
					EndNode:   utils.BuildEndNode(keyNode),
					Path:      keyPath,
					Rule:      context.Rule,
				})
			}

			// Recursively check the value node
			m.checkNodeWithPath(valueNode, keyPath, results, context)
		}

	case yaml.SequenceNode:
		for i, item := range node.Content {
			itemPath := fmt.Sprintf("%s[%d]", currentPath, i)
			m.checkNodeWithPath(item, itemPath, results, context)
		}
	}
}
