package plugin

import (
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadFunctions_Nowhere(t *testing.T) {
	pm, err := LoadFunctions("nowhere", false)
	assert.Nil(t, pm)
	assert.Error(t, err)
}

func TestLoadFunctions(t *testing.T) {
	pm, err := LoadFunctions("../model/test_files", false)
	assert.NotNil(t, pm)
	assert.NoError(t, err)
	assert.Equal(t, 0, pm.LoadedFunctionCount())
}

func TestLoadFunctions_JavaScript_OK(t *testing.T) {
	pm, err := LoadFunctions("sample/js", false)
	assert.NotNil(t, pm)
	assert.NoError(t, err)
	assert.Equal(t, 7, pm.LoadedFunctionCount())
	assert.Equal(t, "uselessFunc",
		pm.GetCustomFunctions()["uselessFunc"].GetSchema().Name)
	assert.Equal(t, "checkForNameAndId",
		pm.GetCustomFunctions()["checkForNameAndId"].GetSchema().Name)
}

func TestLoadFunctions_Sample(t *testing.T) {
	pm, err := LoadFunctions("sample", false)
	if runtime.GOOS != "windows" { // windows does not support this feature, at all.
		assert.NotNil(t, pm)
		assert.NoError(t, err)
		assert.Equal(t, 0, pm.LoadedFunctionCount())
	}
}

func TestLoadFunctions_TestCompile(t *testing.T) {
	pm, err := LoadFunctions("sample", false)
	if runtime.GOOS != "windows" { // windows does not support this feature, at all.
		assert.NotNil(t, pm)
		assert.NoError(t, err)
		assert.Equal(t, 0, pm.LoadedFunctionCount())
	}
}

func TestLoadFunctionsInvalidScripts(t *testing.T) {
	for _, tc := range []struct {
		name   string
		script string
	}{
		{"Spectral module", `import { createRulesetFunction } from "@stoplight/spectral-core";
export default createRulesetFunction({}, () => []);`},
		{"syntax error", `function runRule(`},
		{"missing function", `function getSchema() { return {name: "missing"}; }`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "invalid.js"), []byte(tc.script), 0600))
			stdout, stderr := captureFunctionLoaderOutput(t)
			manager, err := LoadFunctions(dir, true)
			require.ErrorContains(t, err, "no vacuum custom functions loaded")
			require.ErrorContains(t, err, "invalid.js")
			require.ErrorContains(t, err, "npm modules and ES module imports/exports are not supported")
			require.Nil(t, manager)
			assert.Empty(t, readFunctionLoaderOutput(t, stdout))
			assert.Contains(t, readFunctionLoaderOutput(t, stderr), "unable to load custom function")
		})
	}
}

func TestLoadFunctionsKeepsValidScriptsAlongsideInvalidModules(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "spectral.js"), []byte(`import { createRulesetFunction } from "@stoplight/spectral-core";
export default createRulesetFunction({}, () => []);`), 0600))
	// Mentioning module syntax or Spectral in comments and strings does not make
	// an otherwise valid Vacuum function incompatible.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "valid.js"), []byte(`
// This replaces @stoplight/spectral createRulesetFunction.
function getSchema() { return {name: "valid", description: "export default is not used"}; }
function runRule(input) { return []; }
`), 0600))
	stdout, stderr := captureFunctionLoaderOutput(t)
	manager, err := LoadFunctions(dir, true)
	require.NoError(t, err)
	require.NotNil(t, manager)
	assert.Equal(t, 1, manager.LoadedFunctionCount())
	assert.Contains(t, manager.GetCustomFunctions(), "valid")
	assert.Empty(t, readFunctionLoaderOutput(t, stdout))
	assert.Contains(t, readFunctionLoaderOutput(t, stderr), "spectral.js")
}

func captureFunctionLoaderOutput(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	stdout, err := os.CreateTemp(t.TempDir(), "stdout")
	require.NoError(t, err)
	stderr, err := os.CreateTemp(t.TempDir(), "stderr")
	require.NoError(t, err)
	originalOut, originalErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = stdout, stderr
	t.Cleanup(func() {
		os.Stdout, os.Stderr = originalOut, originalErr
		_ = stdout.Close()
		_ = stderr.Close()
	})
	return stdout, stderr
}

func readFunctionLoaderOutput(t *testing.T, file *os.File) string {
	t.Helper()
	data, err := os.ReadFile(file.Name())
	require.NoError(t, err)
	return string(data)
}
