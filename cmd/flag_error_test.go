package cmd

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/pb33f/testify/require"
)

func TestFlagErrorsAreVisible(t *testing.T) {
	if os.Getenv("VACUUM_TEST_FLAG_ERROR") == "1" {
		for i, arg := range os.Args {
			if arg == "--" {
				os.Args = append([]string{os.Args[0]}, os.Args[i+1:]...)
				break
			}
		}
		Execute("test", "test", "test")
		os.Exit(0)
	}
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
			args := append([]string{"-test.run=^TestFlagErrorsAreVisible$", "--"}, tc.args...)
			process := exec.Command(os.Args[0], args...)
			process.Dir = t.TempDir()
			process.Env = append(os.Environ(), "VACUUM_TEST_FLAG_ERROR=1")
			output, err := process.CombinedOutput()
			var exit *exec.ExitError
			require.ErrorAs(t, err, &exit, "%s", output)
			require.Equal(t, ExitCodeInputError, exit.ExitCode())
			require.Equal(t, 1, strings.Count(string(output), tc.message), "%s", output)
		})
	}
}
