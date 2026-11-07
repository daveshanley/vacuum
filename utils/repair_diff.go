package utils

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/aymanbagabas/go-udiff"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// RepairDiff renders diff hunks with three context lines. Preview limits are
// marked explicitly; bounded previews are not patches.
func RepairDiff(original, candidate []byte) string {
	var out strings.Builder
	if err := writeRepairDiff(&out, original, candidate, true); err != nil {
		return fmt.Sprintf("Cannot display fix preview: %v\n", err)
	}
	return out.String()
}

// WriteRepairDiff writes every changed line in contextual diff hunks without
// truncation. It omits unchanged gaps between hunks and is suitable for review
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
	// Trim common outer content before matching, so a large unchanged tail
	// does not allocate a line table for the entire document.
	before, after := string(original[start:endA]), string(candidate[start:endB])
	edits, err := repairLineEdits(before, after)
	if err != nil {
		return err
	}
	diff, err := udiff.ToUnifiedDiff("original", "candidate", before, edits, 3)
	if err != nil {
		return err
	}
	fmt.Fprint(out, "--- original\n+++ candidate\n")
	write := func(prefix, segment string) {
		hasNewline := strings.HasSuffix(segment, "\n")
		segment = strings.TrimSuffix(segment, "\n")
		if hasNewline {
			segment = strings.TrimSuffix(segment, "\r")
		}
		truncated := bounded && len(segment) > 4096
		if truncated {
			end := 4096
			for end > 0 && !utf8.RuneStart(segment[end]) {
				end--
			}
			segment = segment[:end]
		}
		out.WriteString(prefix)
		// Escape terminal controls without reflowing YAML/JSON or copying a long line.
		start := 0
		for i, c := range []byte(segment) {
			if (c < 32 && c != '\t') || c == 127 {
				out.WriteString(segment[start:i])
				fmt.Fprintf(out, "\\x%02x", c)
				start = i + 1
			}
		}
		out.WriteString(segment[start:])
		if truncated {
			out.WriteString(" ... line truncated")
		}
		out.WriteByte('\n')
		if !hasNewline {
			out.WriteString("\\ No newline at end of file\n")
		}
	}

	for _, hunk := range diff.Hunks {
		fromCount, toCount := 0, 0
		for _, row := range hunk.Lines {
			if row.Kind != udiff.Insert {
				fromCount++
			}
			if row.Kind != udiff.Delete {
				toCount++
			}
		}
		from, to := hunk.FromLine+line-1, hunk.ToLine+line-1
		if fromCount == 0 {
			from--
		}
		if toCount == 0 {
			to--
		}
		fmt.Fprintf(out, "@@ -%d,%d +%d,%d @@\n", from, fromCount, to, toCount)
		for index, row := range hunk.Lines {
			if bounded && index == 200 {
				fmt.Fprintf(out, "... %d more lines omitted from this view ...\n", len(hunk.Lines)-index)
				break
			}
			prefix := " "
			if row.Kind == udiff.Delete {
				prefix = "-"
			}
			if row.Kind == udiff.Insert {
				prefix = "+"
			}
			write(prefix, row.Content)
		}
		if err := out.Flush(); err != nil {
			return err
		}
	}

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

// Match whole lines with a linear-space bisect search. Disable the deadline:
// timing out must not label unchanged source as a replacement during review.
func repairLineEdits(before, after string) ([]udiff.Edit, error) {
	// go-diff skips surrogate code points and three reserved rune values.
	if err := checkRepairDiffLineLimit(before, after, unicode.MaxRune-2048-3); err != nil {
		return nil, err
	}
	matcher := diffmatchpatch.New()
	matcher.DiffTimeout = 0
	left, right, lines := matcher.DiffLinesToRunes(before, after)
	changes := matcher.DiffCharsToLines(matcher.DiffMainRunes(left, right, false), lines)
	var edits []udiff.Edit
	offset := 0
	for _, change := range changes {
		switch change.Type {
		case diffmatchpatch.DiffEqual:
			offset += len(change.Text)
		case diffmatchpatch.DiffDelete:
			edits = append(edits, udiff.Edit{Start: offset, End: offset + len(change.Text)})
			offset += len(change.Text)
		case diffmatchpatch.DiffInsert:
			// Keep an adjacent replacement together for the hunk formatter.
			if len(edits) > 0 && edits[len(edits)-1].End == offset && edits[len(edits)-1].New == "" {
				edits[len(edits)-1].New = change.Text
			} else {
				edits = append(edits, udiff.Edit{Start: offset, End: offset, New: change.Text})
			}
		}
	}
	return edits, nil
}

// Only count distinct lines near the encoder limit. Common lines in the two
// versions share a token and must not cause a false rejection.
func checkRepairDiffLineLimit(before, after string, limit int) error {
	if strings.Count(before, "\n")+strings.Count(after, "\n")+2 <= limit {
		return nil
	}
	seen := make(map[string]struct{})
	for _, text := range []string{before, after} {
		for line := range strings.SplitAfterSeq(text, "\n") {
			if line == "" {
				continue
			}
			seen[line] = struct{}{}
			if len(seen) > limit {
				return fmt.Errorf("fix diff exceeds %d distinct lines; no files written", limit)
			}
		}
	}
	return nil
}
