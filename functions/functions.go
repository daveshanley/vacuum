// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package functions

import (
	"sync"

	"github.com/daveshanley/vacuum/functions/core"
	openapi_functions "github.com/daveshanley/vacuum/functions/openapi"
	"github.com/daveshanley/vacuum/functions/owasp"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/plugin"
)

type customFunction struct {
	functionHook plugin.FunctionHook
	schemaHook   plugin.FunctionSchema
}

type functionsModel struct {
	functions       map[string]model.RuleFunction
	customFunctions map[string]customFunction
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
		funcs := make(map[string]model.RuleFunction)
		customFuncs := make(map[string]customFunction)
		functionsSingleton = &functionsModel{
			functions:       funcs,
			customFunctions: customFuncs,
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
		funcs["postResponseSuccess"] = openapi_functions.PostResponseSuccess{}
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
		funcs["oasExampleMissing"] = openapi_functions.ExamplesMissing{}
		funcs["oasExampleExternal"] = openapi_functions.ExamplesExternalCheck{}
		funcs["oasExampleSchema"] = openapi_functions.ExamplesSchema{}
		funcs["oasSchema"] = core.Schema{} // can't see a reason to duplicate this yet.
		funcs["oasDescriptions"] = openapi_functions.OperationDescription{}
		funcs["oasDescriptionDuplication"] = openapi_functions.DescriptionDuplication{}
		funcs["oasComponentDescriptions"] = openapi_functions.ComponentDescription{}
		funcs["oasOperationTags"] = openapi_functions.OperationTags{}
		funcs["oasOpFormDataConsumeCheck"] = openapi_functions.FormDataConsumeCheck{}
		funcs["oasDiscriminator"] = openapi_functions.OAS2Discriminator{}
		funcs["oasDiscriminator"] = openapi_functions.OAS2Discriminator{}
		funcs["oasParamDescriptions"] = openapi_functions.ParameterDescription{}
		funcs["oasOpSecurityDefined"] = openapi_functions.OperationSecurityDefined{}
		funcs["oas2OpSecurityDefined"] = openapi_functions.OAS2OperationSecurityDefined{}
		funcs["oasPolymorphicAnyOf"] = openapi_functions.PolymorphicAnyOf{}
		funcs["oasPolymorphicOneOf"] = openapi_functions.PolymorphicOneOf{}
		funcs["oasDocumentSchema"] = openapi_functions.OASSchema{}
		funcs["oasAPIServers"] = openapi_functions.APIServers{}
		funcs["noAmbiguousPaths"] = openapi_functions.AmbiguousPaths{}
		funcs["noVerbsInPath"] = openapi_functions.VerbsInPaths{}
		funcs["pathsKebabCase"] = openapi_functions.PathsKebabCase{}
		funcs["oasOpErrorResponse"] = openapi_functions.Operation4xResponse{}
		funcs["schemaTypeCheck"] = openapi_functions.SchemaTypeCheck{}

		// add owasp functions used by the owasp rules
		funcs["owaspHeaderDefinition"] = owasp.HeaderDefinition{}
		funcs["owaspDefineErrorDefinition"] = owasp.DefineErrorDefinition{}
		funcs["owaspCheckSecurity"] = owasp.CheckSecurity{}
		funcs["owaspCheckErrorResponse"] = owasp.CheckErrorResponse{}
		funcs["owaspRatelimitRetryAfter"] = owasp.RatelimitRetry429{}
		funcs["owaspArrayLimit"] = owasp.ArrayLimit{}
		funcs["owaspJWTBestPractice"] = owasp.JWTBestPractice{}
		funcs["owaspAuthInsecureSchemes"] = owasp.AuthInsecureSchemes{}
		funcs["owaspNoNumericIds"] = owasp.NoNumericIds{}
		funcs["owaspNoBasicAuth"] = owasp.NoBasicAuth{}
		funcs["owaspNoApiKeyInUrl"] = owasp.NoApiKeyInUrl{}
		funcs["owaspNoCredentialsInUrl"] = owasp.NoCredentialsInUrl{}
		funcs["owaspStringLimit"] = owasp.StringLimit{}
		funcs["owaspStringRestricted"] = owasp.StringRestricted{}
		funcs["owaspIntegerLimit"] = owasp.IntegerLimit{}
		funcs["owaspIntegerFormat"] = owasp.IntegerFormat{}
		funcs["owaspNoAdditionalProperties"] = owasp.NoAdditionalProperties{}
		funcs["owaspNoAdditionalPropertiesConstrained"] = owasp.AdditionalPropertiesConstrained{}
		funcs["owaspHostsHttps"] = owasp.HostsHttps{}

	})

	return functionsSingleton
}

func (fm functionsModel) RegisterCustomFunction(name string, function plugin.FunctionHook, schema plugin.FunctionSchema) {
	fm.customFunctions[name] = customFunction{functionHook: function, schemaHook: schema}
}

func (fm functionsModel) GetAllFunctions() map[string]model.RuleFunction {
	return fm.functions
}

func (fm functionsModel) FindFunction(functionName string) model.RuleFunction {
	return fm.functions[functionName]
}
