package utils

import (
	"github.com/pb33f/testify/require"
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
