package motor

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/testify/require"
	"testing"
	"time"
)

func TestExecutionTimeoutReachesEveryDocumentFormat(t *testing.T) {
	for _, tc := range []struct{ name, spec, format string }{
		{"OpenAPI", timeoutTestSpec, ""},
		{"AsyncAPI", validAsyncAPI31Fixture, ""},
		{"JSONSchema", "type: string\n", model.JSONSchemaDraft2020},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := &timeoutRecordingFunction{}
			execution := timeoutRecordingExecution(recorder, 9*time.Second)
			execution.Spec = []byte(tc.spec)
			if tc.name == "AsyncAPI" {
				execution.RuleSet.Rules["record-timeout"].Formats = []string{model.AsyncAPI3}
			}
			execution.SpecFormat = tc.format
			execution.SkipDocumentCheck = tc.format != ""
			result := ApplyRulesToRuleSet(execution)
			defer result.Release()
			require.Empty(t, result.Errors)
			require.Equal(t, []time.Duration{9 * time.Second}, recorder.timeouts())
		})
	}
}
