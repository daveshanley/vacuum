package cmd

import (
	"github.com/pb33f/testify/require"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyOverlayIssue996WithoutOverlay(t *testing.T) {
	const spec = "openapi: 3.0.3\ninfo: {title: Original, version: '1'}\npaths:\n  /pets:\n    get: {tags: [public], responses: {'200': {description: OK}}}\ncomponents:\n  schemas:\n    Unused: {type: string}\n"
	for _, stdin := range []bool{false, true} {
		for _, stdout := range []bool{false, true} {
			for _, flags := range [][]string{{"--include-tag", "public"}, {"--prune-unused"}, {"--include-tag", "public", "--prune-unused"}} {
				dir := t.TempDir()
				input := filepath.Join(dir, "in.yaml")
				output := filepath.Join(dir, "out.yaml")
				require.NoError(t, os.WriteFile(input, []byte(spec), 0600))
				args := []string{}
				if stdin {
					args = append(args, "--stdin")
				} else {
					args = append(args, input)
				}
				if stdout {
					args = append(args, "--stdout")
				} else {
					args = append(args, output)
				}
				args = append(args, flags...)
				cmd := GetApplyOverlayCommand()
				cmd.SetArgs(args)
				cmd.SetIn(strings.NewReader(spec))
				out, stderr, err := captureProcessStreams(t, cmd.Execute)
				require.NoError(t, err)
				require.Empty(t, stderr)
				if !stdout {
					b, e := os.ReadFile(output)
					require.NoError(t, e)
					out = string(b)
				}
				doc := normalizedYAML(t, []byte(out)).(map[string]any)
				require.Equal(t, "Original", doc["info"].(map[string]any)["title"])
				require.Contains(t, doc["paths"], "/pets")
				if strings.Contains(strings.Join(flags, " "), "--prune-unused") {
					require.NotContains(t, doc, "components")
				}
			}
		}
	}
}

func TestApplyOverlayRequiresActionAndRejectsExtraArguments(t *testing.T) {
	for _, args := range [][]string{{"in.yaml", "out.yaml"}, {"--stdin", "--stdout"}, {"in.yaml", "out.yaml", "--tag-match", "any"}, {"a", "b", "c", "d", "--prune-unused"}} {
		cmd := GetApplyOverlayCommand()
		cmd.SetArgs(args)
		_, _, err := captureProcessStreams(t, cmd.Execute)
		require.Error(t, err)
	}
}

func TestApplyOverlayWithoutOverlayInvalidInputKeepsStdoutClean(t *testing.T) {
	cmd := GetApplyOverlayCommand()
	cmd.SetArgs([]string{"--stdin", "--stdout", "--prune-unused"})
	cmd.SetIn(strings.NewReader("[invalid"))
	out, stderr, err := captureProcessStreams(t, cmd.Execute)
	require.Error(t, err)
	require.Empty(t, out)
	require.NotEmpty(t, stderr)
	var inputErr *ExitError
	require.ErrorAs(t, err, &inputErr)
	require.Equal(t, 2, inputErr.Code)
}
