package harness

import (
	"reflect"
	"sort"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
)

func (r *runner) runCase(c Case) CaseResult {
	cr := CaseResult{ID: c.ID, Status: "passed", Failures: []Failure{}, Errors: []string{}}
	before := r.execute(c, c.Input, nil, false)
	cr.Before = before.snapshot
	cr.Errors = append(cr.Errors, before.errs...)
	if len(cr.Errors) > 0 {
		cr.Status = "error"
		return cr
	}
	cr.Failures = append(cr.Failures, assertSnapshot("expect", c.Expect, before.snapshot)...)
	if c.After != "" {
		after := r.execute(c, c.After, nil, false)
		cr.After = after.snapshot
		cr.Errors = append(cr.Errors, after.errs...)
		if len(after.errs) == 0 {
			if c.AfterExpect != nil {
				cr.Failures = append(cr.Failures, assertSnapshot("afterExpect", *c.AfterExpect, after.snapshot)...)
			}
			if c.Change != nil {
				for _, d := range append(append([]Diagnostic{}, before.snapshot.Diagnostics...), after.snapshot.Diagnostics...) {
					if d.PathsTruncated {
						cr.Errors = append(cr.Errors, "change comparison requires complete diagnostic paths")
						break
					}
				}
				added := diffBySeverity(before.values, after.values, r.path(c.Input), r.path(c.After))
				removed := diffBySeverity(after.values, before.values, r.path(c.After), r.path(c.Input))
				addedD, removedD := r.diagnostics(added, c.After), r.diagnostics(removed, c.Input)
				cr.Failures = append(cr.Failures, exactDiagnostics("change.added", c.Change.Added, addedD)...)
				cr.Failures = append(cr.Failures, exactDiagnostics("change.removed", c.Change.Removed, removedD)...)
				if c.Change.NoNew && len(addedD) > 0 {
					cr.Failures = append(cr.Failures, Failure{Assertion: "change.noNew", Code: "new_diagnostics", Expected: 0, Actual: addedD})
				}
				if c.Change.Unchanged && (len(addedD) > 0 || len(removedD) > 0) {
					cr.Failures = append(cr.Failures, Failure{Assertion: "change.unchanged", Code: "diagnostics_changed", Expected: "unchanged", Actual: map[string]any{"added": addedD, "removed": removedD}})
				}
			}
		}
	}
	if c.Fix != nil {
		r.checkFix(c, &cr)
	}
	if len(cr.Errors) > 0 {
		cr.Status = "error"
	} else if len(cr.Failures) > 0 {
		cr.Status = "failed"
	}
	return cr
}

func (r *runner) checkFix(c Case, cr *CaseResult) {
	fixed := r.execute(c, c.Input, nil, true)
	cr.Fixed = fixed.snapshot
	cr.Errors = append(cr.Errors, fixed.errs...)
	if len(fixed.errs) > 0 {
		return
	}
	modified := fixed.modified
	if modified == nil {
		var err error
		modified, err = r.read(c.Input)
		if err != nil {
			cr.Errors = append(cr.Errors, err.Error())
			return
		}
	}
	expected, err := r.read(c.Fix.Output)
	if err != nil {
		cr.Errors = append(cr.Errors, err.Error())
		return
	}
	equal, err := semanticEqual(expected, modified)
	if err != nil {
		cr.Errors = append(cr.Errors, err.Error())
		return
	}
	if !equal {
		cr.Failures = append(cr.Failures, Failure{Assertion: "fix.output", Code: "document_mismatch", Expected: string(expected), Actual: string(modified)})
	}
	after := r.execute(c, c.Input, modified, false)
	cr.After = after.snapshot
	cr.Errors = append(cr.Errors, after.errs...)
	if len(after.errs) > 0 {
		return
	}
	cr.Failures = append(cr.Failures, assertSnapshot("fix.expect", c.Fix.Expect, after.snapshot)...)
	again := r.execute(c, c.Input, modified, true)
	cr.SecondFix = again.snapshot
	cr.Errors = append(cr.Errors, again.errs...)
	if len(again.errs) > 0 {
		return
	}
	if again.modified != nil {
		equal, err = semanticEqual(modified, again.modified)
		if err != nil {
			cr.Errors = append(cr.Errors, err.Error())
		} else if !equal {
			cr.Failures = append(cr.Failures, Failure{Assertion: "fix.idempotent", Code: "non_idempotent_fix", Expected: "no semantic change", Actual: string(again.modified)})
		}
	}
}
func semanticEqual(a, b []byte) (bool, error) {
	decode := func(data []byte) (any, error) {
		var value any
		if err := decodeSingleDocument(data, &value); err != nil {
			return nil, err
		}
		return value, nil
	}
	x, err := decode(a)
	if err != nil {
		return false, err
	}
	y, err := decode(b)
	return reflect.DeepEqual(x, y), err
}

// Keep vacuum's alias/source identity while treating severity changes as changes
// in the test contract. The normal lint differ intentionally ignores severity.
func diffBySeverity(before, after []model.RuleFunctionResult, beforePath, afterPath string) []model.RuleFunctionResult {
	oldGroups, newGroups := map[string][]model.RuleFunctionResult{}, map[string][]model.RuleFunctionResult{}
	for _, v := range before {
		oldGroups[v.RuleSeverity] = append(oldGroups[v.RuleSeverity], v)
	}
	for _, v := range after {
		newGroups[v.RuleSeverity] = append(newGroups[v.RuleSeverity], v)
	}
	keys := make([]string, 0, len(newGroups))
	for key := range newGroups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var results []model.RuleFunctionResult
	for _, key := range keys {
		added, _ := utils.DiffViolationsValuesWithOriginBases(oldGroups[key], newGroups[key], beforePath, afterPath)
		results = append(results, added...)
	}
	return results
}
