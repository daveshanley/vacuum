package harness

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// Schema returns the authoritative JSON Schema for a suite or result contract.
// Schemas derive their field definitions from the public Go types; semantic
// constraints are added here and also enforced by Suite.Validate.
func Schema(name string) ([]byte, error) {
	var typ reflect.Type
	switch name {
	case "suite":
		typ = reflect.TypeFor[Suite]()
	case "result":
		typ = reflect.TypeFor[Report]()
	default:
		return nil, fmt.Errorf("unknown schema %q; use suite or result", name)
	}
	root := contractSchema(typ)
	root["$schema"] = "https://json-schema.org/draft/2020-12/schema"
	root["$id"] = "https://quobix.com/vacuum/schemas/test-" + name + "-v1.json"
	root["title"] = "vacuum test " + name + " v1"
	return json.MarshalIndent(root, "", "  ")
}
func contractSchema(t reflect.Type) map[string]any {
	if t.Kind() == reflect.Pointer {
		return map[string]any{"anyOf": []any{contractSchema(t.Elem()), map[string]any{"type": "null"}}}
	}
	s := map[string]any{}
	switch t.Kind() {
	case reflect.Interface:
		return s
	case reflect.String:
		s["type"] = "string"
	case reflect.Bool:
		s["type"] = "boolean"
	case reflect.Int, reflect.Int64, reflect.Int32:
		s["type"] = "integer"
	case reflect.Slice:
		s["type"] = []string{"array", "null"}
		s["items"] = contractSchema(t.Elem())
	case reflect.Map:
		s["type"] = "object"
		s["additionalProperties"] = contractSchema(t.Elem())
	case reflect.Struct:
		props := map[string]any{}
		required := []string{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			tag := strings.Split(f.Tag.Get("json"), ",")
			if tag[0] == "-" {
				continue
			}
			name := tag[0]
			if name == "" {
				name = f.Name
			}
			props[name] = contractSchema(f.Type)
			if !strings.Contains(f.Tag.Get("json"), "omitempty") {
				required = append(required, name)
			}
		}
		s["type"] = "object"
		s["properties"] = props
		s["additionalProperties"] = false
		if len(required) > 0 {
			s["required"] = required
		}
		switch t {
		case reflect.TypeFor[Suite]():
			props["version"] = map[string]any{"const": 1}
			props["ruleset"] = map[string]any{"type": "string", "minLength": 1}
			s["required"] = []string{"version", "ruleset", "cases"}
			props["cases"].(map[string]any)["type"] = "array"
			props["cases"].(map[string]any)["minItems"] = 1
		case reflect.TypeFor[Case]():
			for _, key := range []string{"id", "input"} {
				props[key].(map[string]any)["minLength"] = 1
			}
			props["format"] = map[string]any{"enum": []string{"auto", "json-schema", "fragment"}}
			props["rules"].(map[string]any)["uniqueItems"] = true
		case reflect.TypeFor[Expectations]():
			delete(s, "required")
			s["anyOf"] = []any{map[string]any{"required": []string{"diagnostics"}, "properties": map[string]any{"diagnostics": map[string]any{"type": "array"}}}, map[string]any{"required": []string{"execution"}, "properties": map[string]any{"execution": map[string]any{"minProperties": 1}}}}
		case reflect.TypeFor[ExecutionMatch]():
			props["status"] = map[string]any{"enum": []string{"completed", "excluded"}}
		case reflect.TypeFor[DiagnosticMatch]():
			props["rule"].(map[string]any)["minLength"] = 1
			props["severity"] = map[string]any{"enum": []string{"error", "warn", "info", "hint"}}
		case reflect.TypeFor[Count]():
			s["anyOf"] = []any{map[string]any{"required": []string{"min"}, "properties": map[string]any{"min": map[string]any{"type": "integer", "minimum": 0}}}, map[string]any{"required": []string{"max"}, "properties": map[string]any{"max": map[string]any{"type": "integer", "minimum": 0}}}}
		case reflect.TypeFor[Options]():
			for _, key := range []string{"timeoutMs", "runTimeoutMs", "lookupTimeoutMs", "maxRuleConcurrency"} {
				props[key].(map[string]any)["minimum"] = 0
			}
		case reflect.TypeFor[Report]():
			props["version"] = map[string]any{"const": 1}
			props["status"] = map[string]any{"enum": []string{"passed", "failed", "error"}}
			props["exitCode"] = map[string]any{"enum": []int{0, 1, 2}}
		}
	}
	return s
}
