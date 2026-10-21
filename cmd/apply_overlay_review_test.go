package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/testify/require"
)

func TestApplyOverlayDoesNotOverwriteOmittedOutputOverlay(t *testing.T) {
	for _, stdin := range []bool{false, true} {
		for _, overlay := range []string{"overlay: 1.0.0\ninfo: {title: Test, version: '1'}\nactions: []\n", `{"overlay":"1.0.0","actions":[]}`} {
			dir := t.TempDir()
			input, destination := filepath.Join(dir, "input.yaml"), filepath.Join(dir, "overlay.yaml")
			spec := "openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {}\n"
			require.NoError(t, os.WriteFile(input, []byte(spec), 0600))
			require.NoError(t, os.WriteFile(destination, []byte(overlay), 0600))
			args := []string{input, destination, "--prune-unused"}
			if stdin {
				args = []string{destination, "--stdin", "--prune-unused"}
			}
			cmd := GetApplyOverlayCommand()
			cmd.SetArgs(args)
			cmd.SetIn(strings.NewReader(spec))
			_, _, err := captureProcessStreams(t, cmd.Execute)
			require.ErrorContains(t, err, "please supply output file path")
			var exit *ExitError
			require.ErrorAs(t, err, &exit)
			require.Equal(t, 2, exit.Code)
			after, err := os.ReadFile(destination)
			require.NoError(t, err)
			require.Equal(t, overlay, string(after))
		}
	}
}

func TestApplyOverlayExplicitYAMLDestination(t *testing.T) {
	for _, extension := range []string{".yaml", ".yml", ".YAML"} {
		output := filepath.Join(t.TempDir(), "out"+extension)
		cmd := GetApplyOverlayCommand()
		cmd.SetArgs([]string{"--stdin", output, "--prune-unused"})
		cmd.SetIn(strings.NewReader(`{"openapi":"3.1.0","info":{"title":"Test","version":"1"},"paths":{}}`))
		stdout, _, err := captureProcessStreams(t, cmd.Execute)
		require.NoError(t, err)
		require.Contains(t, stdout, "Transforms applied")
		require.NotContains(t, stdout, "Overlay applied")
		raw, err := os.ReadFile(output)
		require.NoError(t, err)
		require.False(t, json.Valid(raw))
		require.Contains(t, normalizedYAML(t, raw).(map[string]any), "openapi")
	}
}

func TestApplyOverlayTransformMissingOutput(t *testing.T) {
	for _, args := range [][]string{{"spec.yaml", "--prune-unused"}, {"--stdin", "--prune-unused"}} {
		cmd := GetApplyOverlayCommand()
		cmd.SetArgs(args)
		_, _, err := captureProcessStreams(t, cmd.Execute)
		require.ErrorContains(t, err, "please supply output file path")
	}
}
