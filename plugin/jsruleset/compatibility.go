package jsruleset

import (
	"fmt"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/dop251/goja"
	"github.com/pb33f/go-yaml"
)

// These exports use the same native implementations as declarative rulesets.
var compatibilityModules = map[string]string{
	"@stoplight/spectral-core": `export function createRulesetFunction(config,fn) {
 if(!config || typeof fn!=='function')throw Error('createRulesetFunction requires schemas and a function');
 const input=__compileSchema(config.input), options=__compileSchema(config.options);
 const validate=opts=>{if(config.options===null){if(opts!=null)throw Error('function does not accept options')}else options(opts)};
 const wrapped=(value,opts,ctx)=>{try{input(value)}catch(e){return config.errorOnInvalidInput?[{message:'invalid input'}]:undefined}validate(opts);return fn(value,opts,ctx)};
 wrapped.__validateOptions=validate;return wrapped;
}`,

	"@stoplight/spectral-functions": `export const truthy=__native.truthy, falsy=__native.falsy, defined=__native.defined, undefined=__native.undefined, pattern=__native.pattern, enumeration=__native.enumeration, length=__native.length, alphabetical=__native.alphabetical, casing=__native.casing, xor=__native.xor, or=__native.or, schema=__native.schema;`,
	"@stoplight/spectral-formats":   `export const oas2 = d => !!d && d.swagger === '2.0'; export const oas3 = d => !!d && typeof d.openapi === 'string' && /^3\./.test(d.openapi); export const oas3_0 = d => oas3(d) && /^3\.0\./.test(d.openapi); export const oas3_1 = d => oas3(d) && /^3\.1\./.test(d.openapi); export const asyncapi2 = d => !!d && typeof d.asyncapi === 'string' && /^2\./.test(d.asyncapi); export const asyncapi3 = d => !!d && typeof d.asyncapi === 'string' && /^3\./.test(d.asyncapi);`,
}

// Both package names use the same supported exports and native behavior.
func compatibilityModule(name string) (string, bool) {
	if suffix, ok := strings.CutPrefix(name, "@openlint/openlint-"); ok {
		name = "@stoplight/spectral-" + suffix
	}
	source, ok := compatibilityModules[name]
	return source, ok
}

func (s *Scope) registerCompatibility() {
	s.registerSchemaValidator()
	native := s.vm.NewObject()
	for _, name := range []string{"truthy", "falsy", "defined", "undefined", "pattern", "enumeration", "length", "alphabetical", "casing", "xor", "or", "schema"} {
		_ = native.Set(name, func(call goja.FunctionCall) goja.Value {
			if s.current == nil {
				panic(s.vm.NewTypeError("native functions require a rule invocation"))
			}
			if goja.IsUndefined(call.Argument(0)) {
				switch name {
				case "truthy", "defined":
					return s.vm.ToValue([]any{map[string]any{"message": "value must be defined"}})
				case "falsy", "undefined":
					return s.vm.ToValue([]any{})
				}
			}
			node, err := s.valueNode(call.Argument(0), map[*goja.Object]bool{})
			if err != nil {
				panic(s.vm.NewTypeError("%s", err))
			}
			rfc := model.RuleFunctionContext{Rule: s.current.Rule, RuleAction: s.current.RuleAction, Given: "$", ReportError: s.current.ReportError}
			action := *rfc.RuleAction
			action.Field = ""
			action.Function = name
			rfc.RuleAction = &action
			rfc.Options = call.Argument(1).Export()
			function := s.native[name]
			if function == nil {
				panic(s.vm.NewTypeError("native function unavailable: %s", name))
			}
			valid, issues := model.ValidateRuleFunctionContextAgainstSchema(function, rfc)
			if !valid || len(issues) > 0 {
				panic(s.vm.NewTypeError("invalid %s options: %v", name, issues))
			}
			results := function.RunRule([]*yaml.Node{node}, rfc)
			out := make([]any, 0, len(results))
			for _, result := range results {
				finding := map[string]any{"message": result.Message}
				if result.Path != "" && result.Path != "unknown" {
					relative, err := pathSegments(result.Path)
					if err != nil {
						panic(s.vm.NewTypeError("native result path: %s", err))
					}
					ctx := call.Argument(2).ToObject(s.vm)
					var prefix []any
					if err := s.vm.ExportTo(ctx.Get("path"), &prefix); err != nil {
						panic(s.vm.NewTypeError("native function requires context.path"))
					}
					finding["path"] = append(prefix, relative...)
				}
				out = append(out, finding)
			}
			return s.vm.ToValue(out)
		})
		_ = native.Get(name).ToObject(s.vm).Set("__nativeName", name)
	}
	if err := s.vm.Set("__native", native); err != nil {
		panic(fmt.Errorf("register native functions: %w", err))
	}
}

// SetNativeFunctions binds existing engine functions before document execution.
func (s *Scope) SetNativeFunctions(functions map[string]model.RuleFunction) { s.native = functions }
