package motor

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
)

func evidenceExecution(given string, formats []string) *RuleSetExecution {
	return &RuleSetExecution{Spec: []byte("openapi: 3.0.3\ninfo:\n  title: Test\n  version: 1.0.0\npaths: {}\n"), SilenceLogs: true, Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{
		"probe": {Id: "probe", Given: given, Formats: formats, Severity: "error", Then: model.RuleAction{Function: "defined", Field: "title"}},
	}}}
}

func TestExecutionEvidenceDistinguishesEmptyResults(t *testing.T) {
	for _, tc := range []struct {
		name, given, status string
		formats             []string
		matches, calls      int
	}{
		{"pass", "$.info", "completed", nil, 1, 1},
		{"no match", "$.absent", "completed", nil, 0, 0},
		{"excluded", "$.info", "excluded", []string{"oas2"}, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			evidence := &ExecutionEvidence{SampleLimit: 1}
			result := ApplyRulesToRuleSetWithOptions(evidenceExecution(tc.given, tc.formats), &ExecutionOptions{Evidence: evidence})
			defer result.ReleaseOwnedResources()
			require.Empty(t, result.Errors)
			require.Empty(t, result.Results)
			require.Len(t, evidence.Rules, 1)
			require.Equal(t, tc.status, evidence.Rules[0].Status)
			require.Equal(t, tc.matches, evidence.Rules[0].Matches)
			require.Equal(t, tc.calls, evidence.Rules[0].Invocations)
		})
	}
}

func TestExecutionEvidenceSamplesBounded(t *testing.T) {
	e := &ExecutionEvidence{SampleLimit: 1}
	r := ApplyRulesToRuleSetWithOptions(evidenceExecution("$.info.*", nil), &ExecutionOptions{Evidence: e})
	defer r.ReleaseOwnedResources()
	require.Empty(t, r.Errors)
	require.Equal(t, 2, e.Rules[0].Matches)
	require.Len(t, e.Rules[0].Selectors[0].Samples, 1)
	require.True(t, e.Rules[0].Selectors[0].Truncated)
}

func TestExecutionEvidenceTimeoutDoesNotPublishWorkerState(t *testing.T) {
	ex := evidenceExecution("$", nil)
	ex.RuleSet.Rules["probe"].Then = model.RuleAction{Function: "slow"}
	ex.CustomFunctions = map[string]model.RuleFunction{"slow": &slowRule{sleep: 100 * time.Millisecond}}
	ex.Timeout = time.Millisecond
	e := &ExecutionEvidence{}
	r := ApplyRulesToRuleSetWithOptions(ex, &ExecutionOptions{Evidence: e})
	defer r.ReleaseOwnedResources()
	require.NotEmpty(t, r.Errors)
	require.Equal(t, "timed_out", e.Rules[0].Status)
	require.Zero(t, e.Rules[0].Invocations)
	time.Sleep(120 * time.Millisecond)
	require.Zero(t, e.Rules[0].Invocations)
}

func TestExecutionEvidencePreCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	e := &ExecutionEvidence{}
	r := ApplyRulesToRuleSetWithOptions(evidenceExecution("$", nil), &ExecutionOptions{Evidence: e, Context: ctx})
	require.NotEmpty(t, r.Errors)
	require.Empty(t, e.Rules)
}

func BenchmarkExecutionEvidence(b *testing.B) {
	for _, enabled := range []bool{false, true} {
		name := "disabled"
		if enabled {
			name = "summary"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				opts := &ExecutionOptions{}
				if enabled {
					opts.Evidence = &ExecutionEvidence{}
				}
				r := ApplyRulesToRuleSetWithOptions(evidenceExecution("$.info", nil), opts)
				r.ReleaseOwnedResources()
			}
		})
	}
}
