package owasp

import (
	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/testify/require"
	"sync"
	"testing"
)

func TestLimitRulesShareLazyDirections(t *testing.T) {
	for _, bounded := range []bool{false, true} {
		spec := "openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths:\n  /test:\n    post:\n      requestBody:\n        content:\n          application/json:\n            schema:\n              type: array\n              items: {type: string"
		if bounded {
			spec += ", maxLength: 5"
		}
		spec += "}\n"
		if bounded {
			spec += "              maxItems: 5\n"
		}
		doc, err := libopenapi.NewDocument([]byte(spec))
		require.NoError(t, err)
		m, errs := doc.BuildV3Model()
		require.Empty(t, errs)
		dr := drModel.NewDrDocument(m)
		t.Cleanup(dr.Release)
		var cache sync.Map
		ctx := model.RuleFunctionContext{DrDocument: dr, SchemaPathCache: &cache, Rule: &model.Rule{Id: "limits"}}
		var wg sync.WaitGroup
		for _, function := range []model.RuleFunction{ArrayLimit{}, StringLimit{}} {
			wg.Go(func() {
				results := function.RunRule(nil, ctx)
				if bounded {
					require.Empty(t, results)
				} else {
					require.Len(t, results, 1)
				}
			})
		}
		wg.Wait()
		_, cached := cache.Load(schemaDirectionsKey{&m.Model})
		require.Equal(t, !bounded, cached, "bounded schemas must not build a direction index")
	}
}
