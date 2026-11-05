package repair

import (
	"github.com/pb33f/go-yaml"
	"strconv"
	"strings"
)

// editIndex checks ancestor/descendant overlap in pointer depth, rather than
// comparing every new operation against the entire batch.
type editIndex struct {
	terminal bool
	children map[string]*editIndex
}

func (x *editIndex) conflicts(path string) bool {
	if x.terminal {
		return true
	}
	if path == "" {
		return len(x.children) > 0
	}
	for _, token := range strings.Split(path[1:], "/") {
		x = x.children[token]
		if x == nil {
			return false
		}
		if x.terminal {
			return true
		}
	}
	return len(x.children) > 0
}
func (x *editIndex) add(path string) {
	if path != "" {
		for _, token := range strings.Split(path[1:], "/") {
			if x.children == nil {
				x.children = make(map[string]*editIndex)
			}
			if x.children[token] == nil {
				x.children[token] = &editIndex{}
			}
			x = x.children[token]
		}
	}
	x.terminal = true
}

func sourceNodes(root *yaml.Node) map[string]*yaml.Node {
	nodes := make(map[string]*yaml.Node)
	var walk func(*yaml.Node, string)
	walk = func(n *yaml.Node, path string) {
		nodes[path] = n
		switch n.Kind {
		case yaml.MappingNode:
			for i := 0; i < len(n.Content); i += 2 {
				walk(n.Content[i+1], path+"/"+EscapeToken(n.Content[i].Value))
			}
		case yaml.SequenceNode:
			for i, c := range n.Content {
				walk(c, path+"/"+strconv.Itoa(i))
			}
		}
	}
	walk(root, "")
	return nodes
}

// Positional edits own their immediate sequence; mapping siblings stay independent.
func editFootprint(path string, nodes map[string]*yaml.Node) string {
	if path != "" {
		parent := path[:strings.LastIndex(path, "/")]
		if n := nodes[parent]; n != nil && n.Kind == yaml.SequenceNode {
			return parent
		}
	}
	return path
}
func operationFootprint(op Operation, nodes map[string]*yaml.Node) string {
	if op.Op == "replace" || op.Op == "test" {
		return op.Path
	}
	return editFootprint(op.Path, nodes)
}

func sourceFootprint(op Operation, nodes map[string]*yaml.Node) string {
	if op.Op == "move" {
		return editFootprint(op.From, nodes)
	}
	return op.From
}
