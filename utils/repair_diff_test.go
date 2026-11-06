package utils

import (
	"github.com/pb33f/testify/require"
	"io"
	"strings"
	"testing"
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
