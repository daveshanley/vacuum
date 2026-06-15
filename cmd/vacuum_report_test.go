package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	vacuum_report "github.com/daveshanley/vacuum/vacuum-report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetVacuumReportCommand(t *testing.T) {
	cmd := GetVacuumReportCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"../model/test_files/petstorev3.json",
	})
	cmdErr := cmd.Execute()
	outBytes, err := io.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)

	time := time.Now()
	file := fmt.Sprintf("vacuum-report-%s.json", time.Format("01-02-06-15_04_05"))
	defer os.Remove(file)
}

func TestGetVacuumReportCommand_StdInOut(t *testing.T) {
	cmd := GetVacuumReportCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{"-i", "-o"})
	cmd.SetIn(strings.NewReader("openapi: 3.1.0"))
	cmdErr := cmd.Execute()
	outBytes, err := io.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
}

func TestGetVacuumReportCommand_NoPretty(t *testing.T) {
	cmd := GetVacuumReportCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{"-i", "-o", "-n"})
	cmd.SetIn(strings.NewReader("openapi: 3.1.0"))
	cmdErr := cmd.Execute()
	outBytes, err := io.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
}

func TestGetVacuumReportCommand_Compress(t *testing.T) {
	cmd := GetVacuumReportCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-c",
		"../model/test_files/petstorev3.json",
	})
	cmdErr := cmd.Execute()
	outBytes, err := io.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)

	time := time.Now()
	file := fmt.Sprintf("vacuum-report-%s.json.gz", time.Format("01-02-06-15_04_05"))
	defer os.Remove(file)
}

func TestGetVacuumReportCommand_CustomPrefix(t *testing.T) {
	cmd := GetVacuumReportCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"../model/test_files/petstorev3.json",
		"cheesy-shoes",
	})
	cmdErr := cmd.Execute()
	outBytes, err := io.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)

	time := time.Now()
	file := fmt.Sprintf("cheesy-shoes-%s.json", time.Format("01-02-06-15_04_05"))
	defer os.Remove(file)
}

func TestGetVacuumReportCommand_WithRuleSet(t *testing.T) {
	cmd := GetVacuumReportCommand()
	// global flag exists on root only.
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")

	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-r",
		"../rulesets/examples/norules-ruleset.yaml",
		"../model/test_files/petstorev3.json",
	})
	cmdErr := cmd.Execute()
	outBytes, err := io.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)

	time := time.Now()
	file := fmt.Sprintf("vacuum-report-%s.json", time.Format("01-02-06-15_04_05"))
	defer os.Remove(file)
}

func TestGetVacuumReportCommand_WithBadRuleset(t *testing.T) {
	cmd := GetVacuumReportCommand()
	// global flag exists on root only.
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")

	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-r",
		"I do not exist",
		"../model/test_files/petstorev3.json",
	})
	cmdErr := cmd.Execute()
	assert.Error(t, cmdErr)
}

func TestGetVacuumReportCommand_WithBadRuleset_WrongFile(t *testing.T) {
	cmd := GetVacuumReportCommand()
	// global flag exists on root only.
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")

	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-r",
		"../model/test_files/petstorev3.json",
		"../model/test_files/petstorev3.json",
	})
	cmdErr := cmd.Execute()
	assert.Error(t, cmdErr)
}

func TestGetVacuumReportCommand_BadWrite(t *testing.T) {
	cmd := GetVacuumReportCommand()
	// global flag exists on root only.
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")

	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"../model/test_files/petstorev3.json",
		"/cant-write-here/oh-noes.json",
	})
	cmdErr := cmd.Execute()
	assert.Error(t, cmdErr)
}

func TestGetVacuumReportCommand_NoArgs(t *testing.T) {
	cmd := GetVacuumReportCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{})
	cmdErr := cmd.Execute()
	assert.Error(t, cmdErr)

}

func TestGetVacuumReportCommand_BadFile(t *testing.T) {
	cmd := GetVacuumReportCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"I do not exist",
	})
	cmdErr := cmd.Execute()
	assert.Error(t, cmdErr)
}

func TestGetVacuumReportCommand_UnparseableSpec(t *testing.T) {
	cmd := GetVacuumReportCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"../rulesets/examples/all-ruleset.yaml",
	})
	cmdErr := cmd.Execute()
	assert.Error(t, cmdErr)
	var exitErr *ExitError
	assert.ErrorAs(t, cmdErr, &exitErr)
	assert.Equal(t, ExitCodeInputError, exitErr.Code)
}

func TestGetVacuumReport_WithIgnoreFile(t *testing.T) {

	yaml := `
extends: [[vacuum:oas, recommended]]
rules:
    url-starts-with-major-version:
        description: Major version must be the first URL component
        message: All paths must start with a version number, eg /v1, /v2
        given: $.paths
        severity: error
        then:
            function: pattern
            functionOptions:
                match: "/v[0-9]+/"
`

	tmp, _ := os.CreateTemp("", "")
	_, _ = io.WriteString(tmp, yaml)

	defer os.Remove(tmp.Name())

	cmd := GetVacuumReportCommand()
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")
	cmd.SetArgs([]string{
		"--ignore-file",
		"../model/test_files/burgershop.ignorefile.yaml",
		"-r",
		tmp.Name(),
		"../model/test_files/burgershop.openapi.yaml",
		"--stdout",
	})
	cmdErr := cmd.Execute()
	assert.NoError(t, cmdErr)
}

func TestGetVacuumReportCommand_IncludesExecutionErrors(t *testing.T) {
	rulesetYAML := `
extends: [[vacuum:oas, recommended]]
rules:
  invalid-jsonpath-selector:
    description: Invalid selector
    message: Invalid selector
    given: "$..["
    severity: error
    then:
      function: truthy
`
	specYAML := `
openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths: {}
`

	tmpDir := t.TempDir()
	rulesetPath := filepath.Join(tmpDir, "ruleset.yaml")
	specPath := filepath.Join(tmpDir, "openapi.yaml")
	require.NoError(t, os.WriteFile(rulesetPath, []byte(rulesetYAML), 0o644))
	require.NoError(t, os.WriteFile(specPath, []byte(specYAML), 0o644))

	reportPrefix := filepath.Join(tmpDir, "vacuum-report-errors")

	cmd := GetVacuumReportCommand()
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")
	cmd.SetArgs([]string{
		"-r",
		rulesetPath,
		"--no-style",
		"--no-pretty",
		specPath,
		reportPrefix,
	})

	cmdErr := cmd.Execute()
	require.NoError(t, cmdErr)

	reportFiles, globErr := filepath.Glob(reportPrefix + "-*.json")
	require.NoError(t, globErr)
	require.Len(t, reportFiles, 1)

	data, readErr := os.ReadFile(reportFiles[0])
	require.NoError(t, readErr)

	var report vacuum_report.VacuumReport
	require.NoError(t, json.Unmarshal(data, &report))
	require.NotNil(t, report.Errors)
	require.NotEmpty(t, report.Errors.Items)
	assert.NotEmpty(t, report.Errors.Items[0].Message)
	assert.Equal(t, vacuum_report.ReportErrorTypeRuleLookup, report.Errors.Items[0].Type)
	assert.Equal(t, "invalid-jsonpath-selector", report.Errors.Items[0].RuleId)
	assert.Equal(t, "$..[", report.Errors.Items[0].Given)
}

func TestGetVacuumReportCommand_Issue907RecursiveFilterPathlessJSResultUsesConcretePath(t *testing.T) {
	tmpDir := t.TempDir()
	functionsDir := filepath.Join(tmpDir, "functions")
	require.NoError(t, os.Mkdir(functionsDir, 0o700))

	specPath := filepath.Join(tmpDir, "openapi.yaml")
	rulesetPath := filepath.Join(tmpDir, "ruleset.yaml")
	functionPath := filepath.Join(functionsDir, "issue907.js")

	require.NoError(t, os.WriteFile(specPath, []byte(`openapi: 3.0.3
info:
  title: issue 907 repro
  version: 1.0.0
paths:
  /pets:
    get:
      parameters:
        - name: X-Trace
          in: header
          schema:
            type: string
      responses:
        "200":
          description: ok
`), 0o600))
	require.NoError(t, os.WriteFile(rulesetPath, []byte(`extends: [[vacuum:oas, off]]
rules:
  issue-907-filter-path:
    description: Header names should be checked at the matched node
    severity: error
    recommended: true
    formats: [oas3]
    given: $..[?(@ && @.in == 'header')].name
    then:
      function: issue907
`), 0o600))
	require.NoError(t, os.WriteFile(functionPath, []byte(`function getSchema() {
  return {
    name: "issue907",
    description: "returns a pathless result for the matched node"
  };
}

function runRule(input) {
  if (!input) {
    return [];
  }
  return [{ message: "header name issue" }];
}
`), 0o600))

	cmd := GetVacuumReportCommand()
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")
	cmd.PersistentFlags().StringP("functions", "f", "", "")
	reportPrefix := filepath.Join(tmpDir, "issue-907-report")
	cmd.SetArgs([]string{
		"--no-style",
		"--no-pretty",
		"-r", rulesetPath,
		"-f", functionsDir,
		specPath,
		reportPrefix,
	})

	cmdErr := cmd.Execute()
	require.NoError(t, cmdErr)

	reportFiles, globErr := filepath.Glob(reportPrefix + "-*.json")
	require.NoError(t, globErr)
	require.Len(t, reportFiles, 1)
	reportBytes, readErr := os.ReadFile(reportFiles[0])
	require.NoError(t, readErr)

	var report vacuum_report.VacuumReport
	require.NoError(t, json.Unmarshal(reportBytes, &report))
	require.NotNil(t, report.ResultSet)
	require.Len(t, report.ResultSet.Results, 1)
	assert.Equal(t, "$.paths['/pets'].get.parameters[0].name", report.ResultSet.Results[0].Path)
	assert.NotContains(t, report.ResultSet.Results[0].Path, "$..")
	assert.NotContains(t, report.ResultSet.Results[0].Path, "[?")
}
