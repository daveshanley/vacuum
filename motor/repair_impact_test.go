package motor

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
)

const impactSpec = `{"openapi":"3.1.0","info":{"title":"Impact","version":"1","description":"Original"},"paths":{"/pets":{"get":{"responses":{"200":{"description":"OK"}}}}}}`

func impactResult(t *testing.T, source string) *RuleSetExecutionResult {
	t.Helper()
	e := &RuleSetExecution{Spec: []byte(source), SpecFileName: "impact.json", SilenceLogs: true,
		RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"info-description": rulesets.GetAllBuiltInRules()["info-description"]}}}
	r := ApplyRulesToRuleSet(e)
	t.Cleanup(r.ReleaseOwnedResources)
	return r
}

func TestRepairImpactActualDocuments(t *testing.T) {
	for _, tc := range []struct {
		name, proposed string
		status         RepairImpactStatus
	}{
		{"unchanged", impactSpec, RepairImpactNoBreaking},
		{"documentation", strings.Replace(impactSpec, "Original", "Updated", 1), RepairImpactNoBreaking},
		{"removed-path", `{"openapi":"3.1.0","info":{"title":"Impact","version":"1","description":"Original"},"paths":{}}`, RepairImpactBreaking},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before, after := impactResult(t, impactSpec), impactResult(t, tc.proposed)
			require.Empty(t, before.Errors)
			require.Empty(t, after.Errors)
			impact := assessRepairImpact(before, after, nil)
			require.Equal(t, tc.status, impact.Status, impact.Reason)
			if tc.name != "unchanged" {
				require.NotEmpty(t, impact.Changes)
			}
			for _, c := range impact.Changes {
				require.Nil(t, c.OriginalObject)
				require.Nil(t, c.NewObject)
			}
			a, err := json.Marshal(impact)
			require.NoError(t, err)
			b, err := json.Marshal(assessRepairImpact(before, after, nil))
			require.NoError(t, err)
			require.Equal(t, string(a), string(b))
		})
	}
}

func TestRepairImpactUnknown(t *testing.T) {
	before, after := impactResult(t, impactSpec), impactResult(t, impactSpec)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.Equal(t, RepairImpactUnknown, assessRepairImpact(before, after, &ExecutionOptions{Context: ctx}).Status)
	after.RuleSetExecution.SkipResolve = true
	require.Equal(t, RepairImpactUnknown, assessRepairImpact(before, after, nil).Status)
	after.RuleSetExecution.SkipResolve = false
	after.RuleSetExecution.SpecFormat = "json-schema"
	// An unsupported execution must not be reported as an unchanged OpenAPI document.
	require.Equal(t, RepairImpactUnknown, assessRepairImpact(before, after, nil).Status)
	require.Equal(t, RepairImpactUnknown, assessRepairImpact(nil, after, nil).Status)
}

func TestRepairImpactUnresolvedReferenceIsUnknown(t *testing.T) {
	broken := strings.Replace(impactSpec, `"paths":`, `"components":{"schemas":{"Broken":{"$ref":"#/components/schemas/Absent"}}},"paths":`, 1)
	before, after := impactResult(t, broken), impactResult(t, impactSpec)
	impact := assessRepairImpact(before, after, nil)
	require.Equal(t, RepairImpactUnknown, impact.Status, impact.Reason)
	require.Empty(t, impact.Changes)
}

func TestRepairImpactSwagger(t *testing.T) {
	before := impactResult(t, `{"swagger":"2.0","info":{"title":"Impact","version":"1"},"paths":{"/pets":{"get":{"responses":{"200":{"description":"OK"}}}}}}`)
	after := impactResult(t, `{"swagger":"2.0","info":{"title":"Impact","version":"1"},"paths":{}}`)
	impact := assessRepairImpact(before, after, nil)
	require.Equal(t, RepairImpactBreaking, impact.Status, impact.Reason)
	require.Positive(t, impact.BreakingChanges)
}

func TestRepairImpactUsesAlreadyBuiltExternalModels(t *testing.T) {
	dir := t.TempDir()
	dependency := filepath.Join(dir, "pet.yaml")
	require.NoError(t, os.WriteFile(dependency, []byte("type: string\n"), 0600))
	source := strings.Replace(impactSpec, `"paths":`, `"components":{"schemas":{"Pet":{"$ref":"pet.yaml"}}},"paths":`, 1)
	build := func(spec string) *RuleSetExecutionResult {
		e := &RuleSetExecution{Spec: []byte(spec), SpecFileName: filepath.Join(dir, "api.json"), Base: dir, AllowLookup: true, SilenceLogs: true,
			RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"info-description": rulesets.GetAllBuiltInRules()["info-description"]}}}
		r := ApplyRulesToRuleSet(e)
		t.Cleanup(r.ReleaseOwnedResources)
		require.Empty(t, r.Errors)
		return r
	}
	before, after := build(source), build(strings.Replace(source, "Original", "Updated", 1))
	require.NoError(t, os.Remove(dependency))
	impact := assessRepairImpact(before, after, nil)
	require.Equal(t, RepairImpactNoBreaking, impact.Status, impact.Reason)
	require.NotEmpty(t, impact.Changes)
}
