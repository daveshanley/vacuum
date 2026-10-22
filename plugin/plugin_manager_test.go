package plugin

import (
	"github.com/daveshanley/vacuum/functions/core"
	"github.com/pb33f/testify/assert"
	"testing"
)

func TestPluginManager_RegisterFunction(t *testing.T) {

	pm := CreatePluginManager()

	pm.RegisterFunction("defined", core.Defined{})
	assert.Len(t, pm.GetCustomFunctions(), 1)

}

func TestManagerAutoFixRegistration(t *testing.T) {
	pm := CreatePluginManager()
	pm.RegisterAutoFixFunction("repair", nil)
	_, ok := pm.GetAutoFixFunctions()["repair"]
	assert.True(t, ok)
}
