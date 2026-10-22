package cmd

import (
	"strings"
	"testing"

	"github.com/pb33f/testify/require"
)

func TestFlagErrorsAreVisible(t *testing.T) {
	for _, tc := range []struct {
		name    string
		args    []string
		message string
	}{
		{"missing functions directory", []string{"lint", "api.yaml", "--functions"}, "flag needs an argument: --functions"},
		{"missing shorthand directory", []string{"lint", "api.yaml", "-f"}, "flag needs an argument"},
		{"invalid timeout", []string{"lint", "api.yaml", "--timeout", "bad"}, "invalid argument"},
		{"unknown flag", []string{"lint", "api.yaml", "--unknown-941"}, "unknown flag: --unknown-941"},
		{"root flag", []string{"--functions"}, "flag needs an argument: --functions"},
		{"report flag", []string{"spectral-report", "api.yaml", "--functions"}, "flag needs an argument: --functions"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output, code := runVacuum(t, tc.args...)
			require.Equal(t, ExitCodeInputError, code)
			require.Equal(t, 1, strings.Count(string(output), tc.message), "%s", output)
		})
	}
}
