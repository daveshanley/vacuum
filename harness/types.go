// Package harness executes declarative rule tests through vacuum's motor.
// Suites use explicit settings and do not read vacuum.conf.yaml or environment
// overrides. Each case owns its documents; callers own registered functions.
package harness

import (
	"context"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
)

// Suite is the versioned, suite-relative test contract.
type Suite struct {
	Version    int     `json:"version" yaml:"version"`
	Ruleset    string  `json:"ruleset,omitempty" yaml:"ruleset,omitempty"`
	Functions  string  `json:"functions,omitempty" yaml:"functions,omitempty"`
	IgnoreFile string  `json:"ignoreFile,omitempty" yaml:"ignoreFile,omitempty"`
	Options    Options `json:"options,omitempty" yaml:"options,omitempty"`
	Cases      []Case  `json:"cases" yaml:"cases"`
}

// Options controls execution. Zero limits use bounded motor defaults.
// Remote applies to document references; remote ruleset extensions use the
// existing ruleset loader and are marked non-reproducible in the manifest.
type Options struct {
	TimeoutMS            int  `json:"timeoutMs,omitempty" yaml:"timeoutMs,omitempty"`
	RunTimeoutMS         int  `json:"runTimeoutMs,omitempty" yaml:"runTimeoutMs,omitempty"`
	LookupTimeoutMS      int  `json:"lookupTimeoutMs,omitempty" yaml:"lookupTimeoutMs,omitempty"`
	MaxRuleConcurrency   int  `json:"maxRuleConcurrency,omitempty" yaml:"maxRuleConcurrency,omitempty"`
	Remote               bool `json:"remote,omitempty" yaml:"remote,omitempty"`
	ResolveAllRefs       bool `json:"resolveAllRefs,omitempty" yaml:"resolveAllRefs,omitempty"`
	NestedRefsDocContext bool `json:"nestedRefsDocContext,omitempty" yaml:"nestedRefsDocContext,omitempty"`
}

// Case describes one independent execution and optional change verification.
type Case struct {
	ID          string              `json:"id" yaml:"id"`
	Input       string              `json:"input" yaml:"input"`
	Format      string              `json:"format,omitempty" yaml:"format,omitempty"`
	Rules       []string            `json:"rules,omitempty" yaml:"rules,omitempty"`
	Expect      Expectations        `json:"expect" yaml:"expect"`
	After       string              `json:"after,omitempty" yaml:"after,omitempty"`
	AfterExpect *Expectations       `json:"afterExpect,omitempty" yaml:"afterExpect,omitempty"`
	Change      *ChangeExpectations `json:"change,omitempty" yaml:"change,omitempty"`
	Fix         *FixExpectations    `json:"fix,omitempty" yaml:"fix,omitempty"`
}

// Expectations checks an exact, unordered diagnostic multiset and rule evidence.
// A nil Diagnostics leaves findings unconstrained; an empty array requires none.
type Expectations struct {
	Diagnostics []DiagnosticMatch         `json:"diagnostics" yaml:"diagnostics"`
	Execution   map[string]ExecutionMatch `json:"execution,omitempty" yaml:"execution,omitempty"`
}

// DiagnosticMatch matches only supplied fields. Every entry consumes one result.
// Paths compares the complete alternate-path set; Path matches the primary path.
type DiagnosticMatch struct {
	Rule     string   `json:"rule" yaml:"rule"`
	Severity string   `json:"severity,omitempty" yaml:"severity,omitempty"`
	Path     string   `json:"path,omitempty" yaml:"path,omitempty"`
	Paths    []string `json:"paths" yaml:"paths"`
	Source   string   `json:"source,omitempty" yaml:"source,omitempty"`
	Line     int      `json:"line,omitempty" yaml:"line,omitempty"`
	Column   int      `json:"column,omitempty" yaml:"column,omitempty"`
	Message  *string  `json:"message,omitempty" yaml:"message,omitempty"`
}

// Count checks an inclusive range. Set both bounds for an exact count.
type Count struct {
	Min *int `json:"min,omitempty" yaml:"min,omitempty"`
	Max *int `json:"max,omitempty" yaml:"max,omitempty"`
}

// ExecutionMatch asserts the execution status and optional counters.
type ExecutionMatch struct {
	Status      string `json:"status" yaml:"status"`
	Matches     *Count `json:"matches,omitempty" yaml:"matches,omitempty"`
	Invocations *Count `json:"invocations,omitempty" yaml:"invocations,omitempty"`
	Ignored     *Count `json:"ignored,omitempty" yaml:"ignored,omitempty"`
	Fixed       *Count `json:"fixed,omitempty" yaml:"fixed,omitempty"`
}

// ChangeExpectations compares diagnostics using vacuum's source-aware differ.
type ChangeExpectations struct {
	Added     []DiagnosticMatch `json:"added" yaml:"added"`
	Removed   []DiagnosticMatch `json:"removed" yaml:"removed"`
	NoNew     bool              `json:"noNew,omitempty" yaml:"noNew,omitempty"`
	Unchanged bool              `json:"unchanged,omitempty" yaml:"unchanged,omitempty"`
}

// FixExpectations verifies semantic output, post-fix diagnostics, and idempotence.
// Fixes execute in memory. Input files are never overwritten.
type FixExpectations struct {
	Output string       `json:"output" yaml:"output"`
	Expect Expectations `json:"expect" yaml:"expect"`
}

// RunOptions supplies caller-owned functions and invocation controls.
// Functions must support the motor's concurrency contract; suites run serially.
type RunOptions struct {
	Context          context.Context
	CaseIDs          []string
	SampleLimit      int
	Version          string
	CustomFunctions  map[string]model.RuleFunction
	AutoFixFunctions map[string]model.AutoFixFunction
}

// Report is the versioned machine result. ExitCode uses vacuum's 0/1/2 contract.
type Report struct {
	Version  int          `json:"version"`
	Status   string       `json:"status"`
	ExitCode int          `json:"exitCode"`
	Cases    []CaseResult `json:"cases"`
	Errors   []string     `json:"errors"`
	Manifest Manifest     `json:"manifest"`
}

// CaseResult contains the evidence and assertion failures for one case.
type CaseResult struct {
	Fixed    *Snapshot `json:"fixed,omitempty"`
	ID       string    `json:"id"`
	Status   string    `json:"status"`
	Failures []Failure `json:"failures"`
	Errors   []string  `json:"errors"`
	Before   *Snapshot `json:"before,omitempty"`
	After    *Snapshot `json:"after,omitempty"`
}

// Failure identifies a stable assertion and its expected and actual values.
type Failure struct {
	Assertion string `json:"assertion"`
	Code      string `json:"code"`
	Expected  any    `json:"expected"`
	Actual    any    `json:"actual"`
}

// Snapshot is detached from motor resources before those resources are released.
type Snapshot struct {
	Diagnostics []Diagnostic           `json:"diagnostics"`
	Evidence    []motor.RuleEvidence   `json:"evidence"`
	Rules       map[string]*model.Rule `json:"rules"`
}

// Diagnostic exposes canonical source identity and one-based coordinates.
type Diagnostic struct {
	Rule           string   `json:"rule"`
	Severity       string   `json:"severity"`
	Path           string   `json:"path"`
	Paths          []string `json:"paths,omitempty"`
	Source         string   `json:"source"`
	Line           int      `json:"line"`
	Column         int      `json:"column"`
	Message        string   `json:"message"`
	PathsTruncated bool     `json:"pathsTruncated,omitempty"`
}

// Manifest records the exact local inputs and effective execution settings.
type Manifest struct {
	SuiteSHA256   string         `json:"suiteSha256"`
	CaseIDs       []string       `json:"caseIds"`
	SampleLimit   int            `json:"sampleLimit"`
	VacuumVersion string         `json:"vacuumVersion"`
	Suite         string         `json:"suite"`
	Options       Options        `json:"options"`
	Files         []FileIdentity `json:"files"`
	Reproducible  bool           `json:"reproducible"`
	Limitations   []string       `json:"limitations"`
}

// FileIdentity identifies a dependency by suite-relative path and SHA-256.
type FileIdentity struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}
