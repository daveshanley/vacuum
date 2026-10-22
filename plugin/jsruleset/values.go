package jsruleset

import (
	"fmt"
	"github.com/dop251/goja"
	"github.com/pb33f/go-yaml"
	"strconv"
)

// Preserve property order across the bridge, including alphabetical checks.
func (s *Scope) nodeValue(node *yaml.Node, active map[*yaml.Node]bool) (goja.Value, error) {
	if node == nil {
		return goja.Undefined(), nil
	}
	if active[node] {
		return nil, fmt.Errorf("cyclic YAML value cannot be passed to JavaScript")
	}
	active[node] = true
	defer delete(active, node)
	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) > 0 {
			return s.nodeValue(node.Content[0], active)
		}
	case yaml.AliasNode:
		return s.nodeValue(node.Alias, active)
	case yaml.MappingNode:
		object := s.vm.NewObject()
		for i := 0; i+1 < len(node.Content); i += 2 {
			v, err := s.nodeValue(node.Content[i+1], active)
			if err != nil {
				return nil, err
			}
			if err = object.Set(node.Content[i].Value, v); err != nil {
				return nil, err
			}
		}
		return object, nil
	case yaml.SequenceNode:
		values := make([]any, len(node.Content))
		for i, child := range node.Content {
			v, err := s.nodeValue(child, active)
			if err != nil {
				return nil, err
			}
			values[i] = v
		}
		return s.vm.NewArray(values...), nil
	}
	var value any
	if err := node.Decode(&value); err != nil {
		return nil, err
	}
	return s.vm.ToValue(value), nil
}
func (s *Scope) valueNode(value goja.Value, active map[*goja.Object]bool) (*yaml.Node, error) {
	if object, ok := value.(*goja.Object); ok {
		if active[object] {
			return nil, fmt.Errorf("cyclic JavaScript input")
		}
		active[object] = true
		defer delete(active, object)
		node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		if object.ClassName() == "Array" {
			node.Kind = yaml.SequenceNode
			node.Tag = "!!seq"
			for i := 0; i < int(object.Get("length").ToInteger()); i++ {
				child, err := s.valueNode(object.Get(strconv.Itoa(i)), active)
				if err != nil {
					return nil, err
				}
				node.Content = append(node.Content, child)
			}
		} else {
			for _, key := range object.Keys() {
				child, err := s.valueNode(object.Get(key), active)
				if err != nil {
					return nil, err
				}
				node.Content = append(node.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, child)
			}
		}
		return node, nil
	}
	node := &yaml.Node{}
	err := node.Encode(value.Export())
	return node, err
}
