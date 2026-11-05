package cmd

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
)

func TestDashboardRepairPreviewAndSourceCheckedApply(t *testing.T) {
	source := []byte(`{"openapi":"3.1.0","info":{"title":"Test","version":"1"},"paths":{},"components":{"schemas":{"Pet":{"type":"string","enum":["a","a"]}}}}`)
	path := filepath.Join(t.TempDir(), "api.json")
	require.NoError(t, os.WriteFile(path, source, 0600))
	execution := &motor.RuleSetExecution{Spec: source, SpecFileName: path, SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"duplicated-entry-in-enum": rulesets.GetAllBuiltInRules()["duplicated-entry-in-enum"]}}}
	config := dashboardRepairConfig(path, execution, nil, nil, "")
	review, err := config.Open(source)
	require.NoError(t, err)
	defer review.Close()
	var choices []motor.FixChoice
	for _, option := range review.Options() {
		if option.Selected {
			choices = append(choices, motor.FixChoice{Index: option.Index})
		}
	}
	preview, err := review.Preview(choices)
	require.NoError(t, err)
	defer preview.Close()
	require.Equal(t, 1, preview.Fixed)
	actual, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, source, actual)
	changed := append(append([]byte(nil), source...), '\n')
	require.NoError(t, os.WriteFile(path, changed, 0600))
	require.ErrorContains(t, config.Write(source, preview.Candidate), "changed")
	_, err = config.Open(source)
	require.ErrorContains(t, err, "changed")
	actual, err = os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, changed, actual)
	require.NoError(t, os.WriteFile(path, source, 0600))
	require.NoError(t, config.Write(source, preview.Candidate))
	actual, err = os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, preview.Candidate, actual)
}

func TestDashboardRepairReadOnlyInputs(t *testing.T) {
	for _, tc := range []struct{ source, reason string }{{"saved.json", ""}, {"https://example.com/api.yaml", ""}, {"api.yaml", "Open the full dashboard"}} {
		config := dashboardRepairConfig(tc.source, nil, nil, nil, tc.reason)
		_, err := config.Open(nil)
		require.Error(t, err)
		require.Error(t, config.Write(nil, []byte("changed")))
	}
}

func TestDashboardRepairReloadUsesCurrentRulesAndIgnores(t *testing.T) {
	source := []byte(`{"openapi":"3.1.0","info":{"title":"Test","version":"1"},"paths":{},"components":{"schemas":{"Pet":{"type":"string","enum":["a","a"]}}}}`)
	path := filepath.Join(t.TempDir(), "api.json")
	require.NoError(t, os.WriteFile(path, source, 0600))
	rules := &rulesets.RuleSet{Rules: map[string]*model.Rule{"duplicated-entry-in-enum": rulesets.GetAllBuiltInRules()["duplicated-entry-in-enum"]}}
	config := dashboardRepairConfig(path, &motor.RuleSetExecution{RuleSet: rules, Spec: source, SpecFileName: path, SilenceLogs: true}, nil, nil, "")
	review, err := config.Open(source)
	require.NoError(t, err)
	require.Len(t, review.Options(), 1)
	review.Close()
	config.Reload(&rulesets.RuleSet{Rules: map[string]*model.Rule{}}, nil)
	review, err = config.Open(source)
	require.NoError(t, err)
	require.Empty(t, review.Options())
	review.Close()
	config.Reload(rules, model.IgnoredItems{"duplicated-entry-in-enum": []string{"$.components.schemas.Pet.enum"}})
	review, err = config.Open(source)
	require.NoError(t, err)
	require.Empty(t, review.Options())
	review.Close()
}

func TestDashboardRepairOpenDuringConfigurationReload(t *testing.T) {
	source := []byte(`{"openapi":"3.1.0","info":{"title":"Test","version":"1"},"paths":{}}`)
	path := filepath.Join(t.TempDir(), "api.json")
	require.NoError(t, os.WriteFile(path, source, 0600))
	rules := &rulesets.RuleSet{Rules: map[string]*model.Rule{"info-description": rulesets.GetAllBuiltInRules()["info-description"]}}
	config := dashboardRepairConfig(path, &motor.RuleSetExecution{RuleSet: rules, Spec: source, SpecFileName: path, SilenceLogs: true}, nil, nil, "")
	var workers sync.WaitGroup
	start := make(chan struct{})
	workers.Add(1)
	go func() {
		defer workers.Done()
		<-start
		for i := 0; i < 500; i++ {
			config.Reload(rules, model.IgnoredItems{"info-description": []string{"$.info"}})
			config.Reload(rules, nil)
		}
	}()
	close(start)
	for i := 0; i < 20; i++ {
		review, err := config.Open(source)
		require.NoError(t, err)
		require.LessOrEqual(t, len(review.Options()), 1)
		review.Close()
	}
	workers.Wait()
	config.Reload(rules, model.IgnoredItems{"info-description": []string{"$.info"}})
	review, err := config.Open(source)
	require.NoError(t, err)
	defer review.Close()
	require.Empty(t, review.Options())
}
