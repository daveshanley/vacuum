package jsruleset

import (
	"fmt"
	"strconv"

	"encoding/json"
	"github.com/daveshanley/vacuum/model"
	"github.com/dop251/goja"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/jsonpath/pkg/jsonpath"
)

type adapter struct {
	name  string
	scope *Scope
}

func (a *adapter) GetCategory() string                 { return model.FunctionCategoryCustomJS }
func (a *adapter) GetSchema() model.RuleFunctionSchema { return model.RuleFunctionSchema{Name: a.name} }
func (a *adapter) RunRule(nodes []*yaml.Node, rfc model.RuleFunctionContext) []model.RuleFunctionResult {
	fail := func(err error) []model.RuleFunctionResult {
		if rfc.ReportError != nil {
			rfc.ReportError(err)
		}
		return nil
	}
	if a.scope == nil {
		return fail(fmt.Errorf("executable ruleset requires a document scope"))
	}
	ctx, cancel := invocationContext(rfc)
	defer cancel()
	var results []model.RuleFunctionResult
	err := a.scope.run(ctx, func() error {
		s := a.scope
		s.current = &rfc
		defer func() { s.current = nil }()
		pathIndex := s.paths[rfc.ExecutionRoot]
		if pathIndex == nil {
			pathIndex = nodePaths(rfc.ExecutionRoot)
			s.paths[rfc.ExecutionRoot] = pathIndex
		}
		for _, node := range nodes {
			segments, ok := pathIndex[node]
			if !ok {
				return fmt.Errorf("cannot locate JavaScript rule input in execution document")
			}
			segments = append([]any(nil), segments...)
			selected := node
			if rfc.RuleAction.Field != "" {
				field, err := pathSegments("$." + rfc.RuleAction.Field)
				if err != nil {
					return err
				}
				segments = append(segments, field...)
				selected = exactNode(node, field)
			}

			input := goja.Undefined()
			if selected != nil {
				var err error
				input, err = s.nodeValue(selected, map[*yaml.Node]bool{})
				if err != nil {
					return err
				}
			}

			rule := map[string]any{"name": rfc.Rule.Id, "description": rfc.Rule.Description, "severity": rfc.Rule.GetSeverityAsIntValue(), "resolved": rfc.Rule.Resolved, "recommended": rfc.Rule.Recommended, "given": rfc.Rule.Given, "formats": rfc.Rule.Formats}
			document, err := s.strictObject(map[string]any{"data": s.document, "source": s.source}, "context.document")
			if err != nil {
				return err
			}
			metadata, err := s.strictObject(rule, "context.rule")
			if err != nil {
				return err
			}
			contextValue, err := s.strictObject(map[string]any{"path": segments, "document": document, "rule": metadata}, "context")
			if err != nil {
				return err
			}
			options := goja.Null()
			if rfc.Options != nil {
				encoded, err := json.Marshal(rfc.Options)
				if err != nil {
					return err
				}
				var copy any
				if err = json.Unmarshal(encoded, &copy); err != nil {
					return err
				}
				options = s.vm.ToValue(copy)
			}
			output, err := s.loop.Run(ctx, func(vm *goja.Runtime) (goja.Value, error) {
				return s.functions[a.name](goja.Undefined(), input, options, contextValue)
			})
			if err != nil {
				return err
			}
			result := s.loop.AwaitPromise(ctx, output)
			if result.Error != nil {
				return result.Error
			}
			output = result.Value
			if goja.IsUndefined(output) {
				continue
			}
			array, ok := output.(*goja.Object)
			if !ok || array.ClassName() != "Array" {
				return fmt.Errorf("JavaScript rule must return undefined or an array of findings")
			}
			for i := 0; i < int(array.Get("length").ToInteger()); i++ {
				item, ok := array.Get(strconv.Itoa(i)).(*goja.Object)
				if !ok {
					return fmt.Errorf("finding %d must be an object", i)
				}
				messageValue := item.Get("message")
				if messageValue == nil {
					return fmt.Errorf("finding %d requires a string message", i)
				}
				message, ok := messageValue.Export().(string)
				if !ok {
					return fmt.Errorf("finding %d requires a string message", i)
				}
				resultPath := segments
				if pv := item.Get("path"); pv != nil && !goja.IsUndefined(pv) {
					p, ok := pv.(*goja.Object)
					if !ok || p.ClassName() != "Array" {
						return fmt.Errorf("finding path must be an array")
					}
					resultPath = make([]any, int(p.Get("length").ToInteger()))
					for j := range resultPath {
						v := p.Get(strconv.Itoa(j))
						switch x := v.Export().(type) {
						case string:
							resultPath[j] = x
						case int64:
							if x < 0 {
								return fmt.Errorf("path index must be nonnegative")
							}
							resultPath[j] = int(x)
						default:
							return fmt.Errorf("path segments must be strings or integer indices")
						}
					}
				}
				located := locateNode(rfc.ExecutionRoot, resultPath)
				if located == nil {
					located = node
				}
				results = append(results, model.RuleFunctionResult{Message: message, Path: resultPathString(resultPath), StartNode: located, EndNode: located, Rule: rfc.Rule})
			}
		}
		return nil
	})
	if err != nil {
		return fail(err)
	}
	return results
}
func (s *Scope) strictObject(value map[string]any, label string) (goja.Value, error) {
	factory, err := s.vm.RunString(`(function(value,label){return new Proxy(Object.freeze(Object.assign({},value)),{get(target,key){if(typeof key==='symbol') return target[key];if(!Object.prototype.hasOwnProperty.call(target,key))throw Error('Unsupported '+label+' API: '+key);return target[key];}})})`)
	if err != nil {
		return nil, err
	}
	fn, _ := goja.AssertFunction(factory)
	return fn(goja.Undefined(), s.vm.ToValue(value), s.vm.ToValue(label))
}
func pathSegments(path string) ([]any, error) {
	p, err := jsonpath.NewPath(path)
	if err != nil {
		return nil, err
	}
	parts, err := p.GetSegmentInfo()
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(parts))
	for _, part := range parts {
		if part.Kind == jsonpath.SegmentKindArrayIndex {
			out = append(out, int(part.Index))
		} else {
			out = append(out, part.Key)
		}
	}
	return out, nil
}
func resultPathString(parts []any) string {
	path := "$"
	for _, part := range parts {
		switch p := part.(type) {
		case string:
			path = model.AppendResultPathSegment(path, p)
		case int:
			path = model.AppendResultPathIndex(path, p)
		}
	}
	return path
}
func locateNode(node *yaml.Node, parts []any) *yaml.Node {
	if node != nil && node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}
	for _, part := range parts {
		if node == nil {
			return nil
		}
		if node.Kind == yaml.AliasNode {
			node = node.Alias
		}
		switch p := part.(type) {
		case string:
			if node.Kind != yaml.MappingNode {
				return nil
			}
			var next *yaml.Node
			for i := 0; i+1 < len(node.Content); i += 2 {
				if node.Content[i].Value == p {
					next = node.Content[i+1]
					break
				}
			}
			if next == nil {
				return node
			}
			node = next
		case int:
			if node.Kind != yaml.SequenceNode || p >= len(node.Content) {
				return node
			}
			node = node.Content[p]
		}
	}
	return node
}

func nodePaths(root *yaml.Node) map[*yaml.Node][]any {
	out := map[*yaml.Node][]any{}
	var walk func(*yaml.Node, []any)
	walk = func(n *yaml.Node, path []any) {
		if n == nil {
			return
		}
		if _, ok := out[n]; ok {
			return
		}
		out[n] = append([]any{}, path...)
		switch n.Kind {
		case yaml.DocumentNode:
			if len(n.Content) > 0 {
				walk(n.Content[0], path)
			}
		case yaml.AliasNode:
			walk(n.Alias, path)
		case yaml.MappingNode:
			for i := 0; i+1 < len(n.Content); i += 2 {
				next := append(append([]any{}, path...), n.Content[i].Value)
				walk(n.Content[i+1], next)
			}
		case yaml.SequenceNode:
			for i, child := range n.Content {
				walk(child, append(append([]any{}, path...), i))
			}
		}
	}
	walk(root, nil)
	return out
}
func exactNode(n *yaml.Node, parts []any) *yaml.Node {
	if n != nil && n.Kind == yaml.DocumentNode && len(n.Content) > 0 {
		n = n.Content[0]
	}
	for _, part := range parts {
		if n == nil {
			return nil
		}
		if n.Kind == yaml.AliasNode {
			n = n.Alias
		}
		if n == nil {
			return nil
		}
		switch v := part.(type) {
		case string:
			if n.Kind != yaml.MappingNode {
				return nil
			}
			var found *yaml.Node
			for i := 0; i+1 < len(n.Content); i += 2 {
				if n.Content[i].Value == v {
					found = n.Content[i+1]
					break
				}
			}
			n = found
		case int:
			if n.Kind != yaml.SequenceNode || v < 0 || v >= len(n.Content) {
				return nil
			}
			n = n.Content[v]
		}
	}
	return n
}
