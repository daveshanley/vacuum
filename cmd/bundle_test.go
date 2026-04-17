package cmd

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newBundleTestCommand() *cobra.Command {
	cmd := GetBundleCommand()
	cmd.PersistentFlags().StringP("base", "p", "", "")
	cmd.PersistentFlags().BoolP("remote", "u", true, "")
	cmd.PersistentFlags().Bool("ext-refs", false, "")
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	return cmd
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o644))
}

func readOutputFile(t *testing.T, path string) string {
	t.Helper()
	out, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(out)
}

func chdirForTest(t *testing.T, dir string) {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() {
		if chdirErr := os.Chdir(wd); chdirErr != nil {
			t.Errorf("restore cwd: %v", chdirErr)
		}
	})
}

func writeBundleFixture(t *testing.T, dir string) (string, string) {
	t.Helper()
	rootPath := filepath.Join(dir, "root.yaml")
	schemasPath := filepath.Join(dir, "schemas.yaml")

	writeTestFile(t, rootPath, `
openapi: 3.1.0
info:
  title: Test API
  version: "1"
paths:
  /pet:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "schemas.yaml#/components/schemas/Pet"
`)
	writeTestFile(t, schemasPath, `
components:
  schemas:
    Pet:
      type: object
      properties:
        petName:
          type: string
`)

	return rootPath, schemasPath
}

func decodeJSONOutput(t *testing.T, path string) map[string]any {
	t.Helper()
	out := readOutputFile(t, path)
	var doc map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &doc))
	return doc
}

func TestResolveBundleOutputFormat(t *testing.T) {
	tests := []struct {
		name       string
		formatFlag string
		stdOut     bool
		args       []string
		want       string
		wantErr    string
	}{
		{
			name:   "defaults to yaml for stdout",
			stdOut: true,
			args:   []string{"root.yaml"},
			want:   bundleOutputFormatYAML,
		},
		{
			name: "uses output json extension",
			args: []string{"root.yaml", "out.json"},
			want: bundleOutputFormatJSON,
		},
		{
			name: "uses output yaml extension",
			args: []string{"root.yaml", "out.yml"},
			want: bundleOutputFormatYAML,
		},
		{
			name: "unknown extension falls back to yaml",
			args: []string{"root.yaml", "out.bundle"},
			want: bundleOutputFormatYAML,
		},
		{
			name:       "explicit format overrides extension",
			formatFlag: "json",
			args:       []string{"root.yaml", "out.yaml"},
			want:       bundleOutputFormatJSON,
		},
		{
			name:       "yml alias resolves to yaml",
			formatFlag: "yml",
			args:       []string{"root.yaml", "out.json"},
			want:       bundleOutputFormatYAML,
		},
		{
			name:       "invalid format returns error",
			formatFlag: "toml",
			args:       []string{"root.yaml", "out.json"},
			wantErr:    `invalid bundle output format "toml", expected yaml or json`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveBundleOutputFormat(tt.formatFlag, tt.stdOut, tt.args)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBundleCommand_DefaultsToYAMLOutput(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "out.yaml")
	rootPath, _ := writeBundleFixture(t, tmpDir)

	cmd := newBundleTestCommand()
	cmd.SetArgs([]string{"-p", tmpDir, rootPath, outPath})

	err := cmd.Execute()
	require.NoError(t, err)

	output := readOutputFile(t, outPath)
	assert.Contains(t, output, "openapi: 3.1.0")
	assert.Contains(t, output, "petName:")
	assert.Contains(t, output, "type: object")
}

func TestBundleCommand_UsesJSONOutputForJSONExtension(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "out.json")
	rootPath, _ := writeBundleFixture(t, tmpDir)

	cmd := newBundleTestCommand()
	cmd.SetArgs([]string{"-p", tmpDir, rootPath, outPath})

	err := cmd.Execute()
	require.NoError(t, err)

	doc := decodeJSONOutput(t, outPath)
	assert.Equal(t, "3.1.0", doc["openapi"])

	paths, ok := doc["paths"].(map[string]any)
	require.True(t, ok)
	petPath, ok := paths["/pet"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, petPath, "get")
}

func TestBundleCommand_FormatFlagOverridesOutputExtension(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "out.json")
	rootPath, _ := writeBundleFixture(t, tmpDir)

	cmd := newBundleTestCommand()
	cmd.SetArgs([]string{"--format", "yaml", "-p", tmpDir, rootPath, outPath})

	err := cmd.Execute()
	require.NoError(t, err)

	output := readOutputFile(t, outPath)
	assert.Contains(t, output, "openapi: 3.1.0")
	assert.NotContains(t, output, `"openapi":`)
}

func TestBundleCommand_ComposedModeSupportsJSONOutput(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "out.json")
	rootPath, _ := writeBundleFixture(t, tmpDir)

	cmd := newBundleTestCommand()
	cmd.SetArgs([]string{"--composed", "--format", "json", "-p", tmpDir, rootPath, outPath})

	err := cmd.Execute()
	require.NoError(t, err)

	doc := decodeJSONOutput(t, outPath)
	components, ok := doc["components"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, components, "schemas")
}

func TestBundleCommand_InvalidFormatReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "out.yaml")
	rootPath, _ := writeBundleFixture(t, tmpDir)

	cmd := newBundleTestCommand()
	cmd.SetArgs([]string{"--format", "toml", "-p", tmpDir, rootPath, outPath})

	err := cmd.Execute()
	require.EqualError(t, err, `invalid bundle output format "toml", expected yaml or json`)
}
