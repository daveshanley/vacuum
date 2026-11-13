// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package repair

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/go-yaml"
)

type sourceSpan struct{ start, end int }
type sourceColumnCorrection struct{ column, extra int }

type sourceRenderer struct {
	data    []byte
	json    bool
	lines   []int
	spans   map[*yaml.Node]sourceSpan
	format  utils.RepairYAMLFormat
	columns map[int][]sourceColumnCorrection
	entries map[*yaml.Node]map[string]int
}

// renderSource reuses source tokens for unchanged nodes and renders only edited
// values. The resulting tree is checked against the applied plan before return.
func renderSource(before, document *yaml.Node, data []byte, jsonDocument bool, format utils.RepairYAMLFormat) ([]byte, error) {
	after := document.Content[0]
	r := &sourceRenderer{data: data, json: jsonDocument, spans: make(map[*yaml.Node]sourceSpan), format: format, lines: []int{0}}
	r.columns = make(map[int][]sourceColumnCorrection)
	r.entries = make(map[*yaml.Node]map[string]int)
	line, column, extra := 1, 1, 0
	for i := 0; i < len(data); {
		c, size := utf8.DecodeRune(data[i:])
		if size > 1 {
			extra += size - 1
			r.columns[line] = append(r.columns[line], sourceColumnCorrection{column + 1, extra})
		}
		i += size
		column++
		if c == '\n' {
			r.lines = append(r.lines, i)
			line++
			column, extra = 1, 0
		}
	}
	r.index(before, len(data), r.json)
	span := r.spans[before]
	body, err := r.render(before, after)
	if err != nil {
		return nil, err
	}
	out := append(append(append([]byte(nil), data[:span.start]...), body...), data[span.end:]...)
	if span.end == len(data) && bytes.HasSuffix(data, []byte("\n")) && !bytes.HasSuffix(out, []byte("\n")) {
		if bytes.HasSuffix(data, []byte("\r\n")) {
			out = append(out, '\r')
		}
		out = append(out, '\n')
	}
	if jsonDocument && !json.Valid(out) {
		return nil, fmt.Errorf("source-preserving repair produced invalid JSON")
	}
	parsed, err := ParseDocument(out)
	if err != nil {
		return nil, fmt.Errorf("cannot preserve source syntax: %w", err)
	}
	if !sameNodeValues(parsed, document) {
		return nil, fmt.Errorf("source-preserving repair differs from the requested edits")
	}
	return out, nil
}
func (r *sourceRenderer) offset(n *yaml.Node) int {
	if n.Line < 1 || n.Line > len(r.lines) {
		return 0
	}
	p := r.lines[n.Line-1] + max(0, n.Column-1)
	corrections := r.columns[n.Line]
	i := sort.Search(len(corrections), func(i int) bool { return corrections[i].column > n.Column })
	if i > 0 {
		p += corrections[i-1].extra
	}
	return min(p, len(r.data))
}
func (r *sourceRenderer) index(n *yaml.Node, bound int, inFlow bool) {
	start := r.offset(n)
	if start > bound {
		start = bound
	}
	end := bound
	if n.Kind == yaml.MappingNode || n.Kind == yaml.SequenceNode {
		for i, c := range n.Content {
			limit := bound
			if i+1 < len(n.Content) {
				limit = r.offset(n.Content[i+1])
				if n.Content[i+1].Line > c.Line {
					limit = r.lines[n.Content[i+1].Line-1]
				}
			}
			// Mapping keys end before their value. Values end before the next key.
			r.index(c, limit, r.json || n.Style&yaml.FlowStyle != 0)
		}
		if n.Style&yaml.FlowStyle != 0 {
			end = r.flowEnd(start, bound)
		} else if len(n.Content) > 0 {
			step := 1
			if n.Kind == yaml.MappingNode {
				step = 2
			}
			start, _ = r.blockEntryStart(n, 0)
			end = r.blockEntryEnd(n, len(n.Content)-step, step)
		}
	} else {
		end = r.scalarEnd(n, start, bound, inFlow)
	}
	r.spans[n] = sourceSpan{start, end}
}
func (r *sourceRenderer) flowEnd(start, bound int) int {
	depth := 0
	for i := start; i < bound; i++ {
		switch r.data[i] {
		case '\'', '"':
			i = r.quoteEnd(i, bound) - 1
		case '{', '[':
			depth++
		case '}', ']':
			depth--
			if depth == 0 {
				return i + 1
			}
		case '#':
			if !r.json {
				for i < bound && r.data[i] != '\n' {
					i++
				}
			}
		}
	}
	return bound
}
func (r *sourceRenderer) quoteEnd(start, bound int) int {
	q := r.data[start]
	for i := start + 1; i < bound; i++ {
		if q == '"' && r.data[i] == '\\' {
			i++
			continue
		}
		if r.data[i] == q {
			if q == '\'' && i+1 < bound && r.data[i+1] == '\'' {
				i++
				continue
			}
			return i + 1
		}
	}
	return bound
}
func (r *sourceRenderer) scalarEnd(n *yaml.Node, start, bound int, inFlow bool) int {
	if start >= bound {
		return start
	}
	if r.data[start] == '"' || r.data[start] == '\'' {
		return r.quoteEnd(start, bound)
	}
	if n.Style&(yaml.LiteralStyle|yaml.FoldedStyle) != 0 {
		end := bound
		indicatorEnd := bytes.IndexByte(r.data[start:bound], '\n')
		if indicatorEnd < 0 {
			indicatorEnd = bound - start
		}
		if bytes.Contains(r.data[start:start+indicatorEnd], []byte("+")) {
			for end > start && (r.data[end-1] == ' ' || r.data[end-1] == '\t') {
				end--
			}
			return end
		}
		// The following sibling's indentation is not part of the scalar.
		for end > start && (r.data[end-1] == ' ' || r.data[end-1] == '\t' || r.data[end-1] == '\r' || r.data[end-1] == '\n') {
			end--
		}
		return end
	}
	end := start
	for end < bound {
		c := r.data[end]
		if inFlow && (c == ',' || c == ']' || c == '}') || c == ':' && end+1 < bound && (r.data[end+1] == ' ' || r.data[end+1] == '\n') {
			break
		}
		if c == '#' && (end == start || r.data[end-1] == ' ' || r.data[end-1] == '\t') {
			break
		}
		if c == '\n' || c == '\r' {
			break
		}
		end++
	}
	if !inFlow && end < bound && (r.data[end] == '\r' || r.data[end] == '\n') {
		tail := end
		for tail < bound {
			next := bytes.IndexByte(r.data[tail:bound], '\n')
			if next < 0 {
				break
			}
			tail += next + 1
			rowEnd := bound
			if next := bytes.IndexByte(r.data[tail:bound], '\n'); next >= 0 {
				rowEnd = tail + next
			}
			row := r.data[tail:rowEnd]
			if len(row) >= 3 && (bytes.Equal(row[:3], []byte("...")) || bytes.Equal(row[:3], []byte("---"))) {
				rest := bytes.TrimSpace(row[3:])
				separated := len(row) == 3 || row[3] == ' ' || row[3] == '\t' || row[3] == '\r'
				if separated && (len(rest) == 0 || rest[0] == '#') {
					break
				}
			}
			line := bytes.TrimSpace(row)
			if len(line) > 0 && line[0] != '#' {
				end = rowEnd
			}
		}
	}
	for end > start && (r.data[end-1] == ' ' || r.data[end-1] == '\t' || r.data[end-1] == '\r' || r.data[end-1] == '\n') {
		end--
	}
	return end
}

// Compare parsed node values directly: authored YAML may contain timestamp or
// other tagged scalars that cannot be represented through JSON normalization.
func sameNodeValues(a, b *yaml.Node) bool {
	if a.Kind != b.Kind || a.Tag != b.Tag || a.Value != b.Value || a.HeadComment != b.HeadComment || a.LineComment != b.LineComment || a.FootComment != b.FootComment || len(a.Content) != len(b.Content) {
		return false
	}
	for i := range a.Content {
		if !sameNodeValues(a.Content[i], b.Content[i]) {
			return false
		}
	}
	return true
}
func samePresentation(a, b *yaml.Node) bool {
	if a.Kind != b.Kind || a.Tag != b.Tag || a.Value != b.Value || a.Style != b.Style || len(a.Content) != len(b.Content) {
		return false
	}
	for i := range a.Content {
		if !samePresentation(a.Content[i], b.Content[i]) {
			return false
		}
	}
	return true
}
func (r *sourceRenderer) render(before, after *yaml.Node) ([]byte, error) {
	span := r.spans[before]
	if samePresentation(before, after) {
		return r.data[span.start:span.end], nil
	}
	if before.Kind == after.Kind && (before.Kind == yaml.MappingNode || before.Kind == yaml.SequenceNode) && len(before.Content) > 0 && len(after.Content) > 0 {
		if r.json || before.Style&yaml.FlowStyle != 0 {
			return r.flow(before, after)
		}
		return r.block(before, after)
	}
	return r.fresh(after, before.Column-1, before.Style&yaml.FlowStyle != 0)
}
func (r *sourceRenderer) fresh(n *yaml.Node, indent int, flow bool) ([]byte, error) {
	if r.json {
		return utils.RenderRepairJSON(n, []byte(""))
	}
	copy := clone(n)
	copy.HeadComment, copy.FootComment, copy.LineComment = "", "", ""
	if flow && copy.Kind == yaml.ScalarNode && copy.Tag == "!!str" && strings.ContainsAny(copy.Value, "{}[],") {
		copy.Style = yaml.DoubleQuotedStyle
	}
	if flow && (copy.Kind == yaml.MappingNode || copy.Kind == yaml.SequenceNode) {
		copy.Style |= yaml.FlowStyle
	}
	out, err := utils.RenderRepairYAML(copy, nil, r.format)
	if err != nil {
		return nil, err
	}
	out = bytes.TrimRight(out, "\r\n")
	if indent > 0 {
		out = bytes.ReplaceAll(out, []byte("\n"), []byte("\n"+strings.Repeat(" ", indent)))
	}
	if bytes.Contains(r.data, []byte("\r\n")) {
		out = bytes.ReplaceAll(out, []byte("\n"), []byte("\r\n"))
	}
	return out, nil
}
func (r *sourceRenderer) oldEntry(before *yaml.Node, after *yaml.Node, i int) int {
	entries := r.entries[before]
	if entries == nil {
		entries = make(map[string]int, len(before.Content))
		step := 1
		if before.Kind == yaml.MappingNode {
			step = 2
		}
		for j := 0; j < len(before.Content); j += step {
			n := before.Content[j]
			entries[fmt.Sprintf("%d:%d", n.Line, n.Column)] = j
			if step == 2 {
				entries["key:"+n.Value] = j
			}
		}
		r.entries[before] = entries
	}
	n := after.Content[i]
	if n.Line > 0 {
		if j, ok := entries[fmt.Sprintf("%d:%d", n.Line, n.Column)]; ok {
			return j
		}
	}
	if before.Kind == yaml.MappingNode {
		if j, ok := entries["key:"+n.Value]; ok {
			return j
		}
	}
	return -1
}

func (r *sourceRenderer) flow(before, after *yaml.Node) ([]byte, error) {
	span := r.spans[before]
	step := 1
	if before.Kind == yaml.MappingNode {
		step = 2
	}
	first := r.spans[before.Content[0]].start
	last := r.spans[before.Content[len(before.Content)-1]].end
	prefix, suffix := r.data[span.start:first], r.data[last:span.end]
	separator := []byte(", ")
	if r.json {
		separator = []byte(",")
	}
	if len(before.Content) == step && bytes.Contains(prefix, []byte("\n")) {
		separator = append([]byte(","), prefix[1:]...)
	}
	if len(before.Content) > step {
		separator = r.data[r.spans[before.Content[step-1]].end:r.spans[before.Content[step]].start]
	}
	colon := []byte(": ")
	if r.json {
		colon = []byte(":")
	}
	if step == 2 {
		colon = r.data[r.spans[before.Content[0]].end:r.spans[before.Content[1]].start]
	}
	var out bytes.Buffer
	out.Write(prefix)
	for i := 0; i < len(after.Content); i += step {
		j := r.oldEntry(before, after, i)
		if i > 0 {
			gap := separator
			if j >= step {
				previous := before.Content[j-1]
				gap = r.data[r.spans[previous].end:r.spans[before.Content[j]].start]
				if r.oldEntry(before, after, i-step) != j-step && previous.LineComment != "" {
					// In flow syntax a comment after the comma belongs to the preceding item.
					if k := bytes.Index(gap, []byte(previous.LineComment)); k >= 0 {
						gap = append(bytes.Clone(bytes.TrimRight(gap[:k], " \t")), gap[k+len(previous.LineComment):]...)
					}
				}
			}
			out.Write(gap)
		}
		for k := 0; k < step; k++ {
			if k > 0 {
				gap := colon
				if j >= 0 {
					gap = r.data[r.spans[before.Content[j]].end:r.spans[before.Content[j+1]].start]
				}
				out.Write(gap)
			}
			var value []byte
			var err error
			if j >= 0 {
				value, err = r.render(before.Content[j+k], after.Content[i+k])
			} else {
				value, err = r.fresh(after.Content[i+k], 0, true)
			}
			if err != nil {
				return nil, err
			}
			out.Write(value)
		}
	}
	out.Write(suffix)
	return out.Bytes(), nil
}

// A block entry owns its dash, leading comments and trailing comments. Keep
// separating whitespace outside the entry so moves carry the author's comments.
func (r *sourceRenderer) blockEntryStart(block *yaml.Node, i int) (start, token int) {
	entry := block.Content[i]
	token = r.spans[entry].start
	if block.Kind == yaml.SequenceNode {
		pos := token
		for pos > 0 && strings.IndexByte(" \t\r\n", r.data[pos-1]) >= 0 {
			pos--
		}
		if pos > 0 && r.data[pos-1] == '-' {
			token = pos - 1
		}
	}
	start = token
	if entry.HeadComment == "" {
		return
	}
	head := strings.Split(entry.HeadComment, "\n")
	row := bytes.LastIndexByte(r.data[:token], '\n') + 1
	for j := len(head) - 1; j >= 0 && row > 0; j-- {
		prev := bytes.LastIndexByte(r.data[:row-1], '\n') + 1
		line := bytes.TrimSpace(r.data[prev:row])
		if string(line) != strings.TrimSpace(head[j]) {
			return token, token
		}
		start = prev + len(r.data[prev:row]) - len(bytes.TrimLeft(r.data[prev:row], " \t"))
		row = prev
	}
	return
}

func (r *sourceRenderer) blockEntryEnd(block *yaml.Node, i, step int) int {
	first, last := block.Content[i], block.Content[i+step-1]
	end := r.spans[last].end
	if last.LineComment != "" || first.LineComment != "" {
		lineEnd := bytes.IndexAny(r.data[end:], "\r\n")
		if lineEnd < 0 {
			lineEnd = len(r.data) - end
		}
		comment := string(bytes.TrimSpace(r.data[end : end+lineEnd]))
		if comment != "" && (comment == last.LineComment || comment == first.LineComment) {
			end += lineEnd
		}
	}
	if first.FootComment != "" {
		for _, line := range strings.Split(first.FootComment, "\n") {
			next := bytes.IndexByte(r.data[end:], '\n')
			if next < 0 {
				break
			}
			row := end + next + 1
			stop := bytes.IndexAny(r.data[row:], "\r\n")
			if stop < 0 {
				stop = len(r.data) - row
			}
			if string(bytes.TrimSpace(r.data[row:row+stop])) != strings.TrimSpace(line) {
				break
			}
			end = row + stop
		}
	}
	return end
}
func (r *sourceRenderer) block(before, after *yaml.Node) ([]byte, error) {
	step := 1
	if before.Kind == yaml.MappingNode {
		step = 2
	}
	indent := max(0, before.Column-1)
	newline := "\n"
	if bytes.Contains(r.data, []byte("\r\n")) {
		newline = "\r\n"
	}
	separator := newline + strings.Repeat(" ", indent)
	var out bytes.Buffer
	for i := 0; i < len(after.Content); i += step {
		j := r.oldEntry(before, after, i)
		if i > 0 {
			if j >= step {
				start, _ := r.blockEntryStart(before, j)
				out.Write(r.data[r.blockEntryEnd(before, j-step, step):start])
			} else {
				if bytes.HasSuffix(out.Bytes(), []byte(newline)) {
					out.WriteString(strings.TrimPrefix(separator, newline))
				} else {
					out.WriteString(separator)
				}
			}
		}
		if j >= 0 {
			start, _ := r.blockEntryStart(before, j)
			out.Write(r.data[start:r.spans[before.Content[j]].start])
			if step == 2 {
				if !isBlock(before.Content[j+1]) && isBlock(after.Content[i+1]) {
					pair := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: after.Content[i : i+2]}
					value, err := r.fresh(pair, indent, false)
					if err != nil {
						return nil, err
					}
					out.Write(value)
					out.Write(r.data[r.spans[before.Content[j+1]].end:r.blockEntryEnd(before, j, step)])
					continue
				}
				key, err := r.render(before.Content[j], after.Content[i])
				if err != nil {
					return nil, err
				}
				out.Write(key)
				from, to := r.spans[before.Content[j]], r.spans[before.Content[j+1]]
				gap := r.data[from.end:to.start]
				value, err := r.render(before.Content[j+1], after.Content[i+1])
				if err != nil {
					return nil, err
				}
				out.Write(gap)
				out.Write(value)
			} else {
				value, err := r.render(before.Content[j], after.Content[i])
				if err != nil {
					return nil, err
				}
				out.Write(value)
			}
			out.Write(r.data[r.spans[before.Content[j+step-1]].end:r.blockEntryEnd(before, j, step)])
		} else {
			if step == 1 {
				out.WriteString("- ")
			}
			var n *yaml.Node
			if step == 2 {
				n = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: after.Content[i : i+2]}
			} else {
				n = after.Content[i]
			}
			valueIndent := indent
			if step == 1 {
				valueIndent += 2
			}
			value, err := r.fresh(n, valueIndent, false)
			if err != nil {
				return nil, err
			}
			out.Write(value)
		}
	}
	return out.Bytes(), nil
}
func isBlock(n *yaml.Node) bool {
	return (n.Kind == yaml.MappingNode || n.Kind == yaml.SequenceNode) && n.Style&yaml.FlowStyle == 0 && len(n.Content) > 0
}
