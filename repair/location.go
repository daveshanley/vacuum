package repair

import (
	"github.com/pb33f/go-yaml"
	"strconv"
)

// Locations indexes authored source nodes by identity and exact parser position.
// Position fallback is used only for matching nodes from a separate parse of
// the same source; callers must check document ownership first.
type Locations struct {
	nodes     map[*yaml.Node]string
	positions map[nodePosition][]locatedNode
}
type nodePosition struct {
	line, column int
	kind         yaml.Kind
	tag, value   string
}
type locatedNode struct {
	pointer string
	node    *yaml.Node
}

// IndexLocations builds a pointer index for one canonical document tree.
func IndexLocations(root *yaml.Node) *Locations {
	l := &Locations{nodes: make(map[*yaml.Node]string), positions: make(map[nodePosition][]locatedNode)}
	var walk func(*yaml.Node, string)
	walk = func(n *yaml.Node, path string) {
		if n == nil {
			return
		}
		if _, seen := l.nodes[n]; seen {
			return
		}
		l.nodes[n] = path
		if n.Line > 0 && n.Column > 0 {
			key := nodePosition{n.Line, n.Column, n.Kind, n.Tag, n.Value}
			l.positions[key] = append(l.positions[key], locatedNode{path, n})
		}
		switch n.Kind {
		case yaml.DocumentNode:
			for _, child := range n.Content {
				walk(child, path)
			}
		case yaml.MappingNode:
			for i := 0; i+1 < len(n.Content); i += 2 {
				p := path + "/" + EscapeToken(n.Content[i].Value)
				walk(n.Content[i], p)
				walk(n.Content[i+1], p)
			}
		case yaml.SequenceNode:
			for i, child := range n.Content {
				walk(child, path+"/"+strconv.Itoa(i))
			}
		}
	}
	walk(root, "")
	return l
}

// FindExact returns the pointer only when n belongs to the indexed tree.
// It does not match nodes from other documents by parser position.
func (l *Locations) FindExact(n *yaml.Node) (string, bool) {
	p, ok := l.nodes[n]
	return p, ok
}

// Find returns an unambiguous source pointer, including mapping-key locations.
func (l *Locations) Find(n *yaml.Node) (string, bool) {
	if n == nil {
		return "", false
	}
	if p, ok := l.nodes[n]; ok {
		return p, true
	}
	entries := l.positions[nodePosition{n.Line, n.Column, n.Kind, n.Tag, n.Value}]
	if len(entries) == 1 {
		return entries[0].pointer, true
	}
	return "", false
}
