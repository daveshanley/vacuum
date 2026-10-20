package harness

import (
	"fmt"
	"path/filepath"
	"slices"
	"sort"
)

func matches(w DiagnosticMatch, a Diagnostic) bool {
	if w.Rule != a.Rule || w.Severity != "" && w.Severity != a.Severity || w.Path != "" && w.Path != a.Path || w.Source != "" && filepath.ToSlash(filepath.Clean(w.Source)) != a.Source || w.Line != 0 && w.Line != a.Line || w.Column != 0 && w.Column != a.Column || w.Message != nil && *w.Message != a.Message {
		return false
	}
	if w.Paths != nil {
		if a.PathsTruncated {
			return false
		}
		x, y := slices.Clone(w.Paths), slices.Clone(a.Paths)
		sort.Strings(x)
		sort.Strings(y)
		if !slices.Equal(x, y) {
			return false
		}
	}
	return true
}

// exactDiagnostics uses bipartite matching so a broad assertion cannot consume
// the sole result satisfying a later, more specific assertion.
func exactDiagnostics(id string, wants []DiagnosticMatch, actual []Diagnostic) []Failure {
	if wants == nil {
		return nil
	}
	assigned := make([]int, len(actual))
	for i := range assigned {
		assigned[i] = -1
	}
	var visit func(int, []bool) bool
	visit = func(w int, seen []bool) bool {
		for a := range actual {
			if seen[a] || !matches(wants[w], actual[a]) {
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
	var failures []Failure
	for w := range wants {
		if !visit(w, make([]bool, len(actual))) {
			failures = append(failures, Failure{Assertion: fmt.Sprintf("%s[%d]", id, w), Code: "missing_diagnostic", Expected: wants[w], Actual: nil})
		}
	}
	for a, w := range assigned {
		if w < 0 {
			failures = append(failures, Failure{Assertion: id, Code: "unexpected_diagnostic", Expected: nil, Actual: actual[a]})
		}
	}
	return failures
}
func assertSnapshot(prefix string, e Expectations, s *Snapshot) []Failure {
	out := exactDiagnostics(prefix+".diagnostics", e.Diagnostics, s.Diagnostics)
	ids := make([]string, 0, len(e.Execution))
	for id := range e.Execution {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		want := e.Execution[id]
		found := false
		for _, a := range s.Evidence {
			if a.RuleID != id {
				continue
			}
			found = true
			if want.Status != a.Status {
				out = append(out, Failure{Assertion: prefix + ".execution." + id + ".status", Code: "execution_status", Expected: want.Status, Actual: a.Status})
			}
			for _, v := range []struct {
				name  string
				count *Count
				value int
			}{{"matches", want.Matches, a.Matches}, {"invocations", want.Invocations, a.Invocations}, {"ignored", want.Ignored, a.Ignored}, {"fixed", want.Fixed, a.Fixed}} {
				if v.count != nil && (v.count.Min != nil && v.value < *v.count.Min || v.count.Max != nil && v.value > *v.count.Max) {
					out = append(out, Failure{Assertion: prefix + ".execution." + id + "." + v.name, Code: "count_mismatch", Expected: v.count, Actual: v.value})
				}
			}
		}
		if !found {
			out = append(out, Failure{Assertion: prefix + ".execution." + id, Code: "missing_evidence", Expected: want, Actual: nil})
		}
	}
	return out
}
