package motor

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	doctorModel "github.com/pb33f/doctor/model"
	drV3 "github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/jsonpath/pkg/jsonpath"
)

// formatRuleMessages runs once for each function result, before auto-fixes can
// change its value. Replacement text is never interpreted as another template.
func formatRuleMessages(rule *model.Rule, action model.RuleAction, selected []*yaml.Node, results []model.RuleFunctionResult, root *yaml.Node, paths *vacuumUtils.NodePathIndex, updates *sync.Map) {
	var selectedNodes map[*yaml.Node]struct{}
	if len(selected) > 1 && ruleMessageNeedsLocation(rule.Message) {
		selectedNodes = make(map[*yaml.Node]struct{}, len(selected))
		for _, node := range selected {
			selectedNodes[node] = struct{}{}
		}
	}
	for i := range results {
		result := &results[i]
		result.Rule = rule
		if !strings.Contains(rule.Message, "{{") {
			result.Message = rule.Message
			continue
		}
		original := doctorMessageKey{rule.Id, result.StartNode, result.Path, result.Message}
		var property, pointer, value string
		if ruleMessageNeedsLocation(rule.Message) {
			path := result.Path
			target := result.StartNode
			_, targetIsValue := selectedNodes[target]
			targetIsValue = targetIsValue || len(selected) == 1 && selected[0] == target
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
					targetIsValue = true
					if located, ok := paths.Lookup(target); ok {
						path = located
					}
				}
				_, isSelected := selectedNodes[target]
				isSelected = isSelected || len(selected) == 1 && selected[0] == target
				if action.Field != "" && isSelected {
					if target != nil && target.Kind == yaml.DocumentNode && len(target.Content) > 0 {
						target = target.Content[0]
					}
					if target != nil {
						field := vacuumUtils.FindFieldPath(action.Field, target.Content, vacuumUtils.FieldPathOptions{ResolveSingleItemCombinators: rule.Resolved})
						target = field.ValueNode
						targetIsValue = field.Found
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
			var valueNode *yaml.Node
			valueRoot := root
			knownPath, known := paths.Lookup(target)
			if !strings.Contains(rule.Message, "{{value}}") || targetIsValue && known && knownPath == path {
				valueRoot = nil
			}
			property, pointer, valueNode = ruleMessageLocation(valueRoot, path)
			if valueNode != nil {
				target = valueNode
			}
			if target != nil && strings.Contains(rule.Message, "{{value}}") {
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
		}
		result.Message = strings.NewReplacer(
			"{{property}}", property, "{{path}}", pointer, "{{value}}", value,
			"{{description}}", rule.Description, "{{error}}", result.Message,
		).Replace(rule.Message)
		if updates != nil {
			updates.Store(original, doctorMessageUpdate{result.Message, rule.Message})
		}
	}
}

// Rule message paths use escaped JSON Pointers, as in Spectral templates.
func ruleMessageLocation(root *yaml.Node, path string) (property, pointer string, node *yaml.Node) {
	node = root
	if node != nil && node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}
	parsed, err := jsonpath.NewPath(path)
	if err != nil || !parsed.IsSingular() {
		return "", path, nil
	}
	steps, err := parsed.GetSegmentInfo()
	if err != nil {
		return "", path, nil
	}
	for _, step := range steps {
		switch step.Kind {
		case jsonpath.SegmentKindMemberName:
			property = step.Key
			var child *yaml.Node
			if node != nil && node.Kind == yaml.MappingNode {
				for i := 0; i+1 < len(node.Content); i += 2 {
					if node.Content[i].Value == step.Key {
						child = node.Content[i+1]
						break
					}
				}
			}
			node = child
		case jsonpath.SegmentKindArrayIndex:
			property = strconv.FormatInt(step.Index, 10)
			if node != nil && node.Kind == yaml.SequenceNode && int(step.Index) >= 0 && int(step.Index) < len(node.Content) {
				node = node.Content[int(step.Index)]
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

// Doctor keeps copies of findings on models and at their root. Update those
// copies once after rule execution, using the original result identity.
type doctorMessageKey struct {
	rule          string
	node          *yaml.Node
	path, message string
}
type doctorMessageUpdate struct{ message, template string }

func updateDoctorMessages(doc *doctorModel.DrDocument, updates *sync.Map) {
	if doc == nil || updates == nil {
		return
	}
	seen := make(map[*drV3.Foundation]bool)
	update := func(f *drV3.Foundation) {
		if f == nil || seen[f] {
			return
		}
		seen[f] = true
		f.Mutex.Lock()
		defer f.Mutex.Unlock()
		for _, result := range f.RuleResults {
			if result == nil {
				continue
			}
			if value, ok := updates.Load(doctorMessageKey{result.RuleId, result.StartNode, result.Path, result.Message}); ok {
				message := value.(doctorMessageUpdate)
				result.Message = message.message
				if result.Rule != nil {
					result.Rule.Message = message.template
				}
			}
		}
	}
	if doc.V3Document != nil {
		update(&doc.V3Document.Foundation)
		return
	}
	for _, schema := range doc.Schemas {
		if schema == nil {
			continue
		}
		update(&schema.Foundation)
		if root, ok := schema.GetRoot().(*drV3.Foundation); ok {
			update(root)
		}
	}
}

func ruleMessageNeedsLocation(message string) bool {
	return strings.Contains(message, "{{property}}") || strings.Contains(message, "{{path}}") || strings.Contains(message, "{{value}}")
}

type messagePathIndexKey struct{ root *yaml.Node }

func messagePathIndex(root *yaml.Node, cache *sync.Map) *vacuumUtils.NodePathIndex {
	if cache == nil {
		return vacuumUtils.BuildNodePathIndex(root)
	}
	key := messagePathIndexKey{root}
	if cached, ok := cache.Load(key); ok {
		return cached.(func() *vacuumUtils.NodePathIndex)()
	}
	build := sync.OnceValue(func() *vacuumUtils.NodePathIndex { return vacuumUtils.BuildNodePathIndex(root) })
	cached, _ := cache.LoadOrStore(key, build)
	return cached.(func() *vacuumUtils.NodePathIndex)()
}
