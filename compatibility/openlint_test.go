package compatibility

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/model/reports"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
)

// This corpus checks the fields asserted by the pinned upstream fixtures. It is
// not a claim of full OpenLint conformance or identical diagnostic messages.
type resultMatch struct {
	Code *string   `json:"code,omitempty"`
	Path *[]string `json:"path,omitempty"`
}

type functionFixture struct {
	For      []string `json:"for"`
	Purpose  string   `json:"purpose"`
	Ruleset  string   `json:"ruleset"`
	Document string   `json:"document"`
	Expect   struct {
		Results []resultMatch `json:"results"`
	} `json:"expect"`
}

type difference struct {
	Reason  string        `json:"reason"`
	Results []resultMatch `json:"results,omitempty"`
	Errors  []string      `json:"errors,omitempty"`
}

func readJSON(t *testing.T, path string, target any) {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	require.NoError(t, dec.Decode(target))
	require.Equal(t, io.EOF, dec.Decode(new(any)), "trailing JSON is not a fixture")
}

func TestOpenLintFunctions(t *testing.T) {
	files, err := filepath.Glob("testdata/openlint/functions/*/test.json")
	require.NoError(t, err)
	require.Len(t, files, 31, "the pinned corpus must not silently lose cases")
	var differences map[string]difference
	readJSON(t, "differences.json", &differences)
	seen := map[string]bool{}
	compatible, different, unsupported := 0, 0, 0
	for _, file := range files {
		name := filepath.Base(filepath.Dir(file))
		seen[name] = true
		t.Run(name, func(t *testing.T) {
			var fixture functionFixture
			readJSON(t, file, &fixture)
			require.NotEmpty(t, fixture.For)
			require.NotEmpty(t, fixture.Purpose)
			require.NotNil(t, fixture.Expect.Results, "missing expectations must fail")
			for _, want := range fixture.Expect.Results {
				require.True(t, want.Code != nil || want.Path != nil, "empty result matcher")
			}
			dir, err := filepath.Abs(filepath.Dir(file))
			require.NoError(t, err)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			raw, err := rulesets.LoadLocalRuleSet(ctx, filepath.Join(dir, fixture.Ruleset))
			require.NoError(t, err)
			rs := rulesets.BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(raw)
			require.NoError(t, rs.LoadError())
			// Invalid options must never masquerade as expected lint findings.
			preflight := motor.ValidateRuleSet(rs, nil)
			known, isDifferent := differences[name]
			if isDifferent {
				require.NotEmpty(t, known.Reason)
				require.True(t, (known.Results != nil) != (known.Errors != nil), "difference requires results or errors")
			}
			if len(preflight) > 0 {
				errors := make([]string, len(preflight))
				for i, e := range preflight {
					errors[i] = e.Error()
				}
				require.True(t, isDifferent, "unexpected preflight failure: %v", errors)
				require.Equal(t, known.Errors, errors, "unsupported behavior changed; reassess the difference")
				require.Empty(t, known.Results)
				unsupported++
				t.Logf("UNSUPPORTED: %s", known.Reason)
				return
			}
			require.Empty(t, known.Errors, "previously unsupported case now loads; remove or update its difference")
			doc := filepath.Join(dir, fixture.Document)
			spec, err := os.ReadFile(doc)
			require.NoError(t, err)
			result := motor.ApplyRulesToRuleSetWithOptions(&motor.RuleSetExecution{RuleSet: rs, Spec: spec, SpecFileName: doc, Base: dir, SkipDocumentCheck: true, SilenceLogs: true}, &motor.ExecutionOptions{Context: ctx, RunTimeout: 5 * time.Second, ReportFunctionErrors: true})
			defer result.Release()
			require.Empty(t, result.Errors, "execution failures cannot satisfy a diagnostic expectation")
			require.NoError(t, ctx.Err())
			actual := model.NewRuleResultSet(result.Results).GenerateSpectralReport(doc)
			matches := matchResults(fixture.Expect.Results, actual)
			if isDifferent {
				require.False(t, matches, "now compatible; remove the difference")
				require.True(t, matchResults(known.Results, actual), "known difference changed: %+v", actual)
				different++
				t.Logf("DIFFERENT: %s", known.Reason)
			} else {
				expectedJSON, err := json.Marshal(fixture.Expect.Results)
				require.NoError(t, err)
				require.True(t, matches, "%s: expected %s; got %+v", fixture.Purpose, expectedJSON, actual)
				compatible++
				t.Log("COMPATIBLE")
			}
		})
	}
	for name := range differences {
		require.True(t, seen[name], "difference without a fixture: %s", name)
	}
	t.Logf("Selected cases: %d compatible, %d different, %d unsupported; corpus contains 31 fixtures", compatible, different, unsupported)
}

// Match an unordered multiset. A broad expectation must not consume the only
// diagnostic that satisfies a later, narrower expectation.
func matchResults(want []resultMatch, actual []reports.SpectralReport) bool {
	if len(want) != len(actual) {
		return false
	}
	assigned := make([]int, len(actual))
	for i := range assigned {
		assigned[i] = -1
	}
	var visit func(int, []bool) bool
	visit = func(w int, seen []bool) bool {
		for a, got := range actual {
			if seen[a] || want[w].Code != nil && *want[w].Code != got.Code || want[w].Path != nil && (got.Path == nil || !slices.Equal(*want[w].Path, got.Path)) {
				continue
			}
			seen[a] = true
			if assigned[a] < 0 || visit(assigned[a], seen) {
				assigned[a] = w
				return true
			}
		}
		return false
	}
	for w := range want {
		if !visit(w, make([]bool, len(actual))) {
			return false
		}
	}
	return true
}
