package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/model/reports"
	vacuum_report "github.com/daveshanley/vacuum/vacuum-report"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"github.com/spf13/cobra"
)

const circularArrayOpenAPISpec = `openapi: 3.1.0
info:
  title: Circular Array API
  version: 1.0.0
paths:
  /one:
    get:
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/arr'
components:
  schemas:
    arr:
      type: array
      items:
        $ref: '#/components/schemas/obj'
    obj:
      type: object
      properties:
        self:
          $ref: '#/components/schemas/obj'
`

func TestReportCommandsExposeCircularReferenceFlags(t *testing.T) {
	for _, tc := range []struct {
		name string
		cmd  func() *cobra.Command
	}{
		{name: "html-report", cmd: GetHTMLReportCommand},
		{name: "dashboard", cmd: GetDashboardCommand},
		{name: "report", cmd: GetVacuumReportCommand},
		{name: "spectral-report", cmd: GetSpectralReportCommand},
	} {
		t.Run(tc.name, func(t *testing.T) {
			flags := tc.cmd().Flags()
			assert.NotNil(t, flags.Lookup("ignore-array-circle-ref"))
			assert.NotNil(t, flags.Lookup("ignore-polymorph-circle-ref"))
		})
	}
}

func TestVacuumReportCommand_IgnoreArrayCircularReferenceFlag(t *testing.T) {
	specPath := writeCircularArrayOpenAPISpec(t)
	withoutFlagReport := runVacuumReportForCircularSpec(t, specPath, false)
	withFlagReport := runVacuumReportForCircularSpec(t, specPath, true)

	require.NotNil(t, withoutFlagReport.ResultSet)
	require.NotNil(t, withFlagReport.ResultSet)
	assertRuleResultPresent(t, withoutFlagReport.ResultSet.Results, "circular-references")
	assertRuleResultAbsent(t, withFlagReport.ResultSet.Results, "circular-references")
}

func TestSpectralReportCommand_IgnoreArrayCircularReferenceFlag(t *testing.T) {
	specPath := writeCircularArrayOpenAPISpec(t)
	withoutFlagReport := runSpectralReportForCircularSpec(t, specPath, false)
	withFlagReport := runSpectralReportForCircularSpec(t, specPath, true)

	assertSpectralResultPresent(t, withoutFlagReport, "circular-references")
	assertSpectralResultAbsent(t, withFlagReport, "circular-references")
}

func TestDashboardCommand_IgnoreArrayCircularReferenceFlag(t *testing.T) {
	specPath := writeCircularArrayOpenAPISpec(t)
	cmd := GetDashboardCommand()
	registerPersistentFlags(cmd)
	cmd.Flags().Bool("silent", false, "Show nothing except the result")

	output := bytes.NewBuffer(nil)
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetArgs([]string{
		"--silent",
		"--ruleset", "../rulesets/examples/norules-ruleset.yaml",
		"--ignore-array-circle-ref",
		"--ignore-polymorph-circle-ref",
		specPath,
	})

	assert.NoError(t, cmd.Execute())
}

func writeCircularArrayOpenAPISpec(t *testing.T) string {
	t.Helper()

	specPath := filepath.Join(t.TempDir(), "circular-array.yaml")
	writeTestFile(t, specPath, circularArrayOpenAPISpec)
	return specPath
}

func runVacuumReportForCircularSpec(t *testing.T, specPath string, ignoreArrayCircularRef bool) vacuum_report.VacuumReport {
	t.Helper()

	cmd := GetVacuumReportCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))

	reportPrefix := filepath.Join(t.TempDir(), "vacuum-report")
	args := []string{
		"--no-style",
		"--ruleset", "../rulesets/examples/norules-ruleset.yaml",
		specPath,
		reportPrefix,
	}
	if ignoreArrayCircularRef {
		args = append([]string{"--ignore-array-circle-ref"}, args...)
	}
	cmd.SetArgs(args)

	require.NoError(t, cmd.Execute())

	reportPath := requireSingleGeneratedFile(t, reportPrefix+"-*.json")
	reportBytes, err := os.ReadFile(reportPath)
	require.NoError(t, err)

	var report vacuum_report.VacuumReport
	require.NoError(t, json.Unmarshal(reportBytes, &report))
	return report
}

func runSpectralReportForCircularSpec(t *testing.T, specPath string, ignoreArrayCircularRef bool) []reports.SpectralReport {
	t.Helper()

	cmd := GetSpectralReportCommand()
	registerPersistentFlags(cmd)
	cmd.SetOut(bytes.NewBuffer(nil))

	reportPath := filepath.Join(t.TempDir(), "spectral-report.json")
	args := []string{
		"--no-style",
		"--ruleset", "../rulesets/examples/norules-ruleset.yaml",
		specPath,
		reportPath,
	}
	if ignoreArrayCircularRef {
		args = append([]string{"--ignore-array-circle-ref"}, args...)
	}
	cmd.SetArgs(args)

	require.NoError(t, cmd.Execute())

	reportBytes, err := os.ReadFile(reportPath)
	require.NoError(t, err)

	var report []reports.SpectralReport
	require.NoError(t, json.Unmarshal(reportBytes, &report))
	return report
}

func assertRuleResultPresent(t *testing.T, results []*model.RuleFunctionResult, ruleID string) {
	t.Helper()

	for _, result := range results {
		if result.RuleId == ruleID {
			return
		}
	}
	t.Fatalf("expected rule result %q to be present", ruleID)
}

func assertRuleResultAbsent(t *testing.T, results []*model.RuleFunctionResult, ruleID string) {
	t.Helper()

	for _, result := range results {
		if result.RuleId == ruleID {
			t.Fatalf("expected rule result %q to be absent", ruleID)
		}
	}
}

func assertSpectralResultPresent(t *testing.T, results []reports.SpectralReport, ruleID string) {
	t.Helper()

	for _, result := range results {
		if result.Code == ruleID {
			return
		}
	}
	t.Fatalf("expected spectral result %q to be present", ruleID)
}

func assertSpectralResultAbsent(t *testing.T, results []reports.SpectralReport, ruleID string) {
	t.Helper()

	for _, result := range results {
		if result.Code == ruleID {
			t.Fatalf("expected spectral result %q to be absent", ruleID)
		}
	}
}
