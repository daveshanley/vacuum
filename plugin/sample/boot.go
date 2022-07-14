package main

import "github.com/daveshanley/vacuum/plugin"

// Boot is called by the Manager when the module is located.
// all custom functions should be registered here.
func Boot(pm *plugin.Manager) {

	sampleA := SampleRuleFunction_A{}
	sampleB := SampleRuleFunction_B{}

	// register custom functions with vacuum plugin manager.
	pm.RegisterFunction(sampleA.GetSchema().Name, sampleA)
	pm.RegisterFunction(sampleB.GetSchema().Name, sampleB)
}
