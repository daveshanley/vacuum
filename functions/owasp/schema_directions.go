package owasp

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/go-yaml"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"sync"
)

type schemaDirectionsKey struct{ document *v3.Document }

// Cache schema identity and request usage once per document in this execution.
// A typed key keeps this index separate from cached schema paths.
func schemaNodeDirections(ctx model.RuleFunctionContext) map[*yaml.Node]utils.DirectionType {
	doc := ctx.DrDocument.V3Document.Document
	build := func() map[*yaml.Node]utils.DirectionType { return utils.GetSchemaNodeDirections(doc) }
	if ctx.SchemaPathCache == nil {
		return build()
	}
	key := schemaDirectionsKey{doc}
	if cached, ok := ctx.SchemaPathCache.Load(key); ok {
		return cached.(func() map[*yaml.Node]utils.DirectionType)()
	}
	cached, _ := ctx.SchemaPathCache.LoadOrStore(key, sync.OnceValue(build))
	return cached.(func() map[*yaml.Node]utils.DirectionType)()
}
