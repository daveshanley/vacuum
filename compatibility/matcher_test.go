package compatibility

import (
	"testing"

	"github.com/daveshanley/vacuum/model/reports"
	"github.com/pb33f/testify/require"
)

func TestMatchResults(t *testing.T) {
	code := "r"
	a, b, root := []string{"a"}, []string{"b"}, []string{}
	specific := resultMatch{Code: &code, Path: &a}
	broad := resultMatch{Code: &code}
	first := reports.SpectralReport{Code: code, Path: a}
	second := reports.SpectralReport{Code: code, Path: b}
	tests := []struct {
		name   string
		want   []resultMatch
		actual []reports.SpectralReport
		match  bool
	}{
		{"empty", []resultMatch{}, []reports.SpectralReport{}, true},
		{"missing output", []resultMatch{specific}, nil, false},
		{"extra finding", []resultMatch{}, []reports.SpectralReport{first}, false},
		{"duplicate required", []resultMatch{specific, specific}, []reports.SpectralReport{first, second}, false},
		{"broad before specific", []resultMatch{broad, specific}, []reports.SpectralReport{first, second}, true},
		{"wrong path", []resultMatch{specific}, []reports.SpectralReport{second}, false},
		{"null is not root", []resultMatch{{Path: &root}}, []reports.SpectralReport{{Code: code}}, false},
		{"explicit root", []resultMatch{{Path: &root}}, []reports.SpectralReport{{Path: root}}, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) { require.Equal(t, test.match, matchResults(test.want, test.actual)) })
	}
}
