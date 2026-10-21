package harness

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	schemautil "github.com/daveshanley/vacuum/jsonschema"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/plugin"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/index"
)

type runner struct {
	base        string
	suite       *Suite
	opts        RunOptions
	ctx         context.Context
	logger      *slog.Logger
	files       map[string]string
	limitations map[string]bool
	functions   map[string]model.RuleFunction
}

// RunFile loads and executes a suite. All failures, including loading failures,
// are returned in a serializable report with exit code 0, 1, or 2.
func RunFile(path string, opts RunOptions) *Report {
	s, err := Load(path)
	if err != nil {
		return errorReport(ErrorSuite, PhaseLoad, err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return errorReport(ErrorSuite, PhaseLoad, err)
	}
	return Run(s, abs, opts)
}

// Run executes a validated suite using suitePath as its location. Settings are
// explicit; no process environment or vacuum configuration is consulted.
func Run(s *Suite, suitePath string, opts RunOptions) (report *Report) {
	defer func() {
		if v := recover(); v != nil {
			report = errorReport(ErrorInternal, PhaseLoad, fmt.Errorf("harness execution panicked: %v", v))
		}
	}()
	if err := s.Validate(); err != nil {
		return errorReport(ErrorSuite, PhaseLoad, err)
	}
	if opts.SampleLimit < 0 || opts.SampleLimit > 100 {
		return errorReport(ErrorArguments, PhaseLoad, fmt.Errorf("sample limit must be between 0 and 100"))
	}
	abs, err := filepath.Abs(suitePath)
	if err != nil {
		return errorReport(ErrorSuite, PhaseLoad, err)
	}
	r := &runner{base: filepath.Dir(abs), suite: s, opts: opts, ctx: opts.Context, logger: slog.New(slog.NewTextHandler(io.Discard, nil)), files: map[string]string{}, limitations: map[string]bool{}, functions: map[string]model.RuleFunction{}}
	if r.ctx == nil {
		r.ctx = context.Background()
	}
	for _, dependency := range []string{s.Ruleset, s.Functions, s.IgnoreFile} {
		r.trackAbsolutePath(dependency)
	}
	for _, c := range s.Cases {
		r.trackAbsolutePath(c.Input)
		r.trackAbsolutePath(c.After)
		if c.Fix != nil {
			r.trackAbsolutePath(c.Fix.Output)
		}
	}
	report = &Report{Version: 1, Status: "passed", Cases: []CaseResult{}, Errors: []ExecutionError{}, Manifest: Manifest{VacuumVersion: opts.Version, Suite: filepath.Base(abs), Options: effectiveOptions(s.Options), Files: []FileIdentity{}, Limitations: []string{}}}
	effective, _ := json.Marshal(s)
	effectiveHash := sha256.Sum256(effective)
	report.Manifest.SuiteSHA256 = hex.EncodeToString(effectiveHash[:])
	report.Manifest.CaseIDs = append([]string{}, opts.CaseIDs...)
	report.Manifest.SampleLimit = opts.SampleLimit
	if disk, err := Load(abs); err != nil {
		r.limitations["suite supplied in memory differs from its file"] = true
	} else if b, _ := json.Marshal(disk); string(b) != string(effective) {
		r.limitations["suite supplied in memory differs from its file"] = true
	}
	if _, err := r.read(abs); err != nil {
		r.limitations["suite supplied in memory; suite file unavailable"] = true
	}
	r.opts.AutoFixFunctions = make(map[string]model.AutoFixFunction, len(opts.AutoFixFunctions))
	for name, f := range opts.AutoFixFunctions {
		r.opts.AutoFixFunctions[name] = f
	}
	for name, f := range opts.CustomFunctions {
		r.functions[name] = f
	}
	if len(opts.CustomFunctions) > 0 || len(opts.AutoFixFunctions) > 0 {
		r.limitations["caller-provided functions require the same host registration"] = true
	}
	if s.Functions != "" {
		dir := r.path(s.Functions)
		entries, e := os.ReadDir(dir)
		if e != nil {
			return errorReport(ErrorFunctionLoad, PhaseLoad, e)
		}
		for _, entry := range entries {
			if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".js") || strings.HasSuffix(entry.Name(), ".so")) {
				if _, e := r.read(filepath.Join(dir, entry.Name())); e != nil {
					return errorReport(ErrorFunctionLoad, PhaseLoad, e)
				}
			}
		}
		manager, e := plugin.LoadFunctionsStrict(dir, time.Duration(effectiveOptions(s.Options).TimeoutMS)*time.Millisecond)
		if e != nil {
			report := errorReport(ErrorFunctionLoad, PhaseLoad, e)
			for i := range report.Errors {
				if report.Errors[i].Source != "" {
					report.Errors[i].Source = r.relative(report.Errors[i].Source)
				}
			}
			return report
		}
		for name, f := range manager.GetAutoFixFunctions() {
			if r.opts.AutoFixFunctions == nil {
				r.opts.AutoFixFunctions = map[string]model.AutoFixFunction{}
			}
			r.opts.AutoFixFunctions[name] = f
		}
		for name, f := range manager.GetCustomFunctions() {
			r.functions[name] = f
		}
		r.limitations["custom functions may read additional files, time, or network state"] = true
	}
	selected := map[string]bool{}
	for _, id := range opts.CaseIDs {
		selected[id] = false
	}
	for _, c := range s.Cases {
		if _, ok := selected[c.ID]; ok {
			selected[c.ID] = true
		}
	}
	selectedIDs := make([]string, 0, len(selected))
	for id := range selected {
		selectedIDs = append(selectedIDs, id)
	}
	sort.Strings(selectedIDs)
	for _, id := range selectedIDs {
		if !selected[id] {
			return errorReport(ErrorSelection, PhaseLoad, fmt.Errorf("unknown case %q", id))
		}
	}
	for _, c := range s.Cases {
		if len(selected) > 0 && !selected[c.ID] {
			continue
		}
		cr := r.runCase(c)
		report.Cases = append(report.Cases, cr)
		if cr.Status == "error" {
			report.ExitCode = 2
		} else if cr.Status == "failed" && report.ExitCode == 0 {
			report.ExitCode = 1
		}
	}
	if report.ExitCode == 1 {
		report.Status = "failed"
	}
	if report.ExitCode == 2 {
		report.Status = "error"
	}
	for path, hash := range r.files {
		report.Manifest.Files = append(report.Manifest.Files, FileIdentity{Path: path, SHA256: hash})
	}
	sort.Slice(report.Manifest.Files, func(i, j int) bool { return report.Manifest.Files[i].Path < report.Manifest.Files[j].Path })
	for reason := range r.limitations {
		report.Manifest.Limitations = append(report.Manifest.Limitations, reason)
	}
	sort.Strings(report.Manifest.Limitations)
	report.Manifest.Reproducible = len(report.Manifest.Limitations) == 0
	return report
}
func errorReport(code ErrorCode, phase Phase, err error) *Report {
	return &Report{Version: 1, Status: "error", ExitCode: 2, Cases: []CaseResult{}, Errors: (&runner{}).executionErrors(err, code, phase, "", ""), Manifest: Manifest{Files: []FileIdentity{}, Limitations: []string{"execution did not start"}}}
}
func effectiveOptions(o Options) Options {
	if o.TimeoutMS == 0 {
		o.TimeoutMS = 5000
	}
	if o.LookupTimeoutMS == 0 {
		o.LookupTimeoutMS = 500
	}
	if o.MaxRuleConcurrency == 0 {
		o.MaxRuleConcurrency = 32
	}
	return o
}
func (r *runner) path(p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Join(r.base, p)
}
func (r *runner) relative(p string) string {
	if strings.Contains(p, "://") {
		return p
	}
	a, err := filepath.Rel(r.base, r.path(p))
	if err != nil {
		return filepath.ToSlash(p)
	}
	return filepath.ToSlash(a)
}
func (r *runner) read(p string) ([]byte, error) {
	p = r.path(p)
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	h := sha256.Sum256(b)
	key := r.relative(p)
	if old, ok := r.files[key]; ok && old != hex.EncodeToString(h[:]) {
		return nil, fmt.Errorf("dependency changed during run: %s", key)
	}
	r.files[key] = hex.EncodeToString(h[:])
	return b, nil
}
func (r *runner) loadRules(c Case) (*rulesets.RuleSet, error) {
	path := r.path(r.suite.Ruleset)
	if err := r.trackRules(path, map[string]bool{}); err != nil {
		return nil, err
	}
	rs, err := rulesets.LoadLocalRuleSet(r.ctx, path)
	if err != nil {
		return nil, err
	}
	rs = rulesets.BuildDefaultRuleSetsWithLogger(r.logger).GenerateRuleSetFromSuppliedRuleSet(rs)
	if err := rs.LoadError(); err != nil {
		return nil, err
	}
	if len(rs.Rules) == 0 {
		return nil, fmt.Errorf("ruleset contains no enabled rules")
	}
	if len(c.Rules) > 0 {
		selected := make(map[string]*model.Rule, len(c.Rules))
		for _, id := range c.Rules {
			rule, ok := rs.Rules[id]
			if !ok || rule == nil {
				return nil, &motor.RuleExecutionError{Kind: motor.RuleErrorConfiguration, RuleID: id, Err: fmt.Errorf("rule is unknown or disabled")}
			}
			selected[id] = rule
		}
		rs.Rules = selected
	}
	ids := map[string]bool{}
	for id, rule := range rs.Rules {
		if rule != nil {
			ids[id] = true
		}
	}
	expectations := []*Expectations{&c.Expect, c.AfterExpect}
	if c.Fix != nil {
		expectations = append(expectations, &c.Fix.Expect)
	}
	for _, e := range expectations {
		if e != nil {
			for _, id := range slices.Sorted(maps.Keys(e.Execution)) {
				if !ids[id] {
					return nil, fmt.Errorf("execution assertion refers to unselected rule %q", id)
				}
			}
		}
	}
	if errs := motor.ValidateRuleSet(rs, r.functions); len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return rs, nil
}
func (r *runner) trackRules(path string, seen map[string]bool) error {
	if seen[path] {
		return nil
	}
	seen[path] = true
	b, err := r.read(path)
	if err != nil {
		return err
	}
	rs, err := rulesets.CreateRuleSetFromData(b)
	if err != nil {
		return err
	}
	for ref := range rs.GetExtendsValue() {
		r.trackAbsolutePath(ref)
		if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
			r.limitations["remote ruleset extensions are not captured"] = true
			continue
		}
		if strings.HasPrefix(ref, "vacuum:") || strings.HasPrefix(ref, "spectral:") {
			continue
		}
		next := ref
		if !filepath.IsAbs(next) {
			next = filepath.Join(filepath.Dir(path), next)
		}
		if err := r.trackRules(next, seen); err != nil {
			return err
		}
	}
	return nil
}

type execution struct {
	snapshot *Snapshot
	values   []model.RuleFunctionResult
	modified []byte
	errs     []ExecutionError
}

func (r *runner) execute(c Case, path string, override []byte, fix bool, phase Phase) execution {
	out := execution{errs: []ExecutionError{}}
	fail := func(code ErrorCode, source string, err error) {
		out.errs = append(out.errs, r.executionErrors(err, code, phase, c.ID, source)...)
	}
	if err := r.ctx.Err(); err != nil {
		fail(ErrorCancelled, path, err)
		return out
	}
	rs, err := r.loadRules(c)
	if err != nil {
		fail(ErrorRuleset, r.suite.Ruleset, err)
		return out
	}
	b := override
	if b == nil {
		b, err = r.read(path)
		if err != nil {
			fail(ErrorInput, path, err)
			return out
		}
	}
	var root yaml.Node
	if err := decodeSingleDocument(b, &root); err != nil {
		fail(ErrorDocument, path, err)
		return out
	}
	o := effectiveOptions(r.suite.Options)
	evidence := &motor.ExecutionEvidence{SampleLimit: r.opts.SampleLimit}
	if fix {
		for _, id := range slices.Sorted(maps.Keys(rs.Rules)) {
			rule := rs.Rules[id]
			if rule.AutoFixFunction != "" && r.opts.AutoFixFunctions[rule.AutoFixFunction] == nil {
				fail(ErrorAutoFix, path, &motor.RuleExecutionError{Kind: motor.RuleErrorAutoFix, RuleID: id, Function: rule.AutoFixFunction, Err: fmt.Errorf("autofix is not registered")})
			}
		}
		if len(out.errs) > 0 {
			return out
		}
	}
	ex := &motor.RuleSetExecution{Spec: b, SpecFileName: r.path(path), Base: filepath.Dir(r.path(path)), RuleSet: rs, Logger: r.logger, SilenceLogs: true, CustomFunctions: r.functions, AutoFixFunctions: r.opts.AutoFixFunctions, ApplyAutoFixes: fix, AllowLookup: o.Remote, Timeout: time.Duration(o.TimeoutMS) * time.Millisecond, NodeLookupTimeout: time.Duration(o.LookupTimeoutMS) * time.Millisecond, SkipDocumentCheck: c.Format == "fragment"}
	if c.Format == "json-schema" {
		ex.SkipDocumentCheck = true
		ex.SpecFormat = schemautil.DetectDialect(&root).Format
	}
	result := motor.ApplyRulesToRuleSetWithOptions(ex, &motor.ExecutionOptions{ReportFunctionErrors: true, Context: r.ctx, RunTimeout: time.Duration(o.RunTimeoutMS) * time.Millisecond, MaxRuleConcurrency: o.MaxRuleConcurrency, ResolveAllRefs: o.ResolveAllRefs, NestedRefsDocContext: o.NestedRefsDocContext, Evidence: evidence})
	defer result.ReleaseOwnedResources()
	for _, e := range result.Errors {
		fail(ErrorExecution, path, e)
	}
	if len(evidence.Rules) == 0 && len(rs.Rules) > 0 && len(out.errs) == 0 {
		fail(ErrorEvidence, path, fmt.Errorf("rule execution evidence unavailable"))
	}
	if o.Remote {
		r.limitations["remote document lookup enabled; external responses are not captured"] = true
	}
	r.trackDocumentPaths(&root)
	if ex.Arazzo != nil {
		for _, source := range slices.Sorted(maps.Keys(ex.Arazzo.Sources)) {
			if strings.Contains(source, "://") {
				r.limitations["remote Arazzo sources are not captured"] = true
			} else if _, err := r.read(source); err != nil {
				fail(ErrorDependency, source, err)
			}
		}
	}
	seenDependencies := map[string]bool{}
	for _, idx := range []*index.SpecIndex{ex.IndexResolved, ex.IndexUnresolved, result.Index} {
		if idx == nil || idx.GetRolodex() == nil {
			continue
		}
		for _, dep := range idx.GetRolodex().GetIndexes() {
			source := dep.GetSpecAbsolutePath()
			if source == "" || seenDependencies[source] {
				continue
			}
			seenDependencies[source] = true
			r.trackDocumentPaths(dep.GetRootNode())
			if strings.Contains(source, "://") {
				r.limitations["remote document responses are not captured"] = true
				continue
			}
			if _, err := r.read(source); err != nil {
				fail(ErrorDependency, source, err)
			}
		}
	}
	values := result.Results
	if r.suite.IgnoreFile != "" {
		data, e := r.read(r.suite.IgnoreFile)
		if e != nil {
			fail(ErrorIgnoreFile, r.suite.IgnoreFile, e)
		} else {
			var ignored model.IgnoredItems
			if e = yaml.Unmarshal(data, &ignored); e != nil {
				fail(ErrorIgnoreFile, r.suite.IgnoreFile, e)
			} else {
				matcher := utils.NewIgnoreMatcher(ignored, utils.IgnoreMatcherOptions{RootNode: ex.CanonicalDocument, SpecBytes: b, LookupTimeout: time.Duration(o.LookupTimeoutMS) * time.Millisecond})
				kept := make([]model.RuleFunctionResult, 0, len(values))
				for i := range values {
					if matcher.Matches(&values[i]) {
						for j := range evidence.Rules {
							if evidence.Rules[j].RuleID == values[i].RuleId {
								evidence.Rules[j].Ignored++
								evidence.Rules[j].Findings--
							}
						}
					} else {
						kept = append(kept, values[i])
					}
				}
				values = kept
			}
		}
	}
	out.snapshot = &Snapshot{Diagnostics: r.diagnostics(values, path), Evidence: evidence.Rules, Rules: rs.Rules}
	out.values = detachedValues(out.snapshot.Diagnostics, r.base)
	out.modified = append([]byte(nil), result.ModifiedSpec...)
	return out
}
func (r *runner) diagnostics(values []model.RuleFunctionResult, input string) []Diagnostic {
	out := make([]Diagnostic, 0, len(values))
	for _, v := range values {
		d := Diagnostic{Rule: v.RuleId, Severity: v.RuleSeverity, Path: v.Path, Paths: append([]string(nil), v.Paths...), Source: r.relative(input), Message: v.Message, PathsTruncated: v.PathsTruncated}
		if v.Rule != nil {
			if d.Rule == "" {
				d.Rule = v.Rule.Id
			}
			if d.Severity == "" {
				d.Severity = v.Rule.Severity
			}
		}
		if v.StartNode != nil {
			d.Line = v.StartNode.Line
			d.Column = v.StartNode.Column
		}
		if v.Origin != nil && v.Origin.AbsoluteLocation != "" {
			d.Source = r.relative(v.Origin.AbsoluteLocation)
			if v.Origin.Line > 0 {
				d.Line = v.Origin.Line
				d.Column = v.Origin.Column
			}
		}
		sort.Strings(d.Paths)
		out = append(out, d)
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Rule != b.Rule {
			return a.Rule < b.Rule
		}
		if a.Source != b.Source {
			return a.Source < b.Source
		}
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Column != b.Column {
			return a.Column < b.Column
		}
		return a.Message < b.Message
	})
	return out
}
func detachedValues(ds []Diagnostic, base string) []model.RuleFunctionResult {
	out := make([]model.RuleFunctionResult, 0, len(ds))
	for _, d := range ds {
		source := d.Source
		if !strings.Contains(source, "://") {
			source = filepath.Join(base, source)
		}
		out = append(out, model.RuleFunctionResult{RuleId: d.Rule, RuleSeverity: d.Severity, Path: d.Path, Paths: d.Paths, Message: d.Message, PathsTruncated: d.PathsTruncated, Origin: &index.NodeOrigin{AbsoluteLocation: source, Line: d.Line, Column: d.Column}, StartNode: &yaml.Node{Line: d.Line, Column: d.Column}})
	}
	return out
}
