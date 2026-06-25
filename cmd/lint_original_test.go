// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	vacuum_report "github.com/daveshanley/vacuum/vacuum-report"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v4"
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
	pf.Bool("resolve-all-refs", false, "Resolve all refs")
	pf.Bool("nested-refs-doc-context", false, "Nested refs doc context")
}

func runIssue839OriginalDiffRegressionSerial(t *testing.T) {
	t.Helper()

	origProcs := runtime.GOMAXPROCS(1)
	t.Cleanup(func() {
		runtime.GOMAXPROCS(origProcs)
	})
}

type originalShortcutTestFunction struct{}

func (originalShortcutTestFunction) RunRule(_ []*yaml.Node, _ model.RuleFunctionContext) []model.RuleFunctionResult {
	return nil
}

func (originalShortcutTestFunction) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "original-shortcut-test"}
}

func (originalShortcutTestFunction) GetCategory() string {
	return model.CategoryValidation
}

// --- Lint command tests ---

func TestLintOriginalSpec_ReturnsResults(t *testing.T) {
	results, err := LintOriginalSpec("../model/test_files/burgershop.openapi.yaml", &motor.RuleSetExecution{
		RuleSet: rulesets.BuildDefaultRuleSets().GenerateOpenAPIRecommendedRuleSet(),
	}, nil)

	require.NoError(t, err)
	require.NotEmpty(t, results)
}

func TestOriginalSpecCanReuseCurrentResults_InternalRefsOnly(t *testing.T) {
	dir := t.TempDir()
	original := filepath.Join(dir, "original.yaml")
	current := filepath.Join(dir, "current.yaml")
	spec := []byte(`openapi: 3.0.3
info:
  title: Test
  version: 1.0.0
paths: {}
components:
  schemas:
    Thing:
      $ref: '#/components/schemas/Other'
    Other:
      type: string
`)
	require.NoError(t, os.WriteFile(original, spec, 0o600))
	require.NoError(t, os.WriteFile(current, spec, 0o600))

	assert.True(t, originalSpecCanReuseCurrentResults(original, spec, current, "", nil))
}

func TestOriginalSpecCanReuseCurrentResults_CustomFunctionsRequireOriginalLint(t *testing.T) {
	dir := t.TempDir()
	original := filepath.Join(dir, "original.yaml")
	current := filepath.Join(dir, "current.yaml")
	spec := []byte(`openapi: 3.0.3
info:
  title: Test
  version: 1.0.0
paths: {}
`)
	require.NoError(t, os.WriteFile(original, spec, 0o600))
	require.NoError(t, os.WriteFile(current, spec, 0o600))

	customFunctions := map[string]model.RuleFunction{
		"original-shortcut-test": originalShortcutTestFunction{},
	}
	assert.False(t, originalSpecCanReuseCurrentResults(original, spec, current, "", customFunctions))
}

func TestOriginalSpecCanReuseCurrentResults_CustomBaseRequiresOriginalLint(t *testing.T) {
	dir := t.TempDir()
	original := filepath.Join(dir, "original.yaml")
	current := filepath.Join(dir, "current.yaml")
	customBase := filepath.Join(dir, "custom-base")
	spec := []byte(`openapi: 3.0.3
info:
  title: Test
  version: 1.0.0
paths: {}
`)
	require.NoError(t, os.MkdirAll(customBase, 0o755))
	require.NoError(t, os.WriteFile(original, spec, 0o600))
	require.NoError(t, os.WriteFile(current, spec, 0o600))

	assert.False(t, originalSpecCanReuseCurrentResults(original, spec, current, customBase, nil))
}

func TestOriginalSpecCanReuseCurrentResults_ExternalRefsNeedOriginalLintAcrossDifferentFiles(t *testing.T) {
	dir := t.TempDir()
	original := filepath.Join(dir, "original.yaml")
	current := filepath.Join(dir, "current.yaml")
	spec := []byte(`openapi: 3.0.3
info:
  title: Test
  version: 1.0.0
paths: {}
components:
  schemas:
    Thing:
      $ref: './common.yaml#/Thing'
`)
	require.NoError(t, os.WriteFile(original, spec, 0o600))
	require.NoError(t, os.WriteFile(current, spec, 0o600))

	assert.False(t, originalSpecCanReuseCurrentResults(original, spec, current, "", nil))
	assert.True(t, originalSpecCanReuseCurrentResults(original, spec, original, "", nil))
}

func TestOriginalSpecCanReuseCurrentResults_MirroredExternalRefs(t *testing.T) {
	dir := t.TempDir()
	originalDir := filepath.Join(dir, "original")
	currentDir := filepath.Join(dir, "current")
	require.NoError(t, os.MkdirAll(originalDir, 0o755))
	require.NoError(t, os.MkdirAll(currentDir, 0o755))

	spec := []byte(`openapi: 3.0.3
info:
  title: Test
  version: 1.0.0
paths: {}
components:
  schemas:
    Thing:
      $ref: './common.yaml#/Thing'
`)
	common := []byte(`Thing:
  type: string
`)
	original := filepath.Join(originalDir, "openapi.yaml")
	current := filepath.Join(currentDir, "openapi.yaml")
	require.NoError(t, os.WriteFile(original, spec, 0o600))
	require.NoError(t, os.WriteFile(current, spec, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(originalDir, "common.yaml"), common, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(currentDir, "common.yaml"), common, 0o600))

	assert.True(t, originalSpecCanReuseCurrentResults(original, spec, current, "", nil))

	require.NoError(t, os.WriteFile(filepath.Join(currentDir, "common.yaml"), []byte("Thing:\n  type: integer\n"), 0o600))
	assert.False(t, originalSpecCanReuseCurrentResults(original, spec, current, "", nil))
}

func TestApplyOriginalDiffToValues_ReuseCurrentResultsUsesCanonicalStats(t *testing.T) {
	results := []model.RuleFunctionResult{
		{RuleId: "z-rule", Message: "same", Path: "$.z"},
		{RuleId: "a-rule", Message: "same", Path: "$.a"},
	}

	filtered, stats := applyOriginalDiffToValues(originalValueDiffOptions{
		OriginalPath:        "original.yaml",
		CurrentPath:         "current.yaml",
		Results:             results,
		ReuseCurrentResults: true,
	})

	assert.Empty(t, filtered)
	require.NotNil(t, stats)
	assert.Equal(t, len(results), stats.TotalResultsBefore)
	assert.Equal(t, 0, stats.TotalResultsAfter)
	assert.Equal(t, len(results), stats.ResultsDropped)
	assert.Equal(t, []string{"a-rule", "z-rule"}, stats.RulesFullyFiltered)
	assert.Empty(t, stats.RulesPartialFiltered)
}

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
		"--pipeline-output",
		spec,
	})

	err := cmd.Execute()
	// Same spec vs same spec: no new violations, no errors expected
	assert.NoError(t, err)
}

func TestLintCommand_OriginalSameSpec_RendersComparisonSummary(t *testing.T) {
	spec := "../model/test_files/burgershop.openapi.yaml"

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--original", spec,
		"-n", "error",
		"--no-style",
		spec,
	})

	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestLintCommand_OriginalSuppressesLineShiftedViolations(t *testing.T) {
	original, newSpec, ruleset := writeOriginalLineShiftFixture(t)

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--original", original,
		"-r", ruleset,
		"-n", "warn",
		"--pipeline-output",
		newSpec,
	})

	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestLintCommand_OriginalSameSpec_SuppressesAll_Issue839Regression(t *testing.T) {
	runIssue839OriginalDiffRegressionSerial(t)

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
			"--pipeline-output",
			spec,
		})

		err := cmd.Execute()
		assert.NoErrorf(t, err, "iteration %d should suppress all same-spec violations", i)
	}
}

func TestLintCommand_OriginalSameSpec_SuppressesAll_Issue839CustomerSuppliedExternalRefs(t *testing.T) {
	runIssue839OriginalDiffRegressionSerial(t)

	spec := "../model/test_files/api-main.yaml"
	ruleset := "../model/test_files/issue_839_ruleset.yaml"

	const iterations = 5
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
			"--pipeline-output",
			spec,
		})

		err := cmd.Execute()
		assert.NoErrorf(t, err, "iteration %d should suppress all same-spec violations for issue 839 customer fixtures", i)
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
		"--pipeline-output",
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

func writeOriginalLineShiftFixture(t *testing.T) (originalPath string, newPath string, rulesetPath string) {
	t.Helper()

	dir := t.TempDir()
	originalPath = filepath.Join(dir, "openapi-original.yaml")
	newPath = filepath.Join(dir, "openapi-new.yaml")
	rulesetPath = filepath.Join(dir, "ruleset.yaml")

	ruleset := "extends: [[vacuum:oas, off]]\nrules:\n  operation-description: true\n"
	require.NoError(t, os.WriteFile(rulesetPath, []byte(ruleset), 0o600))

	operation := `paths:
  /pets:
    get:
      operationId: listPets
      responses:
        '200':
          description: ok
`

	original := `openapi: 3.0.3
info:
  title: Line Shift Fixture
  version: 1.0.0
` + operation

	newSpec := `openapi: 3.0.3
info:
  title: Line Shift Fixture
  version: 1.0.0
  description: |
    Added documentation that should not make existing lint findings new.
    This only moves the line numbers below.
x-generated-docs:
  enabled: true
` + operation

	require.NoError(t, os.WriteFile(originalPath, []byte(original), 0o600))
	require.NoError(t, os.WriteFile(newPath, []byte(newSpec), 0o600))

	return originalPath, newPath, rulesetPath
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
		"--pipeline-output",
		spec,
	})

	// Same spec: no breaking changes, no new violations → no error
	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestLintCommand_ResolveAllRefsFlag(t *testing.T) {
	spec := `openapi: "3.0.2"
info:
  title: Test
  version: "1.0"
paths:
  /test:
    get:
      responses:
        '404':
          $ref: '#/components/responses/NotFound'
components:
  responses:
    NotFound:
      description: Not Found
      content:
        application/json:
          schema:
            type: object
`
	ruleset := `extends: [[vacuum:oas, off]]
rules:
  response-has-content:
    description: Ensure referenced responses expose content
    severity: error
    recommended: true
    formats: [oas3]
    resolved: false
    given: "$.paths[*][*].responses['404']"
    then:
      field: content
      function: defined
`

	tempDir := t.TempDir()
	specPath := filepath.Join(tempDir, "spec.yaml")
	rulesetPath := filepath.Join(tempDir, "ruleset.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte(spec), 0o600))
	require.NoError(t, os.WriteFile(rulesetPath, []byte(ruleset), 0o600))

	cmd := GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBufferString(""))
	cmd.SetErr(bytes.NewBufferString(""))
	cmd.SetArgs([]string{
		"-r", rulesetPath,
		"-b",
		"-q",
		specPath,
	})

	err := cmd.Execute()
	assert.Error(t, err)

	cmd = GetLintCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBufferString(""))
	cmd.SetErr(bytes.NewBufferString(""))
	cmd.SetArgs([]string{
		"-r", rulesetPath,
		"--resolve-all-refs",
		"-b",
		"-q",
		specPath,
	})

	err = cmd.Execute()
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

	var spectralResults []map[string]any
	require.NoError(t, json.Unmarshal(data, &spectralResults))
	assert.Empty(t, spectralResults)
}

func TestSpectralReport_OriginalSameSpec_Issue839CustomerSuppliedExternalRefs(t *testing.T) {
	runIssue839OriginalDiffRegressionSerial(t)

	spec := "../model/test_files/api-main.yaml"
	ruleset := "../model/test_files/issue_839_ruleset.yaml"
	reportFile := filepath.Join(t.TempDir(), "spectral-issue-839.json")

	cmd := GetSpectralReportCommand()
	registerPersistentFlags(cmd)
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--original", spec,
		"-r", ruleset,
		"--no-style",
		"--no-pretty",
		spec,
		reportFile,
	})

	err := cmd.Execute()
	require.NoError(t, err)

	data, readErr := os.ReadFile(reportFile)
	require.NoError(t, readErr)

	var spectralResults []map[string]any
	require.NoError(t, json.Unmarshal(data, &spectralResults))
	assert.Empty(t, spectralResults)
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

// --- Vacuum report tests ---

func copyIssue839FixturesToMirroredDirs(t *testing.T) (string, string) {
	t.Helper()

	root := t.TempDir()
	folder1 := filepath.Join(root, "folder1")
	folder2 := filepath.Join(root, "folder2")
	require.NoError(t, os.MkdirAll(folder1, 0o755))
	require.NoError(t, os.MkdirAll(folder2, 0o755))

	for _, fileName := range []string{"api-main.yaml", "api-common.yaml"} {
		sourcePath := filepath.Join("..", "model", "test_files", fileName)
		data, err := os.ReadFile(sourcePath)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(folder1, fileName), data, 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(folder2, fileName), data, 0o644))
	}

	return filepath.Join(folder1, "api-main.yaml"), filepath.Join(folder2, "api-main.yaml")
}

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

	reportFiles, globErr := filepath.Glob(reportPrefix + "-*.json")
	require.NoError(t, globErr)
	require.Len(t, reportFiles, 1)

	data, readErr := os.ReadFile(reportFiles[0])
	require.NoError(t, readErr)

	var report vacuum_report.VacuumReport
	require.NoError(t, json.Unmarshal(data, &report))
	require.NotNil(t, report.ResultSet)
	assert.Empty(t, report.ResultSet.Results)
	assert.Equal(t, 0, report.ResultSet.ErrorCount)
	assert.Equal(t, 0, report.ResultSet.WarnCount)
	assert.Equal(t, 0, report.ResultSet.InfoCount)
	assert.Equal(t, 0, report.ResultSet.HintCount)
}

func TestVacuumReport_OriginalSameSpec_Issue839CustomerSuppliedExternalRefs(t *testing.T) {
	runIssue839OriginalDiffRegressionSerial(t)

	spec := "../model/test_files/api-main.yaml"
	ruleset := "../model/test_files/issue_839_ruleset.yaml"
	reportPrefix := filepath.Join(t.TempDir(), "vacuum-issue-839")

	cmd := GetVacuumReportCommand()
	registerPersistentFlags(cmd)
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--original", spec,
		"-r", ruleset,
		"--no-style",
		"--no-pretty",
		spec,
		reportPrefix,
	})

	err := cmd.Execute()
	require.NoError(t, err)

	reportFiles, globErr := filepath.Glob(reportPrefix + "-*.json")
	require.NoError(t, globErr)
	require.Len(t, reportFiles, 1)

	data, readErr := os.ReadFile(reportFiles[0])
	require.NoError(t, readErr)

	var report vacuum_report.VacuumReport
	require.NoError(t, json.Unmarshal(data, &report))
	require.NotNil(t, report.ResultSet)
	require.NotNil(t, report.Statistics)

	assert.Empty(t, report.ResultSet.Results)
	assert.Equal(t, 0, report.ResultSet.ErrorCount)
	assert.Equal(t, 0, report.ResultSet.WarnCount)
	assert.Equal(t, 0, report.ResultSet.InfoCount)
	assert.Equal(t, 0, report.ResultSet.HintCount)
	assert.Equal(t, 0, report.Statistics.TotalErrors)
	assert.Equal(t, 0, report.Statistics.TotalWarnings)
	assert.Equal(t, 0, report.Statistics.TotalInfo)
	assert.Equal(t, 0, report.Statistics.TotalHints)
}

func TestVacuumReport_OriginalMirroredExternalRefsSuppressesAll(t *testing.T) {
	runIssue839OriginalDiffRegressionSerial(t)

	originalSpec, currentSpec := copyIssue839FixturesToMirroredDirs(t)
	ruleset := "../model/test_files/issue_839_ruleset.yaml"
	reportPrefix := filepath.Join(t.TempDir(), "vacuum-issue-880")

	cmd := GetVacuumReportCommand()
	registerPersistentFlags(cmd)
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{
		"--original", originalSpec,
		"-r", ruleset,
		"--no-style",
		"--no-pretty",
		currentSpec,
		reportPrefix,
	})

	err := cmd.Execute()
	require.NoError(t, err)

	reportFiles, globErr := filepath.Glob(reportPrefix + "-*.json")
	require.NoError(t, globErr)
	require.Len(t, reportFiles, 1)

	data, readErr := os.ReadFile(reportFiles[0])
	require.NoError(t, readErr)

	var report vacuum_report.VacuumReport
	require.NoError(t, json.Unmarshal(data, &report))
	require.NotNil(t, report.ResultSet)

	assert.Empty(t, report.ResultSet.Results)
	assert.Equal(t, 0, report.ResultSet.ErrorCount)
	assert.Equal(t, 0, report.ResultSet.WarnCount)
	assert.Equal(t, 0, report.ResultSet.InfoCount)
	assert.Equal(t, 0, report.ResultSet.HintCount)
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
