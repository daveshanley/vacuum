package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/daveshanley/vacuum/harness"
	"github.com/pb33f/testify/require"
)

func TestHarnessSchemaAndInputErrors(t *testing.T) {
	for _, args := range [][]string{{"--schema", "suite"}, {"--schema", "result"}, {}, {"--format", "xml"}} {
		command := GetTestCommand()
		var out bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&out)
		command.SetArgs(args)
		err := command.Execute()
		if len(args) > 0 && args[0] == "--schema" {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
		require.True(t, json.Valid(out.Bytes()), out.String())
	}
}
func TestHarnessCLIIntegration(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "vacuum")
	build := exec.Command("go", "build", "-o", binary, "..")
	build.Env = append(os.Environ(), "GOWORK=off")
	output, err := build.CombinedOutput()
	require.NoError(t, err, string(output))
	write := func(name, body string) {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(body), 0600))
	}
	write("rules.yaml", "rules:\n  required-title:\n    given: $.info\n    severity: error\n    then: {function: defined, field: title}\n")
	write("api.yaml", "openapi: 3.0.3\ninfo: {title: Test, version: 1.0.0}\npaths: {}\n")
	write("suite.yaml", "version: 1\nruleset: rules.yaml\ncases:\n- id: pass\n  input: api.yaml\n  expect:\n    diagnostics: []\n    execution:\n      required-title:\n        status: completed\n        matches: {min: 1}\n")
	// Ambient config must not override the suite or prevent it from running.
	write("vacuum.conf.yaml", "not: [valid")
	run := func(args ...string) (harness.Report, string) {
		p := exec.Command(binary, append([]string{"test"}, args...)...)
		p.Dir = dir
		var stdout, stderr bytes.Buffer
		p.Stdout = &stdout
		p.Stderr = &stderr
		err := p.Run()
		var report harness.Report
		require.NoError(t, json.Unmarshal(stdout.Bytes(), &report), stdout.String()+stderr.String())
		if report.ExitCode == 0 {
			require.NoError(t, err, stderr.String())
		} else {
			var e *exec.ExitError
			require.ErrorAs(t, err, &e)
			require.Equal(t, report.ExitCode, e.ExitCode())
		}
		return report, stderr.String()
	}
	r, _ := run("suite.yaml", "--bundle", "replay")
	require.Equal(t, 0, r.ExitCode)
	require.True(t, r.Manifest.Reproducible)
	r, _ = run("replay/suite.yaml")
	require.Equal(t, 0, r.ExitCode)
	write("api.yaml", "openapi: 3.0.3\ninfo: {version: 1.0.0}\npaths: {}\n")
	r, _ = run("suite.yaml")
	require.Equal(t, 1, r.ExitCode)
	r, _ = run("suite.yaml", "--case", "missing")
	require.Equal(t, 2, r.ExitCode)
	require.Equal(t, harness.ErrorSelection, r.Errors[0].Code)
	require.Equal(t, harness.PhaseLoad, r.Errors[0].Phase)
	require.NoError(t, os.Mkdir(filepath.Join(dir, "functions"), 0700))
	write("functions/noisy.js", `function getSchema(){return {name:"noisy"};} function runRule(input){console.log("plugin noise");return [];}`)
	write("rules.yaml", "rules:\n  noisy:\n    given: $\n    severity: error\n    then: {function: noisy}\n")
	write("suite.yaml", "version: 1\nruleset: rules.yaml\nfunctions: functions\ncases:\n- id: noisy\n  input: api.yaml\n  expect: {diagnostics: []}\n")
	r, logs := run("suite.yaml")
	require.Equal(t, 0, r.ExitCode)
	require.Contains(t, logs, "plugin noise")
}
