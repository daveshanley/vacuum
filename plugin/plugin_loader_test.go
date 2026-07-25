package plugin

import (
	"github.com/pb33f/testify/assert"
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

func TestIsUnsupportedSpectralFunction(t *testing.T) {
	spectral := []byte(`import { createRulesetFunction } from "@stoplight/spectral-core";
export default createRulesetFunction({}, () => []);
`)
	assert.True(t, isUnsupportedSpectralFunction(spectral))

	// a vacuum function is never rejected, even when it mentions Spectral
	vacuumFunc := []byte(`// a replacement for the Spectral createRulesetFunction approach
function getSchema() { return {"name": "myFunc"}; }
function runRule(input) { return []; }
`)
	assert.False(t, isUnsupportedSpectralFunction(vacuumFunc))
}

func TestLoadFunctionsFailsWhenOnlySpectralModulesArePresent(t *testing.T) {
	tmpDir := t.TempDir()
	assert.NoError(t, os.WriteFile(filepath.Join(tmpDir, "spectral.js"), []byte(`import { createRulesetFunction } from "@stoplight/spectral-core";
export default createRulesetFunction({}, () => []);
`), 0o600))

	pm, err := LoadFunctions(tmpDir, true)

	assert.Nil(t, pm)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no vacuum custom functions loaded")
	assert.Contains(t, err.Error(), "Spectral function, not a vacuum function")
	assert.NotContains(t, err.Error(), "all other functions in the directory still loaded")
}
