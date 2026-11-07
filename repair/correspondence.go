// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package repair

import (
	"fmt"
	"github.com/pb33f/go-yaml"
	"strconv"
)

// Correspondence maps surviving candidate locations to their authored source.
// Replayed moves retain node identity. Existing mapping keys retain field identity;
// newly added fields and sequence elements have no origin.
type Correspondence struct {
	pointers map[string]string
	lines    map[int]int
}

// OriginalPointer returns the source pointer for a surviving candidate node.
func (c *Correspondence) OriginalPointer(pointer string) (string, bool) {
	p, ok := c.pointers[pointer]
	return p, ok
}

// OriginalLine returns an original line only when all matched nodes agree.
func (c *Correspondence) OriginalLine(line int) (int, bool) {
	n, ok := c.lines[line]
	return n, ok && n > 0
}

// TraceChanges replays validated operations to track source identity through
// sequence removals and moves. Candidate parsing supplies the emitted line numbers.
func TraceChanges(source, candidate []byte, plan Plan) (*Correspondence, error) {
	original, err := ParseDocument(source)
	if err != nil {
		return nil, err
	}
	origins := IndexLocations(original).nodes
	root := original.Content[0]
	for _, edit := range plan.KeyRemovals {
		parent, err := at(root, edit.Parent)
		if err != nil {
			return nil, err
		}
		if parent.Kind != yaml.MappingNode || edit.Index < 0 || edit.Index >= len(parent.Content)/2 {
			return nil, fmt.Errorf("invalid duplicate key removal")
		}
		parent.Content = append(parent.Content[:2*edit.Index], parent.Content[2*edit.Index+2:]...)
	}
	for _, op := range plan.Operations {
		root, err = applyOperation(root, op)
		if err != nil {
			return nil, err
		}
	}
	parsed, err := ParseDocument(candidate)
	if err != nil {
		return nil, err
	}
	c := &Correspondence{pointers: map[string]string{}, lines: map[int]int{}}
	var walk func(*yaml.Node, *yaml.Node, string)
	walk = func(old, current *yaml.Node, path string) {
		if old == nil || current == nil || old.Kind != current.Kind {
			return
		}
		if origin, ok := origins[old]; ok {
			c.pointers[path] = origin
			if old.Line > 0 && current.Line > 0 {
				previous, exists := c.lines[current.Line]
				if !exists {
					c.lines[current.Line] = old.Line
				} else if previous != old.Line {
					c.lines[current.Line] = 0
				}
			}
		}
		switch old.Kind {
		case yaml.MappingNode:
			fields := make(map[string]int, len(old.Content)/2)
			for i := 0; i+1 < len(old.Content); i += 2 {
				fields[old.Content[i].Value] = i
			}
			for i := 0; i+1 < len(current.Content); i += 2 {
				if j, ok := fields[current.Content[i].Value]; ok {
					p := path + "/" + EscapeToken(current.Content[i].Value)
					walk(old.Content[j], current.Content[i], p)
					walk(old.Content[j+1], current.Content[i+1], p)
				}
			}
		case yaml.SequenceNode:
			// Unknown plugin structural edits cannot claim positional correspondence.
			if len(old.Content) != len(current.Content) {
				return
			}
			for i, n := range old.Content {
				walk(n, current.Content[i], path+"/"+strconv.Itoa(i))
			}
		}
	}
	walk(root, parsed.Content[0], "")
	return c, nil
}
