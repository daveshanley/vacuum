// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package rulesets

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/testify/require"
)

func TestArazzoRulesetAliasesAndOverrides(t *testing.T) {
	for _, name := range []string{VacuumArazzo, VacuumArazzoRecommended, SpectralArazzo} {
		for _, mode := range []string{VacuumRecommended, VacuumAll, VacuumOff} {
			rs, err := CreateRuleSetFromData([]byte("extends: [[" + name + ", " + mode + "]]\nrules:\n  arazzo-reference: warn\n  arazzo-parameter: off\n"))
			require.NoError(t, err)
			loaded := BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(rs)
			require.NoError(t, loaded.LoadError())
			if mode == VacuumOff {
				require.Empty(t, loaded.Rules)
				continue
			}
			require.Equal(t, model.SeverityWarn, loaded.Rules["arazzo-reference"].Severity)
			require.NotContains(t, loaded.Rules, "arazzo-parameter")
			require.Contains(t, loaded.Rules, ArazzoValidationIncomplete)
			_, advisory := loaded.Rules["arazzo-advisory"]
			require.Equal(t, mode == VacuumAll, advisory)
		}
	}
}

func TestArazzoNestedRuleset(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "child.yaml"), []byte("extends: [spectral:arazzo]\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "parent.yaml"), []byte("extends: [./child.yaml]\nrules:\n  arazzo-reference: warn\n"), 0600))
	rs, err := LoadLocalRuleSet(context.Background(), filepath.Join(dir, "parent.yaml"))
	require.NoError(t, err)
	loaded := BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(rs)
	require.NoError(t, loaded.LoadError())
	require.Contains(t, loaded.Rules, "arazzo-structure")
	require.Equal(t, model.SeverityWarn, loaded.Rules["arazzo-reference"].Severity)
}

func TestArazzoRulesCanBeEnabledIndividually(t *testing.T) {
	for _, name := range []string{VacuumArazzo, VacuumArazzoRecommended, SpectralArazzo} {
		rs, err := CreateRuleSetFromData([]byte("extends: [[" + name + ", off]]\nrules:\n  arazzo-reference: true\n"))
		require.NoError(t, err)
		loaded := BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(rs)
		require.NoError(t, loaded.LoadError())
		require.Len(t, loaded.Rules, 1)
		require.Contains(t, loaded.Rules, "arazzo-reference")
	}
	rs, err := CreateRuleSetFromData([]byte("extends: [vacuum:arazzo]\nrules:\n  arazzo-advisory: true\n  arazzo-reference: false\n"))
	require.NoError(t, err)
	loaded := BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(rs)
	require.NoError(t, loaded.LoadError())
	require.Contains(t, loaded.Rules, "arazzo-advisory")
	require.NotContains(t, loaded.Rules, "arazzo-reference")
}
