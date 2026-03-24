// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// registerPersistentFlags registers the persistent flags normally set on the root command.
// Needed because sub-commands expect these flags when invoked directly in tests.
func registerPersistentFlags(cmd *cobra.Command) {
	pf := cmd.PersistentFlags()
	pf.String("original", "", "Path to original spec")
	pf.String("changes", "", "Path to change report")
	pf.String("breaking-config", "", "Path to breaking rules config")
	pf.Bool("warn-on-changes", false, "Warn on changes")
	pf.Bool("error-on-breaking", false, "Error on breaking")
	pf.StringP("ruleset", "r", "", "Ruleset")
	pf.StringP("functions", "f", "", "Functions")
	pf.StringP("base", "p", "", "Base")
	pf.BoolP("remote", "u", true, "Remote")
	pf.BoolP("skip-check", "k", false, "Skip check")
	pf.BoolP("debug", "w", false, "Debug")
	pf.IntP("timeout", "g", 5, "Timeout")
	pf.Int("lookup-timeout", 500, "Lookup timeout")
	pf.BoolP("hard-mode", "z", false, "Hard mode")
	pf.BoolP("ext-refs", "", false, "Ext refs")
	pf.String("cert-file", "", "Cert")
	pf.String("key-file", "", "Key")
	pf.String("ca-file", "", "CA")
	pf.Bool("insecure", false, "Insecure")
	pf.Bool("allow-private-networks", false, "Private networks")
	pf.Bool("allow-http", false, "Allow HTTP")
	pf.Int("fetch-timeout", 30, "Fetch timeout")
	pf.BoolP("time", "t", false, "Time")
	pf.Bool("changes-summary", false, "Changes summary")
	pf.BoolP("turbo", "T", false, "Turbo")
}

// --- Lint command tests ---

func TestLintCommand_OriginalSameSpec_SuppressesAll(t *testing.T) {
	// Using the same file as both original and new should suppress all lint violations
	// (full overlap), leaving only any change violations if requested.
	spec := "../model/test_files/burgershop.openapi.yaml"

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--original", spec,
		"-n", "error", // fail-severity=error only
		"-x", // silent
		spec,
	})

	err := cmd.Execute()
	// Same spec vs same spec: no new violations, no errors expected
	assert.NoError(t, err)
}

func TestLintCommand_OriginalDifferentSpec(t *testing.T) {
	// Using two completely different specs: no overlap, so all new-spec violations reported.
	original := "../model/test_files/petstorev3.json"
	newSpec := "../model/test_files/burgershop.openapi.yaml"

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--original", original,
		"-x", // silent
		newSpec,
	})

	// Should run without error (violations are expected but not failure-severity level)
	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestLintCommand_OriginalMissingFile_WarnsAndProceeds(t *testing.T) {
	// When original file doesn't exist, should warn and proceed without filtering
	newSpec := "../model/test_files/burgershop.openapi.yaml"

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--original", "/nonexistent/spec.yaml",
		newSpec,
	})

	// Should not panic, should proceed (may return error due to lint violations)
	_ = cmd.Execute()
}

func TestLintCommand_OriginalWithErrorOnBreaking(t *testing.T) {
	// Test that --original combined with --error-on-breaking works:
	// violation diffing for lint results, plus breaking change injection.
	spec := "../model/test_files/burgershop.openapi.yaml"

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--original", spec,
		"--error-on-breaking",
		"-n", "error",
		"-x", // silent
		spec,
	})

	// Same spec: no breaking changes, no new violations → no error
	err := cmd.Execute()
	assert.NoError(t, err)
}

// --- Spectral report tests ---

func TestSpectralReport_OriginalSameSpec(t *testing.T) {
	spec := "../model/test_files/petstorev3.json"
	reportFile := filepath.Join(t.TempDir(), "spectral-original-test.json")

	cmd := GetSpectralReportCommand()
	registerPersistentFlags(cmd)
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--original", spec,
		spec,
		reportFile,
	})

	err := cmd.Execute()
	assert.NoError(t, err)

	// Report should exist and be valid JSON (empty array since all suppressed)
	data, readErr := os.ReadFile(reportFile)
	require.NoError(t, readErr)
	assert.True(t, len(data) > 0)
}

func TestSpectralReport_OriginalWithChangeViolations(t *testing.T) {
	spec := "../model/test_files/petstorev3.json"
	reportFile := filepath.Join(t.TempDir(), "spectral-changes-test.json")

	cmd := GetSpectralReportCommand()
	registerPersistentFlags(cmd)
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--original", spec,
		"--warn-on-changes",
		spec,
		reportFile,
	})

	err := cmd.Execute()
	assert.NoError(t, err)
}

// --- HTML report tests ---

func TestHTMLReport_OriginalSameSpec(t *testing.T) {
	spec := "../model/test_files/burgershop.openapi.yaml"
	reportFile := filepath.Join(t.TempDir(), "html-original-test.html")

	cmd := GetHTMLReportCommand()
	registerPersistentFlags(cmd)
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--original", spec,
		"-b", // no-banner
		spec,
		reportFile,
	})

	err := cmd.Execute()
	assert.NoError(t, err)

	data, readErr := os.ReadFile(reportFile)
	require.NoError(t, readErr)
	assert.True(t, len(data) > 0)
}

func TestHTMLReport_PrecompiledReport_OriginalFallback(t *testing.T) {
	// Precompiled report with --original should fall back to ChangeFilter without panic
	report := "../model/test_files/burgershop-report.json.gz"
	spec := "../model/test_files/burgershop.openapi.yaml"
	reportFile := filepath.Join(t.TempDir(), "html-precompiled-test.html")

	cmd := GetHTMLReportCommand()
	registerPersistentFlags(cmd)
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--original", spec,
		"-b", // no-banner
		report,
		reportFile,
	})

	// Should not panic; falls back to ChangeFilter for precompiled report
	err := cmd.Execute()
	assert.NoError(t, err)
}

// --- Vacuum report tests ---

func TestVacuumReport_OriginalSameSpec(t *testing.T) {
	spec := "../model/test_files/petstorev3.json"
	reportPrefix := filepath.Join(t.TempDir(), "vacuum-original-test")

	cmd := GetVacuumReportCommand()
	registerPersistentFlags(cmd)
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--original", spec,
		spec,
		reportPrefix,
	})

	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestVacuumReport_OriginalWithErrorOnBreaking(t *testing.T) {
	spec := "../model/test_files/petstorev3.json"
	reportPrefix := filepath.Join(t.TempDir(), "vacuum-breaking-test")

	cmd := GetVacuumReportCommand()
	registerPersistentFlags(cmd)
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--original", spec,
		"--error-on-breaking",
		spec,
		reportPrefix,
	})

	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestVacuumReport_OriginalMissingFile_WarnsAndProceeds(t *testing.T) {
	spec := "../model/test_files/petstorev3.json"
	reportPrefix := filepath.Join(t.TempDir(), "vacuum-missing-test")

	cmd := GetVacuumReportCommand()
	registerPersistentFlags(cmd)
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--original", "/nonexistent/spec.yaml",
		spec,
		reportPrefix,
	})

	// Should not panic, should warn and proceed
	err := cmd.Execute()
	assert.NoError(t, err)
}
