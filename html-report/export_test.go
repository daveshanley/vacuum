package html_report

import (
	"bytes"
	"github.com/pb33f/testify/require"
	"strings"
	"testing"
)

func TestWriteExportCompleteAndEscaped(t *testing.T) {
	rows := make([]ExportRow, 150)
	for i := range rows {
		rows[i] = ExportRow{Rule: "test", Message: `<script>alert('bad')</script>`, Source: "a&b.yaml", Line: 1, Column: 1}
	}
	var b bytes.Buffer
	require.NoError(t, WriteExport(&b, rows))
	require.Equal(t, 151, strings.Count(b.String(), "<tr>"))
	require.NotContains(t, b.String(), "<script>")
	require.Contains(t, b.String(), "&lt;script&gt;")
	require.Contains(t, b.String(), "a&amp;b.yaml")
}
