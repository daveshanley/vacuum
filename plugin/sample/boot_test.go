package sample

import (
	"github.com/daveshanley/vacuum/plugin"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLoadFunctions_LoadSample(t *testing.T) {
	pm := plugin.CreatePluginManager()
	Boot(pm)
	assert.Equal(t, 2, pm.LoadedFunctionCount())
}
