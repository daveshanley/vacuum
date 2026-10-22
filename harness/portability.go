package harness

import (
	"path/filepath"
	"strings"

	"github.com/pb33f/go-yaml"
)

// Absolute authored paths remain bound to the original checkout after copying.
// Keep execution available, but do not claim those inputs form a replay bundle.
func (r *runner) trackAbsolutePath(path string) {
	if filepath.IsAbs(strings.SplitN(path, "#", 2)[0]) {
		r.limitations["absolute dependency paths cannot be relocated in a bundle"] = true
	}
}

func (r *runner) trackDocumentPaths(root *yaml.Node) {
	// Walk syntax, not aliases: aliases share their target's authored references.
	// A stack avoids call-stack growth for deeply nested documents.
	pending := []*yaml.Node{root}
	for len(pending) > 0 {
		node := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if node == nil {
			continue
		}
		if node.Kind == yaml.MappingNode {
			for i := 0; i+1 < len(node.Content); i += 2 {
				key, value := node.Content[i], node.Content[i+1]
				switch key.Value {
				case "$ref", "$dynamicRef", "$recursiveRef":
					r.trackAbsolutePath(value.Value)
				case "sourceDescriptions":
					for _, source := range value.Content {
						if source.Kind != yaml.MappingNode {
							continue
						}
						for j := 0; j+1 < len(source.Content); j += 2 {
							if source.Content[j].Value == "url" {
								r.trackAbsolutePath(source.Content[j+1].Value)
							}
						}
					}
				}
			}
		}
		pending = append(pending, node.Content...)
	}
}
