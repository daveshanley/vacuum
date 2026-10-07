package cmd

import (
	"github.com/pb33f/testify/require"
	"os"
	"os/exec"
	"testing"
)

func TestVacuumSubprocess(t *testing.T) {
	if os.Getenv("VACUUM_TEST_SUBPROCESS") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{os.Args[0]}, os.Args[i+1:]...)
			Execute("test", "test", "test")
			os.Exit(0)
		}
	}
	t.Fatal("missing command argument separator")
}

// Run the real exit path in an isolated working and configuration directory.
func runVacuum(t *testing.T, args ...string) ([]byte, int) {
	t.Helper()
	dir := t.TempDir()
	process := exec.Command(os.Args[0], append([]string{"-test.run=^TestVacuumSubprocess$", "--"}, args...)...)
	process.Dir = dir
	process.Env = append(os.Environ(), "VACUUM_TEST_SUBPROCESS=1", "XDG_CONFIG_HOME="+dir)
	output, err := process.CombinedOutput()
	if err == nil {
		return output, 0
	}
	var exit *exec.ExitError
	require.ErrorAs(t, err, &exit, "%s", output)
	return output, exit.ExitCode()
}
