package plugin

import (
	"github.com/daveshanley/vacuum/model"
	"gopkg.in/yaml.v3"
)

type FunctionSchema func() model.RuleFunctionSchema
type FunctionHook func(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult

type Manager struct {
	customFunctions map[string]model.RuleFunction
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
