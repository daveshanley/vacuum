package cmd

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
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

func collectIndexedBaseNames(t *testing.T, specBytes []byte, baseFlag, specFilePath string, allFiles bool) []string {
	t.Helper()
	doc, err := libopenapi.NewDocumentWithConfiguration(
		specBytes,
		buildBundleDocConfig(baseFlag, specFilePath, allFiles, true, false, slog.Default()),
	)
	require.NoError(t, err)

	_, err = doc.BuildV3Model()
	require.NoError(t, err)

	rolodex := doc.GetRolodex()
	require.NotNil(t, rolodex)

	seen := make(map[string]struct{})
	for _, idx := range rolodex.GetIndexes() {
		seen[filepath.Base(idx.GetSpecAbsolutePath())] = struct{}{}
	}

	var names []string
	for name := range seen {
		names = append(names, name)
	}
	return names
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

func TestBundleCommand_Lazy_NestedSpec_RefsResolveCorrectly(t *testing.T) {
	tmpDir := t.TempDir()
	rootPath := filepath.Join(tmpDir, "nested", "root.yaml")
	outPath := filepath.Join(tmpDir, "out.yaml")

	writeTestFile(t, filepath.Join(tmpDir, "schemas.yaml"), `
components:
  schemas:
    Pet:
      type: object
      properties:
        petName:
          type: string
`)
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
                $ref: "../schemas.yaml#/components/schemas/Pet"
`)

	chdirForTest(t, tmpDir)

	cmd := newBundleTestCommand()
	cmd.SetArgs([]string{"-p", tmpDir, filepath.Join("nested", "root.yaml"), outPath})

	err := cmd.Execute()
	require.NoError(t, err)

	output := readOutputFile(t, outPath)
	assert.Contains(t, output, "petName:")
}

func TestBundleCommand_Eager_NestedSpec_RefsResolveCorrectly(t *testing.T) {
	tmpDir := t.TempDir()
	rootPath := filepath.Join(tmpDir, "nested", "root.yaml")
	outPath := filepath.Join(tmpDir, "out.yaml")

	writeTestFile(t, filepath.Join(tmpDir, "schemas.yaml"), `
components:
  schemas:
    Pet:
      type: object
      properties:
        petName:
          type: string
`)
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
                $ref: "../schemas.yaml#/components/schemas/Pet"
`)

	chdirForTest(t, tmpDir)

	cmd := newBundleTestCommand()
	cmd.SetArgs([]string{"-a", "-p", tmpDir, filepath.Join("nested", "root.yaml"), outPath})

	err := cmd.Execute()
	require.NoError(t, err)

	output := readOutputFile(t, outPath)
	assert.Contains(t, output, "petName:")
}

func TestBundleCommand_Eager_NoBase_DefaultsToSpecDir(t *testing.T) {
	specDir := t.TempDir()
	otherDir := t.TempDir()
	rootPath := filepath.Join(specDir, "root.yaml")
	outPath := filepath.Join(otherDir, "out.yaml")

	writeTestFile(t, filepath.Join(specDir, "schemas.yaml"), `
components:
  schemas:
    Pet:
      type: object
      properties:
        petName:
          type: string
`)
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

	chdirForTest(t, otherDir)

	cmd := newBundleTestCommand()
	cmd.SetArgs([]string{"-a", rootPath, outPath})

	err := cmd.Execute()
	require.NoError(t, err)

	output := readOutputFile(t, outPath)
	assert.Contains(t, output, "petName:")
}

func TestBundleCommand_Eager_ComposedMode(t *testing.T) {
	tmpDir := t.TempDir()
	rootPath := filepath.Join(tmpDir, "root.yaml")
	outPath := filepath.Join(tmpDir, "out.yaml")

	writeTestFile(t, filepath.Join(tmpDir, "schemas.yaml"), `
components:
  schemas:
    Pet:
      type: object
`)
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

	cmd := newBundleTestCommand()
	cmd.SetArgs([]string{"-a", "-c", "-p", tmpDir, rootPath, outPath})

	err := cmd.Execute()
	require.NoError(t, err)

	output := readOutputFile(t, outPath)
	assert.Contains(t, output, "components:")
	assert.Contains(t, output, "Pet:")
	assert.NotContains(t, output, "schemas.yaml#")
}

func TestBundleCommand_Eager_StdinRequiresBase(t *testing.T) {
	cmd := newBundleTestCommand()
	cmd.SetIn(bytes.NewBufferString(`
openapi: 3.1.0
info:
  title: Test API
  version: "1"
paths: {}
`))
	cmd.SetArgs([]string{"-i", "-o", "-a"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.EqualError(t, err, "--all-files with --stdin requires --base to be set (no spec path to derive from)")
}

func TestBundleCommand_AllFilesFlagRegistration(t *testing.T) {
	cmd := GetBundleCommand()
	flag := cmd.Flags().Lookup("all-files")
	require.NotNil(t, flag)
	assert.Equal(t, "a", flag.Shorthand)
}

func TestBuildBundleDocConfig_Defaults_LocalFSNil(t *testing.T) {
	cfg := buildBundleDocConfig(".", "root.yaml", false, true, false, slog.Default())

	assert.Nil(t, cfg.LocalFS)
	assert.Equal(t, "root.yaml", cfg.SpecFilePath)
	assert.True(t, cfg.AllowRemoteReferences)
	assert.True(t, cfg.ExcludeExtensionRefs)
}

func TestBuildBundleDocConfig_AllFiles_LocalFSSet(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestFile(t, filepath.Join(tmpDir, "hello.yaml"), "hello: world")

	cfg := buildBundleDocConfig(tmpDir, "root.yaml", true, true, false, slog.Default())

	require.NotNil(t, cfg.LocalFS)
	file, err := cfg.LocalFS.Open("hello.yaml")
	require.NoError(t, err)
	require.NoError(t, file.Close())
}

func TestBuildBundleDocConfig_SpecFilePathPassedThrough(t *testing.T) {
	cfg := buildBundleDocConfig("/tmp/specs", filepath.Join("nested", "root.yaml"), true, true, false, slog.Default())

	assert.Equal(t, filepath.Join("nested", "root.yaml"), cfg.SpecFilePath)
}

func TestBuildBundleDocConfig_RemoteAndExtRefs(t *testing.T) {
	cfg := buildBundleDocConfig(".", "root.yaml", true, false, true, slog.Default())

	assert.False(t, cfg.AllowRemoteReferences)
	assert.False(t, cfg.ExcludeExtensionRefs)
}

func TestBundleDocConfig_EagerIndexesUnreferencedFile_LazyDoesNot(t *testing.T) {
	tmpDir := t.TempDir()
	rootPath := filepath.Join(tmpDir, "root.yaml")

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
	writeTestFile(t, filepath.Join(tmpDir, "schemas.yaml"), `
components:
  schemas:
    Pet:
      type: object
`)
	writeTestFile(t, filepath.Join(tmpDir, "unused.yaml"), `
components:
  schemas:
    Ghost:
      type: object
`)

	rootBytes, err := os.ReadFile(rootPath)
	require.NoError(t, err)

	lazySeen := collectIndexedBaseNames(t, rootBytes, tmpDir, rootPath, false)
	eagerSeen := collectIndexedBaseNames(t, rootBytes, tmpDir, rootPath, true)

	assert.NotContains(t, lazySeen, "unused.yaml")
	assert.Contains(t, eagerSeen, "unused.yaml")
	assert.Contains(t, lazySeen, "schemas.yaml")
}
