// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package utils

import (
	"fmt"
	"github.com/aymanbagabas/go-udiff"
	"github.com/pb33f/testify/require"
	"io"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestRepairDiffShowsChangesAndBoundsOutput(t *testing.T) {
	require.Equal(t, "No changes.\n", RepairDiff([]byte("same"), []byte("same")))
	diff := RepairDiff([]byte("one\nold\nlast\n"), []byte("one\nnew\nlast\n"))
	require.Contains(t, diff, "-old\n+new\n")
	require.Contains(t, diff, " last")
	diff = RepairDiff([]byte(strings.Repeat("old\n", 10000)), []byte("\x1b[2Jnew"))
	require.Contains(t, diff, "omitted")
	require.NotContains(t, diff, "\x1b")
	require.Less(t, len(diff), 4096)
}

func TestRepairDiffCommonBlankTail(t *testing.T) {
	tail := strings.Repeat("\n", 1<<20)
	diff := RepairDiff([]byte("old\n"+tail), []byte("new\n"+tail))
	require.Contains(t, diff, "-old\n+new\n")
	require.Less(t, len(diff), 200)
	require.Contains(t, RepairDiff([]byte("old"), []byte("new")), "No newline at end of file")
}

func BenchmarkRepairDiffBlankTail(b *testing.B) {
	tail := strings.Repeat("\n", 1<<20)
	original, candidate := []byte("old\n"+tail), []byte("new\n"+tail)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RepairDiff(original, candidate)
	}
}

func TestWriteRepairDiffDoesNotOmitChanges(t *testing.T) {
	original := "first: old\n" + strings.Repeat("unchanged: value\n", 300) + "last: old\n"
	candidate := "first: new\n" + strings.Repeat("unchanged: value\n", 300) + "last: new\n"
	var output strings.Builder
	require.NoError(t, WriteRepairDiff(&output, []byte(original), []byte(candidate)))
	require.Contains(t, output.String(), "-last: old\n")
	require.Contains(t, output.String(), "+last: new\n")
	require.NotContains(t, output.String(), "omitted")
	output.Reset()
	original = `{"padding":"` + strings.Repeat("a", 10000) + `","last":"old"}`
	candidate = strings.Replace(original, `"last":"old"`, `"last":"new"`, 1)
	require.NoError(t, WriteRepairDiff(&output, []byte(original), []byte(candidate)))
	require.Contains(t, output.String(), "-"+original)
	require.Contains(t, output.String(), "+"+candidate)
	require.NotContains(t, output.String(), "truncated")
}

func TestWriteRepairDiffRejectsShortWrites(t *testing.T) {
	require.ErrorIs(t, WriteRepairDiff(&shortDiffWriter{}, []byte("old"), []byte(strings.Repeat("new", 10000))), io.ErrShortWrite)
}

type shortDiffWriter struct{ calls int }

func (w *shortDiffWriter) Write(data []byte) (int, error) {
	w.calls++
	if w.calls == 1 {
		return len(data), nil
	}
	return len(data) - 1, nil
}

func TestRepairDiffUsesSeparateContextHunks(t *testing.T) {
	original := "first: old\n" + strings.Repeat("unchanged: value\n", 300) + "last: old\n"
	candidate := "first: new\n" + strings.Repeat("unchanged: value\n", 300) + "last: new\n"
	for _, bounded := range []bool{false, true} {
		var out strings.Builder
		require.NoError(t, writeRepairDiff(&out, []byte(original), []byte(candidate), bounded))
		require.Contains(t, out.String(), "@@ -1,4 +1,4 @@\n-first: old\n+first: new\n")
		require.Contains(t, out.String(), "@@ -299,4 +299,4 @@")
		require.Equal(t, 6, strings.Count(out.String(), " unchanged: value\n"))
		require.NotContains(t, out.String(), "-unchanged")
		require.NotContains(t, out.String(), "+unchanged")
		require.Less(t, len(strings.Split(out.String(), "\n")), 20)
	}
}

func TestRepairDiffManySeparatedEdits(t *testing.T) {
	var original, candidate strings.Builder
	for i := 0; i < 80; i++ {
		fmt.Fprintf(&original, "item%d: old\n", i)
		fmt.Fprintf(&candidate, "item%d: new\n", i)
		for j := 0; j < 20; j++ {
			line := fmt.Sprintf("context%d_%d: unchanged\n", i, j)
			original.WriteString(line)
			candidate.WriteString(line)
		}
	}
	var out strings.Builder
	require.NoError(t, WriteRepairDiff(&out, []byte(original.String()), []byte(candidate.String())))
	require.Equal(t, 80, strings.Count(out.String(), "@@ -"))
	require.NotContains(t, out.String(), "-context")
	require.NotContains(t, out.String(), "+context")
}

func TestRepairDiffHunkCoordinatesAndNewlines(t *testing.T) {
	for _, tc := range []struct{ before, after, header string }{
		{"", "new\n", "@@ -0,0 +1,1 @@"},
		{"old\n", "", "@@ -1,1 +0,0 @@"},
		{"one\ntwo\nthree\nfour\nfive\nsix\nold\nend\n", "one\ntwo\nthree\nfour\nfive\nsix\nnew\nend\n", "@@ -4,5 +4,5 @@"},
		{"same", "same\n", "@@ -1,1 +1,1 @@"},
	} {
		var out strings.Builder
		require.NoError(t, WriteRepairDiff(&out, []byte(tc.before), []byte(tc.after)))
		require.Contains(t, out.String(), tc.header)
	}
}

func TestRepairDiffKeepsRepeatedContextAfterLargeEditBlocks(t *testing.T) {
	before := strings.Repeat("old: value\n", 50) + strings.Repeat("same: value\n", 1000) + strings.Repeat("old: value\n", 50)
	after := strings.Repeat("new: value\n", 50) + strings.Repeat("same: value\n", 1000) + strings.Repeat("new: value\n", 50)
	var out strings.Builder
	require.NoError(t, WriteRepairDiff(&out, []byte(before), []byte(after)))
	require.Equal(t, 2, strings.Count(out.String(), "@@ -"))
	require.Equal(t, 6, strings.Count(out.String(), " same: value\n"))
	require.NotContains(t, out.String(), "-same:")
	require.NotContains(t, out.String(), "+same:")
}

func TestRepairDiffEditsReconstructCandidate(t *testing.T) {
	for _, tc := range []struct{ before, after string }{
		{"", "new\n"}, {"old\n", ""}, {"same", "same\n"},
		{"a\nb\nc\nd\ne\nf\ng\nh\ni\nj\nk\nl\n", "a\ninserted\nb\nc\ne\nf\ng\nh\nchanged\nj\nk\nl\n"},
		{"a\nb\na\nb\na\nb\n", "a\nb\nb\na\nb\na\n"},
	} {
		edits, err := repairLineEdits(tc.before, tc.after)
		require.NoError(t, err)
		actual, err := udiff.Apply(tc.before, edits)
		require.NoError(t, err)
		require.Equal(t, tc.after, actual)
	}
}

func BenchmarkRepairDiffScatteredEdits(b *testing.B) {
	for _, count := range []int{1000, 5000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			var before, after strings.Builder
			for i := 0; i < count; i++ {
				fmt.Fprintf(&before, "item%d: old\n", i)
				fmt.Fprintf(&after, "item%d: new\n", i)
				for j := 0; j < 20; j++ {
					line := fmt.Sprintf("context%d_%d: unchanged\n", i, j)
					before.WriteString(line)
					after.WriteString(line)
				}
			}
			left, right := []byte(before.String()), []byte(after.String())
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := WriteRepairDiff(io.Discard, left, right); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestRepairDiffLineLimitCountsSharedLinesOnce(t *testing.T) {
	require.NoError(t, checkRepairDiffLineLimit("a\na\nb\n", "a\nb\nb\n", 2))
	require.NoError(t, checkRepairDiffLineLimit("", "", 2))
	require.ErrorContains(t, checkRepairDiffLineLimit("a\nb\n", "c\n", 2), "exceeds 2 distinct lines")
	require.ErrorContains(t, checkRepairDiffLineLimit("a", "a\n", 1), "exceeds 1 distinct lines")
}

func TestRepairDiffCRLFAndUTF8Boundary(t *testing.T) {
	before := []byte("title: before\r\nkeep: yes\r\n")
	after := []byte("title: after\r\nkeep: yes\r\n")
	diff := RepairDiff(before, after)
	if strings.Contains(diff, `\x0d`) {
		t.Fatalf("CRLF escaped as content: %s", diff)
	}
	long := strings.Repeat("a", 4095) + "é" + strings.Repeat("b", 20)
	diff = RepairDiff([]byte("before\n"), []byte(long+"\n"))
	if !utf8.ValidString(diff) {
		t.Fatal("preview splits a UTF-8 rune")
	}
}
