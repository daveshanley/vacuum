package plugin

import (
	"github.com/stretchr/testify/assert"
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
