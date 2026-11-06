package utils

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
)

// RepairDiff renders a bounded, line-oriented review of a candidate. It uses a
// single changed range, with three context lines, so time stays linear even for
// large generated specs. Truncated output is marked and is not an applyable patch.
func RepairDiff(original, candidate []byte) string {
	var out strings.Builder
	_ = writeRepairDiff(&out, original, candidate, true)
	return out.String()
}

// WriteRepairDiff writes every changed line without truncation. It shares the
// linear changed-range calculation with RepairDiff, but is suitable for review
// before acceptance. Control bytes are escaped to prevent terminal commands.
// Output errors, including short writes, stop review and are returned to callers.
func WriteRepairDiff(output io.Writer, original, candidate []byte) error {
	return writeRepairDiff(output, original, candidate, false)
}

func writeRepairDiff(output io.Writer, original, candidate []byte, bounded bool) error {
	out := bufio.NewWriterSize(repairDiffWriter{output}, 16384)
	if bytes.Equal(original, candidate) {
		_, _ = out.WriteString("No changes.\n")
		return out.Flush()
	}
	next := func(data []byte, offset int) int {
		if n := bytes.IndexByte(data[offset:], '\n'); n >= 0 {
			return offset + n + 1
		}
		return len(data)
	}
	previous := func(data []byte, end int) int {
		if end > 0 && data[end-1] == '\n' {
			end--
		}
		return bytes.LastIndexByte(data[:end], '\n') + 1
	}
	count := func(data []byte) int {
		n := bytes.Count(data, []byte{'\n'})
		if len(data) > 0 && data[len(data)-1] != '\n' {
			n++
		}
		return n
	}
	first, line := 0, 1
	for first < len(original) && first < len(candidate) {
		a, b := next(original, first), next(candidate, first)
		if !bytes.Equal(original[first:a], candidate[first:b]) {
			break
		}
		first = a
		line++
	}
	ae, be := len(original), len(candidate)
	for ae > first && be > first {
		a, b := previous(original, ae), previous(candidate, be)
		if a < first || b < first || !bytes.Equal(original[a:ae], candidate[b:be]) {
			break
		}
		ae, be = a, b
	}
	start := first
	for i := 0; i < 3 && start > 0; i++ {
		start = previous(original, start)
		line--
	}
	endA, endB := ae, be
	for i := 0; i < 3 && endA < len(original); i++ {
		endA = next(original, endA)
	}
	for i := 0; i < 3 && endB < len(candidate); i++ {
		endB = next(candidate, endB)
	}
	fmt.Fprintf(out, "--- original\n+++ candidate\n@@ -%d,%d +%d,%d @@\n", line, count(original[start:endA]), line, count(candidate[start:endB]))
	write := func(prefix string, data []byte) {
		for offset, n := 0, 0; offset < len(data); n++ {
			if bounded && n == 200 {
				fmt.Fprintf(out, "... %d more lines omitted from this view ...\n", count(data[offset:]))
				break
			}
			end := next(data, offset)
			segment := data[offset:end]
			hasNewline := segment[len(segment)-1] == '\n'
			if hasNewline {
				segment = segment[:len(segment)-1]
			}
			truncated := bounded && len(segment) > 4096
			if truncated {
				segment = segment[:4096]
			}
			out.WriteString(prefix)
			if bounded {
				safe := strings.Map(func(r rune) rune {
					if (r < 32 && r != '\t') || r == 127 {
						return '�'
					}
					return r
				}, string(segment))
				out.WriteString(safe)
			} else {
				// Write source slices directly; a minified document must not
				// require a second document-sized string merely for display.
				start := 0
				for i, c := range segment {
					if (c < 32 && c != '\t') || c == 127 {
						out.Write(segment[start:i])
						fmt.Fprintf(out, "\\x%02x", c)
						start = i + 1
					}
				}
				out.Write(segment[start:])
			}
			if truncated {
				out.WriteString(" ... line truncated")
			}
			out.WriteByte('\n')
			if !hasNewline {
				out.WriteString("\\ No newline at end of file\n")
			}
			offset = end
		}
	}
	write(" ", original[start:first])
	write("-", original[first:ae])
	write("+", candidate[first:be])
	write(" ", original[ae:endA])
	return out.Flush()
}

// repairDiffWriter also checks large writes that bypass bufio's own buffer.
type repairDiffWriter struct{ io.Writer }

func (w repairDiffWriter) Write(data []byte) (int, error) {
	n, err := w.Writer.Write(data)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	return n, err
}
