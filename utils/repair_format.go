package utils

import (
	"bytes"

	"github.com/pb33f/go-yaml"
)

// RepairYAMLFormat captures indentation before edits move or replace source nodes.
// CaptureRepairYAMLFormat derives it from the original parsed document.
type RepairYAMLFormat struct {
	indent  int
	compact bool
}

// CaptureRepairYAMLFormat records the authored block and sequence indentation
// without parsing or retaining another copy of the source tree.
func CaptureRepairYAMLFormat(root *yaml.Node) RepairYAMLFormat {
	indent, compact := repairYAMLIndent(root)
	return RepairYAMLFormat{indent: indent, compact: compact}
}

// RenderRepairYAML retains the source's block and sequence indentation, line
// endings and final newline without introducing automatic scalar wrapping.
func RenderRepairYAML(root *yaml.Node, original []byte, format RepairYAMLFormat) ([]byte, error) {
	var out bytes.Buffer
	// The legacy encoder wraps at 80 columns. Repairs must not reflow unrelated
	// long values, so use the same defaults with wrapping explicitly disabled.
	encoder, err := yaml.NewDumper(&out, yaml.WithV3Defaults(),
		yaml.WithIndent(max(2, format.indent)), yaml.WithCompactSeqIndent(format.compact), yaml.WithLineWidth(-1))
	if err != nil {
		return nil, err
	}
	if err := encoder.Dump(root); err != nil {
		_ = encoder.Close()
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	if original == nil {
		return out.Bytes(), nil
	}
	return repairLineEndings(out.Bytes(), original), nil
}

// Existing nodes retain their authored columns through repair. Ignore new nodes
// (column zero), flow collections and inline values when measuring indentation.
// This avoids parsing and retaining another copy of a large source document.
func repairYAMLIndent(root *yaml.Node) (int, bool) {
	mappings, sequences := make(map[int]int), make(map[int]int)
	seen := make(map[*yaml.Node]bool)
	var walk func(*yaml.Node)
	walk = func(n *yaml.Node) {
		if n == nil || seen[n] {
			return
		}
		seen[n] = true
		if n.Kind == yaml.MappingNode {
			for i := 0; i+1 < len(n.Content); i += 2 {
				key, value := n.Content[i], n.Content[i+1]
				if key.Column > 0 && value.Column > 0 && value.Line > key.Line && value.Style&yaml.FlowStyle == 0 {
					delta := value.Column - key.Column
					if value.Kind == yaml.MappingNode && delta > 0 && delta <= 9 {
						mappings[delta]++
					}
					if value.Kind == yaml.SequenceNode && delta >= 0 && delta <= 9 {
						sequences[delta]++
					}
				}
			}
		}
		for _, child := range n.Content {
			walk(child)
		}
	}
	walk(root)
	mostCommon := func(counts map[int]int, fallback int) int {
		best, count := fallback, 0
		for value, n := range counts {
			if n > count || n == count && value < best {
				best, count = value, n
			}
		}
		return best
	}
	indent := mostCommon(mappings, 0)
	if indent == 0 {
		indent = mostCommon(sequences, 2)
		if indent == 0 {
			indent = 2
		}
	}
	seq := mostCommon(sequences, indent)
	return indent, seq == indent-2
}

func repairJSONIndent(original []byte) string {
	for len(original) > 0 {
		line, rest, found := bytes.Cut(original, []byte{'\n'})
		n := 0
		for n < len(line) && (line[n] == ' ' || line[n] == '\t') {
			n++
		}
		if n > 0 && n < len(line) && (line[n] == '"' || line[n] == '{' || line[n] == '[') {
			return string(line[:n])
		}
		if !found {
			break
		}
		original = rest
	}
	return ""
}

// repairLineEndings retains CRLF and the source's trailing line-break count.
func repairLineEndings(rendered, original []byte) []byte {
	rendered = bytes.TrimRight(rendered, "\r\n")
	end := len(original)
	for end > 0 && (original[end-1] == '\r' || original[end-1] == '\n') {
		end--
	}
	if bytes.Contains(original, []byte("\r\n")) {
		rendered = bytes.ReplaceAll(rendered, []byte{'\n'}, []byte("\r\n"))
	}
	return append(rendered, original[end:]...)
}
