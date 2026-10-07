package motor

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/go-yaml"
)

// formatRuleMessages runs once for each function result, before auto-fixes can
// change its value. Replacement text is never interpreted as another template.
func formatRuleMessages(rule *model.Rule, action model.RuleAction, selected []*yaml.Node, results []model.RuleFunctionResult, root *yaml.Node, paths *vacuumUtils.NodePathIndex) {
	for i := range results {
		result := &results[i]
		result.Rule = rule
		if !strings.Contains(rule.Message, "{{") {
			result.Message = rule.Message
			continue
		}
		path := result.Path
		target := result.StartNode
		fallback := path == "" || path == "unknown" || resultPathHasSelectorSyntax(path)
		for _, given := range resultGivenPaths(rule) {
			fallback = fallback || path == given
		}
		if fallback {
			located, found := paths.Lookup(target)
			if found {
				path = located
			}
			if !found && len(selected) == 1 {
				target = selected[0]
				if located, ok := paths.Lookup(target); ok {
					path = located
				}
			}
			isSelected := false
			for _, node := range selected {
				isSelected = isSelected || node == target
			}
			if action.Field != "" && isSelected {
				if target != nil && target.Kind == yaml.DocumentNode && len(target.Content) > 0 {
					target = target.Content[0]
				}
				if target != nil {
					field := vacuumUtils.FindFieldPath(action.Field, target.Content, vacuumUtils.FieldPathOptions{ResolveSingleItemCombinators: rule.Resolved})
					target = field.ValueNode
					if located, ok := paths.Lookup(target); ok {
						path = located
					} else {
						segments, _ := vacuumUtils.ParseFieldPath(action.Field)
						for _, segment := range segments {
							if segment.Type == vacuumUtils.SegmentArrayIndex {
								path = vacuumUtils.AppendResultPathIndex(path, segment.Index)
							} else {
								path = vacuumUtils.AppendResultPathSegment(path, segment.Key)
							}
						}
					}
				}
			}
		}
		property, pointer, valueNode := ruleMessageLocation(root, path)
		if valueNode != nil {
			target = valueNode
		}
		value := ""
		if target != nil {
			if target.Kind == yaml.ScalarNode {
				value = target.Value
			} else {
				var decoded any
				if target.Decode(&decoded) == nil {
					if encoded, err := json.Marshal(decoded); err == nil {
						value = string(encoded)
					}
				}
			}
		}
		result.Message = strings.NewReplacer(
			"{{property}}", property, "{{path}}", pointer, "{{value}}", value,
			"{{description}}", rule.Description, "{{error}}", result.Message,
		).Replace(rule.Message)
	}
}

// Rule message paths use escaped JSON Pointers, as in Spectral templates.
func ruleMessageLocation(root *yaml.Node, path string) (property, pointer string, node *yaml.Node) {
	node = root
	if node != nil && node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}
	steps, ok := parseResultPathSteps(path)
	if !ok {
		return "", path, nil
	}
	for _, step := range steps {
		switch step.kind {
		case resultPathStepName:
			property = step.name
			var child *yaml.Node
			if node != nil && node.Kind == yaml.MappingNode {
				for i := 0; i+1 < len(node.Content); i += 2 {
					if node.Content[i].Value == step.name {
						child = node.Content[i+1]
						break
					}
				}
			}
			node = child
		case resultPathStepIndex:
			property = strconv.Itoa(step.index)
			if node != nil && node.Kind == yaml.SequenceNode && step.index >= 0 && step.index < len(node.Content) {
				node = node.Content[step.index]
			} else {
				node = nil
			}
		default:
			return "", path, nil
		}
		pointer += "/" + strings.ReplaceAll(strings.ReplaceAll(property, "~", "~0"), "/", "~1")
	}
	return
}
