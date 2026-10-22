package parser

import (
	"runtime"
	"testing"
	"time"
	"weak"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi-validator/cache"
	"github.com/pb33f/libopenapi-validator/config"
	"github.com/pb33f/libopenapi-validator/schema_validation"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/testify/require"
)

type countingSchemaCache struct {
	*cache.DefaultCache
	stores int
}

func (c *countingSchemaCache) Store(key uint64, entry *cache.SchemaCacheEntry) {
	c.stores++
	c.DefaultCache.Store(key, entry)
}

func TestValidateNodeAgainstSchema_ReusesOwnedCache(t *testing.T) {
	compiled := &countingSchemaCache{DefaultCache: cache.NewDefaultCache()}
	validator := schema_validation.NewSchemaValidator(config.WithSchemaCache(compiled))
	defer validator.Release()
	ctx := &model.RuleFunctionContext{SchemaValidator: validator}
	schema, err := ConvertYAMLIntoJSONSchema("type: string\nminLength: 3", nil)
	require.NoError(t, err)
	for _, tc := range []struct {
		value string
		valid bool
	}{{"valid", true}, {"no", false}, {"again", true}} {
		node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: tc.value}
		valid, errs := ValidateNodeAgainstSchema(ctx, schema, node, false)
		require.Equal(t, tc.valid, valid)
		require.Equal(t, !tc.valid, len(errs) > 0)
	}
	require.Equal(t, 1, compiled.stores, "one compilation must serve all payloads")
	validator.Release()
	compiled.Range(func(_ uint64, _ *cache.SchemaCacheEntry) bool {
		t.Error("released validator still retains a schema")
		return true
	})
}

func TestValidateNodeAgainstSchema_DoesNotRetainSchema(t *testing.T) {
	ref := validateTemporarySchema(t)
	require.Eventually(t, func() bool {
		runtime.GC()
		return ref.Value() == nil
	}, 5*time.Second, 10*time.Millisecond, "a completed validation must not retain its schema")
}

// Keep the source pointer off the caller's stack during garbage collection.
func validateTemporarySchema(t *testing.T) weak.Pointer[base.Schema] {
	t.Helper()
	schema, err := ConvertYAMLIntoJSONSchema("type: string\ndescription: temporary-schema-958", nil)
	require.NoError(t, err)
	valid, errs := ValidateNodeAgainstSchema(nil, schema, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "ok"}, false)
	require.True(t, valid)
	require.Empty(t, errs)
	return weak.Make(schema)
}
