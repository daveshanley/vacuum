package functions

import (
	"github.com/daveshanley/vaccum/functions/core"
	openapi_functions "github.com/daveshanley/vaccum/functions/openapi"
	"github.com/daveshanley/vaccum/model"
	"sync"
)

type functionsModel struct {
	functions map[string]model.RuleFunction
}

type Functions interface {
	GetAllFunctions() map[string]model.RuleFunction
	FindFunction(string) model.RuleFunction
}

var functionsSingleton *functionsModel
var functionGrab sync.Once

func MapBuiltinFunctions() Functions {

	functionGrab.Do(func() {
		funcs := make(map[string]model.RuleFunction)
		functionsSingleton = &functionsModel{
			functions: funcs,
		}

		// add known rules
		funcs["hello"] = openapi_functions.HelloFunction{}
		funcs["post_response_success"] = openapi_functions.PostResponseSuccess{}
		funcs["truthy"] = core.Truthy{}
	})

	return functionsSingleton
}

func (fm functionsModel) GetAllFunctions() map[string]model.RuleFunction {
	return fm.functions
}

func (fm functionsModel) FindFunction(functionName string) model.RuleFunction {
	return fm.functions[functionName]
}
