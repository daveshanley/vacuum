package jsruleset

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pb33f/testify/require"
)

// TestSpectralCompatibility compares both CLIs against the same expected findings.
// Run with VACUUM_BINARY and SPECTRAL_BINARY set to installed executables:
//
//	go test ./plugin/jsruleset -run '^TestSpectralCompatibility$' -count=1
//
// Spectral must be version 6.17.0. This test does not install dependencies.
func TestSpectralCompatibility(t *testing.T) {
	vacuum, spectral := os.Getenv("VACUUM_BINARY"), os.Getenv("SPECTRAL_BINARY")
	if vacuum == "" && spectral == "" {
		t.Skip("set VACUUM_BINARY and SPECTRAL_BINARY to run the CLI comparison")
	}
	require.NotEmpty(t, vacuum, "VACUUM_BINARY is required")
	require.NotEmpty(t, spectral, "SPECTRAL_BINARY is required")

	run := func(t *testing.T, binary string, allowFindings bool, args ...string) []byte {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		command := exec.CommandContext(ctx, binary, args...)
		var stderr bytes.Buffer
		command.Stderr = &stderr
		output, err := command.Output()
		require.NoError(t, ctx.Err(), "%s timed out", binary)
		var exitError *exec.ExitError
		if allowFindings && errors.As(err, &exitError) && exitError.ExitCode() == 1 {
			err = nil
		}
		require.NoError(t, err, "%s: %s", binary, stderr.String())
		return output
	}
	require.Equal(t, "6.17.0", strings.TrimSpace(string(run(t, spectral, false, "--version"))))
	fixture, err := filepath.Abs("testdata/compatibility")
	require.NoError(t, err)
	api, rules := filepath.Join(fixture, "api.yaml"), filepath.Join(fixture, "rules.mjs")
	// Source ranges retain each engine's existing mapping contract.
	expected := []map[string]any{{
		"code": "title-required", "path": []any{"info", "title"},
		"message": "Use the expected title.", "severity": float64(0),
	}}
	for _, engine := range []struct {
		name, binary string
		args         []string
	}{
		{"vacuum", vacuum, []string{"spectral-report", api, "-r", rules, "--stdout", "--no-pretty", "--no-style", "--no-update-check"}},
		{"spectral", spectral, []string{"lint", api, "-r", rules, "-f", "json", "--quiet"}},
	} {
		t.Run(engine.name, func(t *testing.T) {
			var findings []map[string]any
			require.NoError(t, json.Unmarshal(run(t, engine.binary, true, engine.args...), &findings))
			for _, finding := range findings {
				for key := range finding {
					if key != "code" && key != "path" && key != "message" && key != "severity" {
						delete(finding, key)
					}
				}
			}
			require.Equal(t, expected, findings)
		})
	}
}
