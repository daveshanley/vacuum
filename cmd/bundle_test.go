package cmd

import (
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

func TestBundleCommand_Lazy_DefaultBehavior(t *testing.T) {
	tmpDir := t.TempDir()
	rootPath := filepath.Join(tmpDir, "root.yaml")
	schemasPath := filepath.Join(tmpDir, "schemas.yaml")
	outPath := filepath.Join(tmpDir, "out.yaml")

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

	cmd := newBundleTestCommand()
	cmd.SetArgs([]string{"-p", tmpDir, rootPath, outPath})

	err := cmd.Execute()
	require.NoError(t, err)

	output := readOutputFile(t, outPath)
	assert.Contains(t, output, "petName:")
	assert.Contains(t, output, "type: object")
}
