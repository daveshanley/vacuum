package plugin

import (
	"errors"
	"fmt"
	"github.com/daveshanley/vacuum/functions/core"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/plugin/javascript"
	"github.com/pb33f/go-yaml"
	"os"
	"path/filepath"
	"plugin"
	"reflect"
	"strings"
)

// LoadFunctions will load custom functions found in the supplied path
func LoadFunctions(path string, silence bool) (*Manager, error) {

	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	pm := CreatePluginManager()
	var loadErrors []error

	for _, entry := range dirEntries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".so") {
			fPath := filepath.Join(path, entry.Name())

			// found something
			if !silence {
				fmt.Printf("● Located custom function plugin: %s\n", fPath)
			}
			// let's try and open it.
			p, e := plugin.Open(fPath)
			if e != nil {
				return nil, e
			}

			// look up the Boot function and store as a Symbol
			var bootFunc plugin.Symbol
			bootFunc, err = p.Lookup("Boot")
			if err != nil {
				return nil, err
			}

			// lets go pedro!
			if bootFunc != nil {
				bootFunc.(func(*Manager))(pm)
			} else {
				fmt.Printf("✗ Unable to boot plugin\n")
			}
		}

		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".js") {
			fPath := filepath.Join(path, entry.Name())
			fName := strings.Split(entry.Name(), ".")[0]

			// let's try and read the file
			p, e := os.ReadFile(fPath)
			if e != nil {
				return nil, e
			}

			function := javascript.NewJSRuleFunction(fName, string(p))

			if err := function.CheckScript(); err != nil {
				loadErr := fmt.Errorf("unable to load custom function %q: %w; vacuum custom functions use a plain script with runRule(input) and getSchema(); npm modules and ES module imports/exports are not supported", fPath, err)
				loadErrors = append(loadErrors, loadErr)
				// A skipped function changes the active rules. Keep the warning visible,
				// but keep stdout available for machine-readable report output.
				_, _ = fmt.Fprintln(os.Stderr, loadErr)
				continue
			}
			if !silence {
				fmt.Printf("✓ Successfully validated JavaScript function: '%s'\n", fName)
			}

			// register core functions with this custom function.
			RegisterCoreFunctions(function)

			// register this function with the plugin manager using the schema name
			schemaName := function.GetSchema().Name
			pm.RegisterFunction(schemaName, function)

			if !silence {
				fmt.Printf("● Registered custom function: '%s' -> available for use in rulesets\n", schemaName)
			}
		}
	}
	if pm.LoadedFunctionCount() == 0 && len(loadErrors) > 0 {
		return nil, fmt.Errorf("no vacuum custom functions loaded: %w", errors.Join(loadErrors...))
	}
	return pm, nil
}

var extractInput = func(input any) *yaml.Node {
	var y yaml.Node
	switch reflect.TypeOf(input).Kind() {
	case reflect.String, reflect.Int32, reflect.Int64, reflect.Float32, reflect.Float64, reflect.Bool:
		_ = yaml.Unmarshal([]byte(fmt.Sprintf("%v", input)), &y)
	case reflect.Map, reflect.Slice:
		_ = y.Encode(input)
	}
	if y.Kind == yaml.DocumentNode {
		return y.Content[0]
	} else {
		return &y
	}
}

var coreError = func() {
	if r := recover(); r != nil {
		fmt.Printf("✗ Core function '%s' had a panic attack via JavaScript: %s\n", r, "truthy")
	}
}

var loadFunc = func(function model.RuleFunction) javascript.CoreFunction {
	return func(input any, context model.RuleFunctionContext) []model.RuleFunctionResult {
		defer coreError()
		extracted := extractInput(input)
		results := function.RunRule([]*yaml.Node{extracted}, context)
		return results
	}
}

var truthy = loadFunc(&core.Truthy{})
var falsy = loadFunc(&core.Falsy{})
var alphabetical = loadFunc(&core.Alphabetical{})
var casing = loadFunc(&core.Casing{})
var defined = loadFunc(&core.Defined{})
var enum = loadFunc(&core.Enumeration{})
var length = loadFunc(&core.Length{})
var pattern = loadFunc(&core.Pattern{})
var undefined = loadFunc(&core.Undefined{})
var xor = loadFunc(&core.Xor{})
var blank = loadFunc(&core.Blank{})

func RegisterCoreFunctions(rule javascript.JSEnabledRuleFunction) {
	rule.RegisterCoreFunction("truthy", truthy)
	rule.RegisterCoreFunction("falsy", falsy)
	rule.RegisterCoreFunction("alphabetical", alphabetical)
	rule.RegisterCoreFunction("casing", casing)
	rule.RegisterCoreFunction("defined", defined)
	rule.RegisterCoreFunction("enum", enum)
	rule.RegisterCoreFunction("length", length)
	rule.RegisterCoreFunction("pattern", pattern)
	rule.RegisterCoreFunction("undefined", undefined)
	rule.RegisterCoreFunction("xor", xor)
	rule.RegisterCoreFunction("blank", blank)
}
