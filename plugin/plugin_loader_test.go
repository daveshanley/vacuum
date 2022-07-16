package plugin

import (
	"github.com/stretchr/testify/assert"
	"runtime"
	"testing"
)

func TestLoadFunctions_Nowhere(t *testing.T) {
	pm, err := LoadFunctions("nowhere")
	assert.Nil(t, pm)
	assert.Error(t, err)
}

func TestLoadFunctions(t *testing.T) {
	pm, err := LoadFunctions("../model/test_files")
	assert.NotNil(t, pm)
	assert.NoError(t, err)
	assert.Equal(t, 0, pm.LoadedFunctionCount())
}

func TestLoadFunctions_Sample(t *testing.T) {
	pm, err := LoadFunctions("sample")
	if runtime.GOOS != "windows" { // windows does not support this feature, at all.
		assert.NotNil(t, pm)
		assert.NoError(t, err)
		assert.Equal(t, 0, pm.LoadedFunctionCount())
	}
}

func TestLoadFunctions_TestCompile(t *testing.T) {
	pm, err := LoadFunctions("sample")
	if runtime.GOOS != "windows" { // windows does not support this feature, at all.
		assert.NotNil(t, pm)
		assert.NoError(t, err)
		assert.Equal(t, 0, pm.LoadedFunctionCount())
	}
}
