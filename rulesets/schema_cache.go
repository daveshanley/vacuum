package rulesets

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/pb33f/jsonschema/v6"
)

// Keep one immutable compiled validator, not parsed user rulesets. The source
// key preserves sequential replacement of the exported RulesetSchema value.
var rulesetSchemaCache struct {
	sync.Mutex
	source string
	schema *jsonschema.Schema
	err    error
}

func compiledRulesetSchema(source string) (*jsonschema.Schema, error) {
	rulesetSchemaCache.Lock()
	defer rulesetSchemaCache.Unlock()
	if rulesetSchemaCache.source == source && (rulesetSchemaCache.schema != nil || rulesetSchemaCache.err != nil) {
		return rulesetSchemaCache.schema, rulesetSchemaCache.err
	}
	var document map[string]any
	err := json.Unmarshal([]byte(source), &document)
	var schema *jsonschema.Schema
	if err == nil {
		compiler := jsonschema.NewCompiler()
		err = compiler.AddResource("schema.json", document)
		if err == nil {
			schema, err = compiler.Compile("schema.json")
		}
	}
	if err != nil {
		err = fmt.Errorf("compile ruleset schema: %w", err)
	}
	rulesetSchemaCache.source = source
	rulesetSchemaCache.schema = schema
	rulesetSchemaCache.err = err
	return schema, err
}
