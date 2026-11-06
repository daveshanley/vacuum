package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/motor"
	"github.com/pb33f/testify/require"
)

func TestConsoleRepairEnterAppliesAndEscapeCancels(t *testing.T) {
	for _, tc := range []struct {
		status   motor.RepairImpactStatus
		answer   string
		accepted bool
	}{
		{motor.RepairImpactNoBreaking, "\r", true},
		{motor.RepairImpactNoBreaking, "\x1b", false},
		{motor.RepairImpactBreaking, "\r", true},
		{motor.RepairImpactBreaking, "\x1b", false},
		{motor.RepairImpactUnknown, "\n", true},
		{motor.RepairImpactUnknown, "\x1b", false},
	} {
		t.Run(string(tc.status)+tc.answer, func(t *testing.T) {
			var out bytes.Buffer
			p := &motor.FixPreview{Candidate: []byte("name: new\n"), Fixed: 1, Impact: motor.RepairImpact{Status: tc.status, Reason: "unsupported format"}}
			accepted, err := reviewCLIRepair(strings.NewReader(tc.answer), &out, true, false, "api.yaml", []byte("name: old\n"), p)
			require.NoError(t, err)
			require.Equal(t, tc.accepted, accepted)
			require.Contains(t, out.String(), "--- original")
			require.Contains(t, out.String(), "name: new")
			require.Less(t, strings.Index(out.String(), "name: new"), strings.Index(out.String(), "enter: apply"))
		})
	}
}

func TestConsoleRepairCannotAcceptWithoutReview(t *testing.T) {
	preview := &motor.FixPreview{Candidate: []byte("{}"), Impact: motor.RepairImpact{Status: motor.RepairImpactNoBreaking}}
	var output bytes.Buffer
	accepted, err := reviewCLIRepair(strings.NewReader("\r"), &output, false, false, "api.json", []byte("{}"), preview)
	require.ErrorContains(t, err, "require terminal review")
	require.False(t, accepted)
	accepted, err = reviewCLIRepair(errorReader{}, &output, false, true, "api.json", []byte("{}"), preview)
	require.NoError(t, err, "dry run must not read stdin")
	require.False(t, accepted)
	accepted, err = reviewCLIRepair(strings.NewReader("\r"), errorWriter{}, true, false, "api.json", []byte("{}"), preview)
	require.ErrorContains(t, err, "cannot display fix review")
	require.False(t, accepted)
	accepted, err = reviewCLIRepair(strings.NewReader("\r"), shortReviewWriter{}, true, false, "api.json", []byte("{}"), preview)
	require.ErrorContains(t, err, "short write")
	require.False(t, accepted)
	accepted, err = reviewCLIRepair(strings.NewReader("yes"), &output, true, false, "api.json", []byte("{}"), preview)
	require.Error(t, err, "EOF without submitted line is not acceptance")
	require.False(t, accepted)
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, errors.New("must not read") }

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) { return 0, errors.New("display failed") }

func TestSilentRepairCannotApproveUnseenChanges(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		t.Run(fmt.Sprint(dryRun), func(t *testing.T) {
			source := []byte(`{"openapi":"3.1.0","info":{"title":"Review","version":"1"},"paths":{}}`)
			execution := &motor.RuleSetExecution{Spec: source, SpecFileName: "review.json", SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"info-description": rulesets.GetAllBuiltInRules()["info-description"]}}}
			var output bytes.Buffer
			result := runReviewedCLIRepairWithIO(execution, nil, &LintFlags{FixFlag: true, SilentFlag: true, DryRun: dryRun}, nil, errorReader{}, &output, true)
			defer result.ReleaseOwnedResources()
			require.Empty(t, output.String())
			if dryRun {
				require.Empty(t, result.Errors)
				require.Len(t, result.FixedResults, 1)
				require.NotEmpty(t, result.ModifiedSpec)
			} else {
				require.Len(t, result.Errors, 1)
				require.ErrorContains(t, result.Errors[0], "visible review")
				require.Empty(t, result.ModifiedSpec)
			}
			require.Equal(t, source, execution.Spec)
		})
	}
}

func TestSilentDryRunSuppressesReviewOutput(t *testing.T) {
	for _, tc := range []struct{ command, format string }{{"lint", ""}, {"schema", ""}, {"lint", "json"}, {"schema", "json"}} {
		t.Run(tc.command+tc.format, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "source.json")
			source := []byte(`{"openapi":"3.1.0","info":{"title":"Review","version":"1"},"paths":{}}`)
			var args []string
			if tc.command == "schema" {
				source = []byte(`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"string","enum":["a","a"]}`)
			} else {
				args = []string{"--ruleset", focusedRepairRules(t, dir, "info-description")}
			}
			require.NoError(t, os.WriteFile(path, source, 0600))
			args = append([]string{tc.command, path, "--fix", "--dry-run", "--silent", "--fail-severity", "none"}, args...)
			if tc.format != "" {
				args = append(args, "--format", tc.format)
			}
			out, errOut, err := exportCommand(t, args...)
			if tc.command == "lint" && tc.format == "json" {
				require.ErrorContains(t, err, "--format cannot be combined with --silent")
				require.Empty(t, out)
				require.Equal(t, "--format cannot be combined with --silent", strings.TrimSpace(errOut))
			} else if tc.format == "json" {
				require.NoError(t, err)
				require.True(t, json.Valid([]byte(out)), "structured output must remain valid")
				require.Empty(t, errOut)
			} else {
				require.NoError(t, err)
				require.Empty(t, out)
				require.Empty(t, errOut)
			}
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, source, data)
		})
	}
}

func TestReviewedCLIRepairAcceptsExactCandidate(t *testing.T) {
	original := []byte(`{"openapi":"3.1.0","info":{"title":"Pets","version":"1"},"paths":{},"components":{"schemas":{"Pet":{"type":"string","enum":["cat","cat"]}}}}`)
	for _, answer := range []string{"all\n\r", "all\n\x1b"} {
		t.Run(answer, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "api.json")
			require.NoError(t, os.WriteFile(path, original, 0600))
			execution := &motor.RuleSetExecution{Spec: original, SpecFileName: path, SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"duplicated-entry-in-enum": rulesets.GetAllBuiltInRules()["duplicated-entry-in-enum"]}}}
			var output bytes.Buffer
			result := runReviewedCLIRepairWithIO(execution, nil, &LintFlags{FixFlag: true}, nil, strings.NewReader(answer), &output, true)
			defer result.ReleaseOwnedResources()
			require.Empty(t, result.Errors)
			require.Contains(t, output.String(), "No breaking changes detected")
			saved, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, original, saved, "review does not write")
			if answer == "all\n\r" {
				require.NotEmpty(t, result.ModifiedSpec)
				require.Len(t, result.FixedResults, 1)
				require.NoError(t, writeFixedFile(result, path, ""))
				saved, err = os.ReadFile(path)
				require.NoError(t, err)
				require.Equal(t, result.ModifiedSpec, saved)
			} else {
				require.Empty(t, result.ModifiedSpec)
			}
		})
	}
}

func TestConsoleRepairSelectionRejectsUnknownAndDuplicateNumbers(t *testing.T) {
	options := []motor.FixOption{{Index: 2, Available: true, Result: &model.RuleFunctionResult{RuleId: "rule", Path: "$.info"}}}
	for _, answer := range []string{"1\n", "3 3\n", "yes\n"} {
		_, err := selectCLIRepairs(strings.NewReader(answer), &bytes.Buffer{}, options)
		require.Error(t, err)
	}
	choices, err := selectCLIRepairs(strings.NewReader("3\n"), &bytes.Buffer{}, options)
	require.NoError(t, err)
	require.Equal(t, []motor.FixChoice{{Index: 2}}, choices)
}

type shortReviewWriter struct{}

func (shortReviewWriter) Write(data []byte) (int, error) { return len(data) - 1, nil }

func TestReviewedCLIAllDeclinesConflictingFixesWithoutWrites(t *testing.T) {
	original := []byte(`{"openapi":"3.1.0","info":{"title":"Pets","version":"1"},"paths":{},"components":{"schemas":{"Pet":{"type":"string","enum":["cat","cat"]}}}}`)
	path := filepath.Join(t.TempDir(), "api.json")
	require.NoError(t, os.WriteFile(path, original, 0600))
	builtins := rulesets.GetAllBuiltInRules()
	execution := &motor.RuleSetExecution{Spec: original, SpecFileName: path, SilenceLogs: true, RuleSet: &rulesets.RuleSet{Rules: map[string]*model.Rule{"duplicated-entry-in-enum": builtins["duplicated-entry-in-enum"], "oas3-unused-component": builtins["oas3-unused-component"]}}}
	var output bytes.Buffer
	result := runReviewedCLIRepairWithIO(execution, nil, &LintFlags{FixFlag: true}, nil, strings.NewReader("all\n\r"), &output, true)
	defer result.ReleaseOwnedResources()
	require.Empty(t, result.Errors)
	require.Empty(t, result.ModifiedSpec)
	require.NotEmpty(t, result.RepairWarnings)
	require.Contains(t, strings.Join(result.RepairWarnings, "\n"), "repairs declined")
	require.Contains(t, strings.Join(result.RepairWarnings, "\n"), "together")
	require.NotContains(t, output.String(), "enter: apply", "invalid selection must not reach acceptance")
	require.Len(t, result.Results, 2, "both original findings must remain")
	saved, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, original, saved)
}

func TestConsoleAcceptanceShowsEveryChangedLine(t *testing.T) {
	for _, tc := range []struct{ name, original, candidate string }{
		{"separated", "first: old\n" + strings.Repeat("same: value\n", 300) + "last: old\n", "first: new\n" + strings.Repeat("same: value\n", 300) + "last: new\n"},
		{"long-json", `{"padding":"` + strings.Repeat("a", 10000) + `","last":"old"}`, `{"padding":"` + strings.Repeat("a", 10000) + `","last":"new"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			preview := &motor.FixPreview{Candidate: []byte(tc.candidate), Impact: motor.RepairImpact{Status: motor.RepairImpactNoBreaking}}
			accepted, err := reviewCLIRepair(strings.NewReader("\r"), &output, true, false, "api.yaml", []byte(tc.original), preview)
			require.NoError(t, err)
			require.True(t, accepted)
			if tc.name == "separated" {
				require.Contains(t, output.String(), "-last: old\n")
				require.Contains(t, output.String(), "+last: new\n")
			} else {
				require.Contains(t, output.String(), "-"+tc.original)
				require.Contains(t, output.String(), "+"+tc.candidate)
			}
			require.NotContains(t, output.String(), "omitted")
			require.NotContains(t, output.String(), "truncated")
			require.Greater(t, strings.Index(output.String(), "enter: apply"), strings.LastIndex(output.String(), "last"))
		})
	}
}

func TestConsoleAcceptanceStopsWhenFullDiffCannotBeWritten(t *testing.T) {
	preview := &motor.FixPreview{Candidate: []byte(strings.Repeat("new: value\n", 1000)), Impact: motor.RepairImpact{Status: motor.RepairImpactNoBreaking}}
	// Allow the styled table heading, then fail partway through the full diff.
	output := &limitedReviewWriter{remaining: 1024}
	accepted, err := reviewCLIRepair(errorReader{}, output, true, false, "api.yaml", []byte(strings.Repeat("old: value\n", 1000)), preview)
	require.ErrorContains(t, err, "cannot display fix review")
	require.False(t, accepted)
	require.Contains(t, output.String(), "--- original")
	require.NotContains(t, output.String(), "enter: apply")
}

type limitedReviewWriter struct {
	bytes.Buffer
	remaining int
}

func (w *limitedReviewWriter) Write(data []byte) (int, error) {
	if len(data) > w.remaining {
		n, _ := w.Buffer.Write(data[:w.remaining])
		w.remaining = 0
		return n, errors.New("display disconnected")
	}
	w.remaining -= len(data)
	return w.Buffer.Write(data)
}
func (w *limitedReviewWriter) WriteString(text string) (int, error) { return w.Write([]byte(text)) }

func TestConsoleSelectionEnterReviewsAllAndEscapeCancels(t *testing.T) {
	options := []motor.FixOption{{Index: 2, Available: true, Result: &model.RuleFunctionResult{RuleId: "test", Path: "$.info"}}}
	for _, tc := range []struct {
		keys  string
		count int
	}{{"\r", 1}, {"\x1b", 0}, {"3\x7f3\r", 1}} {
		var output bytes.Buffer
		selected, err := selectCLIRepairs(strings.NewReader(tc.keys), &output, options)
		require.NoError(t, err)
		require.Len(t, selected, tc.count)
		require.NotContains(t, output.String(), "enter: cancel")
	}
}
