package plugin

import (
	"github.com/daveshanley/vacuum/functions/core"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPluginManager_RegisterFunction(t *testing.T) {

	pm := CreatePluginManager()

	pm.RegisterFunction("defined", core.Defined{})
	assert.Len(t, pm.GetCustomFunctions(), 1)

}
