package repair

import (
	"bytes"
	"fmt"
	"io"

	"github.com/pb33f/go-yaml"
)

// ParseDocument accepts exactly one YAML or JSON document. A stream cannot be
// safely rewritten through a single-root lint model.
func ParseDocument(data []byte) (*yaml.Node, error) {
	if len(data) > 64<<20 {
		return nil, fmt.Errorf("repair source exceeds 64 MiB")
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var doc yaml.Node
	if err := decoder.Decode(&doc); err != nil {
		return nil, err
	}
	if err := decoder.Decode(new(yaml.Node)); err != io.EOF {
		return nil, fmt.Errorf("repair requires exactly one document")
	}
	if len(doc.Content) != 1 {
		return nil, fmt.Errorf("repair requires exactly one document")
	}
	if _, err := treeCost(&doc, 0); err != nil {
		return nil, err
	}
	return &doc, nil
}

// treeCost bounds copying and recursion before mutations allocate another tree.
func treeCost(n *yaml.Node, depth int) (int, error) {
	if depth > 256 {
		return 0, fmt.Errorf("repair tree exceeds 256 levels")
	}
	cost := 256 + len(n.Value) + len(n.HeadComment) + len(n.LineComment) + len(n.FootComment)
	for _, c := range n.Content {
		child, err := treeCost(c, depth+1)
		if err != nil {
			return 0, err
		}
		cost += child
		if cost > 256<<20 {
			return 0, fmt.Errorf("repair tree exceeds memory budget")
		}
	}
	return cost, nil
}
