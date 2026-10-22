package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/testify/require"
)

func TestApplyOverlayIssue994JSON(t *testing.T) {
	const spec = `{"openapi":"3.0.4","info":{"title":"Before","version":"1"},"paths":{"/pets":{"get":{"tags":["public"],"responses":{"200":{"description":"OK"}}}}},"components":{"schemas":{"Unused":{"type":"integer","example":9007199254740993}}}}`
	for _, transforms := range []bool{false, true} {
		for _, stdout := range []bool{false, true} {
			dir := t.TempDir()
			overlay := filepath.Join(dir, "overlay.yaml")
			require.NoError(t, os.WriteFile(overlay, []byte("overlay: 1.0.0\ninfo: {title: Test, version: '1'}\nactions:\n  - target: $.info.title\n    update: After\n"), 0600))
			output := filepath.Join(dir, "out.json")
			args := []string{"--stdin", overlay}
			if stdout {
				args = append(args, "--stdout")
			} else {
				args = append(args, output)
			}
			if transforms {
				args = append(args, "--include-tag", "public")
			}
			cmd := GetApplyOverlayCommand()
			cmd.SetArgs(args)
			cmd.SetIn(strings.NewReader(spec))
			out, _, err := captureProcessStreams(t, cmd.Execute)
			require.NoError(t, err)
			if !stdout {
				b, e := os.ReadFile(output)
				require.NoError(t, e)
				out = string(b)
			}
			require.True(t, json.Valid([]byte(out)), out)
			require.Contains(t, out, `9007199254740993`)
			var doc map[string]json.RawMessage
			require.NoError(t, json.Unmarshal([]byte(out), &doc))
			require.JSONEq(t, `{"title":"After","version":"1"}`, string(doc["info"]))
			require.Contains(t, string(doc["paths"]), `"/pets"`)
		}
	}
}

func TestApplyOverlayYAMLInputToJSONDestination(t *testing.T) {
	dir := t.TempDir()
	input, output := filepath.Join(dir, "input.yaml"), filepath.Join(dir, "output.json")
	require.NoError(t, os.WriteFile(input, []byte("openapi: 3.1.0\ninfo: {title: YAML input, version: '1'}\npaths: {}\n"), 0o600))
	cmd := GetApplyOverlayCommand()
	cmd.SetArgs([]string{input, output, "--prune-unused"})
	_, _, err := captureProcessStreams(t, cmd.Execute)
	require.NoError(t, err)
	out, err := os.ReadFile(output)
	require.NoError(t, err)
	require.JSONEq(t, `{"openapi":"3.1.0","info":{"title":"YAML input","version":"1"},"paths":{}}`, string(out))
}
