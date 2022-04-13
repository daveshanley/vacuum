// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package functions

import (
	"github.com/daveshanley/vacuum/functions/core"
	openapi_functions "github.com/daveshanley/vacuum/functions/openapi"
	"github.com/daveshanley/vacuum/model"
	"sync"
)

type functionsModel struct {
	functions map[string]model.RuleFunction
}

// Functions is used to Query available functions loaded into vacuum
type Functions interface {

	// GetAllFunctions returns a model.RuleFunction map, the key is the function name.
	GetAllFunctions() map[string]model.RuleFunction

	// FindFunction returns a model.RuleFunction with the supplied name, or nil.
	FindFunction(string) model.RuleFunction
}

var functionsSingleton *functionsModel
var coreFunctionGrab sync.Once

// MapBuiltinFunctions will correctly map core (non-specific) functions to correct names.
func MapBuiltinFunctions() Functions {

	coreFunctionGrab.Do(func() {
		var funcs map[string]model.RuleFunction

		if functionsSingleton != nil {
			funcs = functionsSingleton.functions
		} else {
			funcs = make(map[string]model.RuleFunction)
			functionsSingleton = &functionsModel{
				functions: funcs,
			}
		}

		// add known rules
		funcs["truthy"] = &core.Truthy{}
		funcs["falsy"] = core.Falsy{}
		funcs["defined"] = core.Defined{}
		funcs["undefined"] = core.Undefined{}
		funcs["casing"] = core.Casing{}
		funcs["alphabetical"] = core.Alphabetical{}
		funcs["enumeration"] = core.Enumeration{}
		funcs["pattern"] = core.Pattern{}
		funcs["length"] = core.Length{}
		funcs["xor"] = core.Xor{}
		funcs["schema"] = core.Schema{}

		// add known OpenAPI rules
		funcs["post-response-success"] = openapi_functions.PostResponseSuccess{}
		funcs["oasOpSuccessResponse"] = openapi_functions.SuccessResponse{}
		funcs["oasOpIdUnique"] = openapi_functions.UniqueOperationId{}
		funcs["oasOpId"] = openapi_functions.OperationId{}
		funcs["oasOpSingleTag"] = openapi_functions.OperationSingleTag{}
		funcs["oasOpParams"] = openapi_functions.OperationParameters{}
		funcs["oasTagDefined"] = openapi_functions.TagDefined{}
		funcs["oasPathParam"] = openapi_functions.PathParameters{}
		funcs["refSiblings"] = openapi_functions.NoRefSiblings{}
		funcs["typedEnum"] = openapi_functions.TypedEnum{}
		funcs["duplicatedEnum"] = openapi_functions.DuplicatedEnum{}
		funcs["noEvalDescription"] = openapi_functions.NoEvalInDescriptions{}
		funcs["oasUnusedComponent"] = openapi_functions.UnusedComponent{}
		funcs["oasExample"] = openapi_functions.Examples{}
		funcs["oasSchema"] = core.Schema{} // can't see a reason to duplicate this yet.
		funcs["oasDescriptions"] = openapi_functions.OperationDescription{}
		funcs["oasDescriptionDuplication"] = openapi_functions.DescriptionDuplication{}
		funcs["oasOperationTags"] = openapi_functions.OperationTags{}
		funcs["oasOpFormDataConsumeCheck"] = openapi_functions.FormDataConsumeCheck{}
		funcs["oasDiscriminator"] = openapi_functions.OAS2Discriminator{}
		funcs["oasDiscriminator"] = openapi_functions.OAS2Discriminator{}
		funcs["oasParamDescriptions"] = openapi_functions.OAS2ParameterDescription{}
		funcs["oasOpSecurityDefined"] = openapi_functions.OperationSecurityDefined{}
		funcs["oas2OpSecurityDefined"] = openapi_functions.OAS2OperationSecurityDefined{}

	})

	return functionsSingleton
}

func (fm functionsModel) GetAllFunctions() map[string]model.RuleFunction {
	return fm.functions
}

func (fm functionsModel) FindFunction(functionName string) model.RuleFunction {
	return fm.functions[functionName]
}
