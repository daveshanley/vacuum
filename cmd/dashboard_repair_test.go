// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/daveshanley/vacuum/motor/autofix"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/tui"
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
	var choices []autofix.FixChoice
	for _, option := range review.Options() {
		require.False(t, option.Selected, "a fix must start unselected")
		if option.Available {
			choices = append(choices, autofix.FixChoice{Index: option.Index})
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
		require.Error(t, config.Save(nil, []byte("changed"), tui.RepairSaveOptions{}))
		_, err = config.PreviewOutput(nil, nil, tui.RepairSaveOptions{})
		require.Error(t, err)
	}
}

func TestDashboardRepairSaveReviewedOutput(t *testing.T) {
	for _, tc := range []struct {
		name, output string
		jsonOutput   bool
	}{
		{name: "update"},
		{name: "json copy", output: "fixed.JSON", jsonOutput: true},
		{name: "yaml copy", output: "fixed.yml"},
		{name: "preserve custom extension", output: "fixed.openapi"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config, review, choices, source := dashboardRepairSaveFixture(t)
			save := tui.RepairSaveOptions{OutputPath: tc.output, BundleMode: "keep"}
			preview, err := config.PreviewOutput(review, choices, save)
			require.NoError(t, err)
			defer preview.Close()
			require.Equal(t, 1, preview.Fixed)
			require.Equal(t, tc.jsonOutput, json.Valid(preview.Candidate))
			target := config.SourcePath
			if tc.output != "" {
				target = filepath.Join(filepath.Dir(config.SourcePath), tc.output)
				_, err = os.Lstat(target)
				require.True(t, os.IsNotExist(err), "preview must not write a file")
			}
			actual, err := os.ReadFile(config.SourcePath)
			require.NoError(t, err)
			require.Equal(t, source, actual)
			require.NoError(t, config.Save(source, preview.Candidate, save))
			actual, err = os.ReadFile(target)
			require.NoError(t, err)
			require.Equal(t, preview.Candidate, actual)
			if tc.output != "" {
				actual, err = os.ReadFile(config.SourcePath)
				require.NoError(t, err)
				require.Equal(t, source, actual, "copy must preserve the source")
			}
		})
	}
}

func TestDashboardRepairCopyChecksSourceAndLateDestination(t *testing.T) {
	for _, stale := range []bool{false, true} {
		t.Run(map[bool]string{false: "late destination", true: "changed source"}[stale], func(t *testing.T) {
			config, review, choices, source := dashboardRepairSaveFixture(t)
			save := tui.RepairSaveOptions{OutputPath: "fixed.yaml"}
			preview, err := config.PreviewOutput(review, choices, save)
			require.NoError(t, err)
			defer preview.Close()
			target := filepath.Join(filepath.Dir(config.SourcePath), save.OutputPath)
			external := []byte("external edit")
			if stale {
				require.NoError(t, os.WriteFile(config.SourcePath, external, 0600))
			} else {
				require.NoError(t, os.WriteFile(target, external, 0600))
			}
			require.Error(t, config.Save(source, preview.Candidate, save))
			actual, err := os.ReadFile(target)
			if stale {
				require.True(t, os.IsNotExist(err))
			} else {
				require.NoError(t, err)
				require.Equal(t, external, actual)
			}
		})
	}
}

func TestDashboardRepairOutputPathValidation(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "api.yaml")
	require.NoError(t, os.WriteFile(source, []byte("source"), 0600))
	for _, name := range []string{"existing.yaml", "linked.yaml", "symlink.yaml"} {
		path := filepath.Join(dir, name)
		switch name {
		case "linked.yaml":
			require.NoError(t, os.Link(source, path))
		case "symlink.yaml":
			require.NoError(t, os.Symlink(source, path))
		default:
			require.NoError(t, os.WriteFile(path, []byte("keep"), 0600))
		}
		_, err := dashboardRepairOutputPath(source, name)
		require.Error(t, err)
	}
	for _, name := range []string{"api.yaml", source, "  ", "missing/fixed.yaml"} {
		_, err := dashboardRepairOutputPath(source, name)
		require.Error(t, err)
	}
	output, err := dashboardRepairOutputPath(source, "./fixed.yaml")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "fixed.yaml"), output)
	output, err = dashboardRepairOutputPath(source, "")
	require.NoError(t, err)
	require.Empty(t, output)
}

func dashboardRepairSaveFixture(t *testing.T) (*tui.RepairConfig, *autofix.FixReview, []autofix.FixChoice, []byte) {
	t.Helper()
	source := []byte("openapi: 3.1.0\ninfo:\n  title: Test\n  version: '1'\npaths: {}\ncomponents:\n  schemas:\n    Pet:\n      type: string\n      enum: [a, a]\n")
	path := filepath.Join(t.TempDir(), "api.yaml")
	require.NoError(t, os.WriteFile(path, source, 0600))
	execution := &motor.RuleSetExecution{Spec: source, SpecFileName: path, SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"duplicated-entry-in-enum": rulesets.GetAllBuiltInRules()["duplicated-entry-in-enum"]}}}
	config := dashboardRepairConfig(path, execution, nil, nil, "")
	require.Equal(t, path, config.SourcePath)
	review, err := config.Open(source)
	require.NoError(t, err)
	t.Cleanup(review.Close)
	var choices []autofix.FixChoice
	for _, option := range review.Options() {
		if option.Available {
			choices = append(choices, autofix.FixChoice{Index: option.Index})
		}
	}
	require.Len(t, choices, 1)
	return config, review, choices, source
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
