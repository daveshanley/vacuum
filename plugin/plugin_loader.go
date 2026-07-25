package plugin

import (
	"fmt"
	"github.com/daveshanley/vacuum/functions/core"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/plugin/javascript"
	"go.yaml.in/yaml/v4"
	"os"
	"path/filepath"
	"plugin"
	"reflect"
	"strings"
)

// UnsupportedFunctionFormatError reports custom JavaScript authored for a runtime Vacuum does not execute.
type UnsupportedFunctionFormatError struct {
	Path string
}

func (e UnsupportedFunctionFormatError) Error() string {
	return fmt.Sprintf(`'%s' is a Spectral function, not a vacuum function.

vacuum does not run npm modules. there is no module resolution, no bundler and no node
runtime, so imports from '@stoplight/spectral-*' and 'export default' cannot be honoured.

a vacuum custom function is a plain script defining two globals:

  function getSchema() { return {"name": "myFunction"}; }
  function runRule(input) { return []; }

rewrite this function as a vacuum custom function, or remove it from the functions directory.`, e.Path)
}

func skippedUnsupportedFunctionMessage(err UnsupportedFunctionFormatError) string {
	return fmt.Sprintf("%s\n\nthis function has been skipped, all other functions in the directory still loaded.", err.Error())
}

// LoadFunctions will load custom functions found in the supplied path
func LoadFunctions(path string, silence bool) (*Manager, error) {

	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	pm := CreatePluginManager()
	var unsupportedFunctionErrs []error

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
			if isUnsupportedSpectralFunction(p) {
				// migrating users often keep Spectral functions alongside vacuum ones, so skip
				// what cannot run rather than abandoning every function in the directory. this
				// goes to stderr so it cannot corrupt machine-readable output on stdout, and is
				// never silenced, a skipped function silently changes linting results.
				formatErr := UnsupportedFunctionFormatError{Path: fPath}
				unsupportedFunctionErrs = append(unsupportedFunctionErrs, formatErr)
				_, _ = fmt.Fprintf(os.Stderr, "✗ %s\n", skippedUnsupportedFunctionMessage(formatErr))
				continue
			}

			function := javascript.NewJSRuleFunction(fName, string(p))

			// found something
			if !silence {
				fmt.Printf("● Located custom javascript function: '%s' from file: %s\n", function.GetSchema().Name, fPath)
			}
			// check if the function is valid
			sErr := function.CheckScript()

			if sErr != nil {
				fmt.Printf("✗ Failed to load function '%s': %s\n", fName, sErr.Error())
				continue // Skip registering invalid functions
			} else {
				if !silence {
					fmt.Printf("✓ Successfully validated JavaScript function: '%s'\n", fName)
				}
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
	if len(unsupportedFunctionErrs) > 0 && pm.LoadedFunctionCount() == 0 {
		return nil, fmt.Errorf("no vacuum custom functions loaded: %w", unsupportedFunctionErrs[0])
	}
	return pm, nil
}

// isUnsupportedSpectralFunction reports whether a JavaScript file is authored for Spectral rather
// than vacuum. Vacuum functions expose runRule(), so a file providing it is treated as ours no
// matter what else it contains, and only files that both lack it and carry module or Spectral
// markers are rejected.
func isUnsupportedSpectralFunction(data []byte) bool {
	body := string(data)
	if strings.Contains(body, "runRule") {
		return false
	}
	return strings.Contains(body, "createRulesetFunction") ||
		strings.Contains(body, "@stoplight/spectral") ||
		strings.Contains(body, "export default")
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
