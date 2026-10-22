package motor

import (
	"runtime"
	"testing"
	"time"
	"weak"

	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/testify/require"
)

func TestSchemaRule_DoesNotRetainDocumentAfterRelease(t *testing.T) {
	for range 3 {
		root := lintTemporarySchemaDocument(t)
		require.Eventually(t, func() bool {
			runtime.GC()
			return root.Value() == nil
		}, 5*time.Second, 10*time.Millisecond, "schema compilation cache retained a released document")
	}
}

func lintTemporarySchemaDocument(t *testing.T) weak.Pointer[yaml.Node] {
	t.Helper()
	doc, err := libopenapi.NewDocument([]byte(`openapi: 3.0.3
info: {title: Cache lifetime, version: "1"}
paths: {}
components:
  schemas:
    A: {type: string}
    B: {type: object}
`))
	require.NoError(t, err)
	defer doc.Release()
	root := weak.Make(doc.GetSpecInfo().RootNode)
	supplied, err := rulesets.CreateRuleSetFromData([]byte(`rules:
  objects:
    given: $.components.schemas[*]
    severity: error
    then:
      function: schema
      functionOptions:
        forceValidationOnCurrentNode: true
        schema: {type: object, required: [type], description: document-cache-lifetime-958}
`))
	require.NoError(t, err)
	result := ApplyRulesToRuleSet(&RuleSetExecution{
		RuleSet:  rulesets.BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(supplied),
		Document: doc, SpecInfo: doc.GetSpecInfo(), SpecFileName: "cache-lifetime.yaml",
	})
	defer result.Release()
	require.Empty(t, result.Errors)
	require.Empty(t, result.Results)
	return root
}
