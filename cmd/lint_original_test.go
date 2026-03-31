// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func TestLintCommand_OriginalSameSpec_SuppressesAll_Issue839Regression(t *testing.T) {
	origProcs := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(origProcs)

	spec, ruleset := writeIssue839RegressionFixture(t)

	const iterations = 10
	for i := 0; i < iterations; i++ {
		cmd := GetLintCommand()
		registerPersistentFlags(cmd)
		b := bytes.NewBufferString("")
		cmd.SetOut(b)
		cmd.SetErr(b)
		cmd.SetArgs([]string{
			"--original", spec,
			"-r", ruleset,
			"-n", "error",
			"-x",
			spec,
		})

		err := cmd.Execute()
		assert.NoErrorf(t, err, "iteration %d should suppress all same-spec violations", i)
	}
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

func writeIssue839RegressionFixture(t *testing.T) (specPath string, rulesetPath string) {
	t.Helper()

	dir := t.TempDir()
	specPath = filepath.Join(dir, "issue_839.yaml")
	rulesetPath = filepath.Join(dir, "issue_839_ruleset.yaml")

	err := os.WriteFile(rulesetPath, []byte("extends:\n  - [vacuum:oas, all]\n  - [vacuum:owasp, all]\n"), 0o600)
	require.NoError(t, err)

	var builder strings.Builder
	builder.WriteString("openapi: 3.0.3\n")
	builder.WriteString("info:\n")
	builder.WriteString("  title: issue 839 regression fixture\n")
	builder.WriteString("  version: 1.0.0\n")
	builder.WriteString("paths:\n")

	for i := 1; i <= 40; i++ {
		builder.WriteString(fmt.Sprintf("  /p%d:\n", i))
		builder.WriteString("    get:\n")
		builder.WriteString(fmt.Sprintf("      operationId: op%d\n", i))
		builder.WriteString("      parameters:\n")
		builder.WriteString(fmt.Sprintf("        - name: queryA%d\n", i))
		builder.WriteString("          in: query\n")
		builder.WriteString("          schema:\n")
		builder.WriteString("            type: string\n")
		builder.WriteString(fmt.Sprintf("        - name: queryB%d\n", i))
		builder.WriteString("          in: query\n")
		builder.WriteString("          schema:\n")
		builder.WriteString("            type: string\n")
		builder.WriteString(fmt.Sprintf("        - name: ids%d\n", i))
		builder.WriteString("          in: query\n")
		builder.WriteString("          schema:\n")
		builder.WriteString("            type: array\n")
		builder.WriteString("            items:\n")
		builder.WriteString("              type: string\n")
		builder.WriteString("      responses:\n")
		builder.WriteString("        '200':\n")
		builder.WriteString("          description: ok\n")
	}

	builder.WriteString("components:\n")
	builder.WriteString("  schemas:\n")
	builder.WriteString("    StableThing:\n")
	builder.WriteString("      type: object\n")
	builder.WriteString("      properties:\n")
	builder.WriteString("        id:\n")
	builder.WriteString("          type: string\n")
	builder.WriteString("          maxLength: 32\n")

	err = os.WriteFile(specPath, []byte(builder.String()), 0o600)
	require.NoError(t, err)

	return specPath, rulesetPath
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
