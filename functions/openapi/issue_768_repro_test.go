package openapi

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	drModel "github.com/pb33f/doctor/model"
	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIssue768_NonDeterministicPaths reproduces the flaky camel-case-properties
// behavior reported in https://github.com/daveshanley/vacuum/issues/768.
//
// The hedwig_scope property in ServiceLevelObjectiveAlertSeverityV1 is referenced
// via $ref from two locations (ticket and page in ServiceLevelObjectiveAlertV1).
// Due to concurrent schema walking, the JSONPath for the result can non-deterministically
// resolve to either the definition-site path or a usage-site path.
//
// On GitHub runners (single-core), this race is more likely to trigger.
func TestIssue768_NonDeterministicPaths(t *testing.T) {
	specBytes, err := os.ReadFile("../../model/test_files/issue_768_test.yaml")
	require.NoError(t, err, "issue_768_test.yaml must exist in model/test_files")

	// Force single-core execution to maximize race condition likelihood
	origProcs := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(origProcs)

	const iterations = 50
	var failCount int
	var paths []string

	for i := 0; i < iterations; i++ {
		document, err := libopenapi.NewDocument(specBytes)
		require.NoError(t, err)

		m, errs := document.BuildV3Model()
		require.Empty(t, errs)

		drDocument := drModel.NewDrDocumentWithConfig(m, &drModel.DrConfig{
			UseSchemaCache:     true,
			DeterministicPaths: true,
		})

		rule := model.Rule{
			Name: "camel-case-properties",
			Id:   "camel-case-properties",
		}
		ctx := model.RuleFunctionContext{
			Rule:       &rule,
			DrDocument: drDocument,
			Document:   document,
		}

		ccp := CamelCaseProperties{}
		results := ccp.RunRule(nil, ctx)

		for _, r := range results {
			if strings.Contains(r.Message, "hedwig_scope") {
				paths = append(paths, r.Path)
				// Check if the path is the definition-site path (what the ignore file expects)
				if !strings.HasPrefix(r.Path, "$.components.schemas['ServiceLevelObjectiveAlertSeverityV1']") {
					failCount++
					t.Logf("Iteration %d: non-deterministic path: %s", i, r.Path)
				}
			}
		}
	}

	// Collect unique paths to show the non-determinism
	uniquePaths := make(map[string]int)
	for _, p := range paths {
		uniquePaths[p]++
	}

	t.Logf("Total hedwig_scope results across %d iterations: %d", iterations, len(paths))
	t.Logf("Unique paths seen:")
	for p, count := range uniquePaths {
		t.Logf("  [%d times] %s", count, p)
	}

	if failCount > 0 {
		t.Logf("Non-deterministic paths detected in %d results", failCount)
	}

	// ASSERT: all paths should use the definition-site (canonical) path
	assert.Equal(t, 0, failCount,
		"All hedwig_scope results should use the definition-site path "+
			"($.components.schemas['ServiceLevelObjectiveAlertSeverityV1']...), "+
			"but some used usage-site paths. This is the race condition from issue #768.")
}
