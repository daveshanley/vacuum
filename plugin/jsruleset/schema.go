package jsruleset

import (
	"github.com/dop251/goja"
	"github.com/pb33f/jsonschema/v6"
)

func (s *Scope) registerSchemaValidator() {
	_ = s.vm.Set("__compileSchema", func(call goja.FunctionCall) goja.Value {
		value := call.Argument(0)
		if goja.IsNull(value) || goja.IsUndefined(value) {
			return s.vm.ToValue(func(goja.FunctionCall) goja.Value { return goja.Undefined() })
		}
		compiler := jsonschema.NewCompiler()
		if err := compiler.AddResource("schema.json", value.Export()); err != nil {
			panic(s.vm.NewTypeError("invalid function schema: %s", err))
		}
		schema, err := compiler.Compile("schema.json")
		if err != nil {
			panic(s.vm.NewTypeError("invalid function schema: %s", err))
		}
		return s.vm.ToValue(func(c goja.FunctionCall) goja.Value {
			if err := schema.Validate(c.Argument(0).Export()); err != nil {
				panic(s.vm.NewTypeError("function validation: %s", err))
			}
			return goja.Undefined()
		})
	})
}
