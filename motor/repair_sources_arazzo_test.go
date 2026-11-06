package motor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/repair"
	"github.com/daveshanley/vacuum/rulesets"
	arazzosource "github.com/pb33f/libopenapi/arazzo"
	"github.com/pb33f/testify/require"
)

const frozenArazzoAPI = `{"openapi":"3.1.0","info":{"title":"API","version":"1"},"paths":{"/pets":{"get":{"operationId":"read","responses":{"200":{"description":"OK"}}}}}}`

func frozenArazzoExecution(t *testing.T, name, url string) *RuleSetExecution {
	t.Helper()
	source := fmt.Sprintf(`{"arazzo":"1.1.0","info":{"title":"Flow","version":"1"},"sourceDescriptions":[{"name":"api","url":%q,"type":"openapi"}],"workflows":[{"workflowId":"run","steps":[{"stepId":"read","operationId":"$sourceDescriptions.api.read"}]}]}`, url)
	return &RuleSetExecution{Spec: []byte(source), SpecFileName: name, AllowLookup: true, SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"arazzo-reference": rulesets.GetAllArazzoRules()["arazzo-reference"]}}}
}

func checkFrozenArazzoPreview(t *testing.T, e *RuleSetExecution, before *RuleSetExecutionResult, hashes map[string]string) {
	t.Helper()
	plan := repair.NewPlan(e.SpecFileName, e.Spec)
	plan.Sources = hashes
	plan.Operations = []repair.Operation{{Op: "add", Path: "/info/description", Value: json.RawMessage(`"Reviewed workflow"`)}}
	checked, err := validatePreparedPlan(e, plan, nil, before)
	require.NoError(t, err)
	defer checked.Final.ReleaseOwnedResources()
	require.NotEmpty(t, checked.Candidate)
	require.Equal(t, before.FilesProcessed, checked.Final.FilesProcessed)
	require.Equal(t, before.FileSize, checked.Final.FileSize)
}

func TestArazzoRepairUsesFrozenFile(t *testing.T) {
	for _, remove := range []bool{false, true} {
		t.Run(fmt.Sprint(remove), func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, "api.json")
			require.NoError(t, os.WriteFile(file, []byte(frozenArazzoAPI), 0600))
			e := frozenArazzoExecution(t, filepath.Join(dir, "flow.json"), "api.json")
			_, before, err := PlanFixes(e, nil)
			require.NoError(t, err)
			defer before.ReleaseOwnedResources()
			require.Empty(t, before.Results)
			_, hashes, err := freezeRepairSources(before)
			require.NoError(t, err)
			require.NotEmpty(t, hashes)
			if remove {
				require.NoError(t, os.Remove(file))
			} else {
				require.NoError(t, os.WriteFile(file, []byte("invalid changed input"), 0600))
			}
			checkFrozenArazzoPreview(t, e, before, hashes)
			outside := repair.NewPlan(e.SpecFileName, e.Spec)
			outside.Sources = hashes
			outside.Operations = []repair.Operation{{Op: "replace", Path: "/sourceDescriptions/0/url", Value: json.RawMessage(`"missing.json"`)}}
			checked, err := validatePreparedPlan(e, outside, nil, before)
			require.Nil(t, checked)
			require.ErrorContains(t, err, "outside the captured collection")
			for key := range hashes {
				hashes[key] = "changed"
				break
			}
			outside.Sources = hashes
			_, err = validatePreparedPlan(e, outside, nil, before)
			require.ErrorContains(t, err, "dependency snapshots changed")
		})
	}
}

func TestArazzoRepairPreservesRedirectSnapshot(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path == "/alias" {
			http.Redirect(w, r, "/real/api.json", http.StatusFound)
			return
		}
		fmt.Fprint(w, frozenArazzoAPI)
	}))
	defer server.Close()
	e := frozenArazzoExecution(t, filepath.Join(t.TempDir(), "flow.json"), server.URL+"/alias")
	_, before, err := PlanFixes(e, nil)
	require.NoError(t, err)
	defer before.ReleaseOwnedResources()
	snapshots, hashes, err := freezeRepairSources(before)
	require.NoError(t, err)
	require.Equal(t, server.URL+"/real/api.json", snapshots.retrievals[server.URL+"/alias"])
	require.Contains(t, hashes, server.URL+"/alias")
	require.Contains(t, hashes, server.URL+"/real/api.json")
	count := requests.Load()
	server.Close()
	checkFrozenArazzoPreview(t, e, before, hashes)
	require.Equal(t, count, requests.Load())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = (&repairSourceResolver{snapshot: snapshots}).Resolve(ctx, arazzosource.SourceRequest{URL: server.URL + "/alias"})
	require.ErrorIs(t, err, context.Canceled)
}

func TestArazzoRepairCapturesSourcesWithNarrowRules(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "api.json"), []byte(frozenArazzoAPI), 0600))
	e := frozenArazzoExecution(t, filepath.Join(dir, "flow.json"), "api.json")
	e.RuleSet = &rulesets.RuleSet{Rules: map[string]*model.Rule{"arazzo-step-operationPath": rulesets.GetAllArazzoRules()["arazzo-step-operationPath"]}}
	ordinary := ApplyRulesToRuleSet(e)
	require.Empty(t, ordinary.Errors)
	require.Nil(t, ordinary.repairSources)
	require.Nil(t, ordinary.Arazzo.Validation)
	ordinary.ReleaseOwnedResources()
	_, baseline, err := PlanFixes(e, nil)
	require.NoError(t, err)
	defer baseline.ReleaseOwnedResources()
	require.NotNil(t, baseline.Arazzo.Validation)
	require.NotNil(t, baseline.repairSources)
	_, hashes, err := freezeRepairSources(baseline)
	require.NoError(t, err)
	require.NotEmpty(t, hashes)
	require.NoError(t, os.Remove(filepath.Join(dir, "api.json")))
	checkFrozenArazzoPreview(t, e, baseline, hashes)
}
