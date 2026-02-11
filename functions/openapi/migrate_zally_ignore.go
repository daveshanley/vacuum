package openapi

import (
	"strconv"
	"strings"

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
	// pre-allocate path segments slice to avoid per-recursion allocations
	segs := make([]string, 0, 16)
	segs = append(segs, "$")

	for _, node := range nodes {
		m.checkNodeWithPath(node, segs, &results, context)
	}

	return results
}

// buildPath joins path segments into a JSONPath string like "$.foo.bar[0].baz"
func buildPath(segs []string) string {
	if len(segs) <= 1 {
		return "$"
	}
	var b strings.Builder
	b.Grow(len(segs) * 8) // rough estimate
	b.WriteString(segs[0])
	for _, s := range segs[1:] {
		if len(s) > 0 && s[0] == '[' {
			b.WriteString(s) // array index, no dot
		} else {
			b.WriteByte('.')
			b.WriteString(s)
		}
	}
	return b.String()
}

func (m MigrateZallyIgnore) checkNodeWithPath(
	node *yaml.Node,
	segs []string,
	results *[]model.RuleFunctionResult,
	context model.RuleFunctionContext,
) {
	if node == nil {
		return
	}

	switch node.Kind {
	case yaml.DocumentNode:
		for _, content := range node.Content {
			m.checkNodeWithPath(content, segs, results, context)
		}
	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			childSegs := append(segs, keyNode.Value)

			if keyNode.Value == "x-zally-ignore" {
				*results = append(*results, model.RuleFunctionResult{
					Message:   "Convert ignore rules to use x-lint-ignore",
					StartNode: keyNode,
					EndNode:   utils.BuildEndNode(keyNode),
					Path:      buildPath(childSegs),
					Rule:      context.Rule,
				})
			}

			m.checkNodeWithPath(valueNode, childSegs, results, context)
		}

	case yaml.SequenceNode:
		for i, item := range node.Content {
			childSegs := append(segs, "["+strconv.Itoa(i)+"]")
			m.checkNodeWithPath(item, childSegs, results, context)
		}
	}
}
