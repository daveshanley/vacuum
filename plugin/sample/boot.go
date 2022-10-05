package sample

import "github.com/daveshanley/vacuum/plugin"

// Boot is called by the Manager when the module is located.
// all custom functions should be registered here.
func Boot(pm *plugin.Manager) {

	useless := uselessFunc{}
	checkSinglePath := checkSinglePathExists{}

	// register custom functions with vacuum plugin manager.
	pm.RegisterFunction(useless.GetSchema().Name, useless)
	pm.RegisterFunction(checkSinglePath.GetSchema().Name, checkSinglePath)
}
