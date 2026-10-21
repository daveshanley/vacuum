package plugin

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
)

type FunctionSchema func() model.RuleFunctionSchema
type FunctionHook func(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult

type Manager struct {
	autoFixFunctions map[string]model.AutoFixFunction
	customFunctions  map[string]model.RuleFunction
}

func CreatePluginManager() *Manager {
	return &Manager{
		customFunctions: make(map[string]model.RuleFunction),
	}
}

// RegisterFunction allows a custom function to be hooked in
func (pm *Manager) RegisterFunction(name string, ruleFunction model.RuleFunction) {
	pm.customFunctions[name] = ruleFunction
}

// LoadedFunctionCount returns the number of available and ready to use functions.
func (pm *Manager) LoadedFunctionCount() int {
	return len(pm.customFunctions)
}

func (pm *Manager) GetCustomFunctions() map[string]model.RuleFunction {
	return pm.customFunctions
}

// RegisterAutoFixFunction registers a named repair for rules using autoFixFunction.
// Repairs are available to lint --fix and the test harness.
func (pm *Manager) RegisterAutoFixFunction(name string, function model.AutoFixFunction) {
	if pm.autoFixFunctions == nil {
		pm.autoFixFunctions = make(map[string]model.AutoFixFunction)
	}
	pm.autoFixFunctions[name] = function
}

// GetAutoFixFunctions returns repairs registered by a Go plugin's Boot function.
func (pm *Manager) GetAutoFixFunctions() map[string]model.AutoFixFunction {
	return pm.autoFixFunctions
}
