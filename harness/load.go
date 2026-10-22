package harness

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"time"

	"github.com/pb33f/go-yaml"
)

var caseIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)

// Load reads exactly one YAML or JSON suite and rejects unknown fields.
func Load(path string) (*Suite, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(b)
}

// Parse validates a version 1 suite without reading its dependencies.
func Parse(data []byte) (*Suite, error) {
	var s Suite
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&s); err != nil {
		return nil, err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("suite must contain exactly one document")
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return &s, nil
}

// Validate checks semantic constraints also required of programmatically built suites.
func (s *Suite) Validate() error {
	if s == nil || s.Version != 1 {
		return fmt.Errorf("suite version must be 1")
	}
	if s.Ruleset == "" {
		return fmt.Errorf("suite requires a ruleset")
	}
	if len(s.Cases) == 0 {
		return fmt.Errorf("suite must contain at least one case")
	}
	if s.Options.TimeoutMS < 0 || s.Options.LookupTimeoutMS < 0 || s.Options.RunTimeoutMS < 0 || s.Options.MaxRuleConcurrency < 0 {
		return fmt.Errorf("execution limits must not be negative")
	}
	for _, ms := range []int{s.Options.TimeoutMS, s.Options.LookupTimeoutMS, s.Options.RunTimeoutMS} {
		if int64(ms) > math.MaxInt64/int64(time.Millisecond) {
			return fmt.Errorf("timeout exceeds duration range")
		}
	}
	ids := map[string]bool{}
	for _, c := range s.Cases {
		if !caseIDPattern.MatchString(c.ID) || ids[c.ID] {
			return fmt.Errorf("case ID must be unique and match [A-Za-z0-9][A-Za-z0-9_.-]*: %q", c.ID)
		}
		ids[c.ID] = true
		if c.Input == "" {
			return fmt.Errorf("case %q requires input", c.ID)
		}
		switch c.Format {
		case "", "auto", "json-schema", "fragment":
		default:
			return fmt.Errorf("case %q has unsupported format %q", c.ID, c.Format)
		}
		selected := map[string]bool{}
		for _, r := range c.Rules {
			if r == "" || selected[r] {
				return fmt.Errorf("case %q contains empty or duplicate rule ID", c.ID)
			}
			selected[r] = true
		}
		if err := validateExpect(c.Expect); err != nil {
			return fmt.Errorf("case %q: %w", c.ID, err)
		}
		if c.After != "" && c.Fix != nil {
			return fmt.Errorf("case %q cannot combine after and fix", c.ID)
		}
		if c.After == "" && (c.AfterExpect != nil || c.Change != nil) {
			return fmt.Errorf("case %q requires after", c.ID)
		}
		if c.After != "" && c.AfterExpect == nil && c.Change == nil {
			return fmt.Errorf("case %q requires afterExpect or change", c.ID)
		}
		if c.AfterExpect != nil {
			if err := validateExpect(*c.AfterExpect); err != nil {
				return err
			}
		}
		if c.Fix != nil {
			if c.Fix.Output == "" {
				return fmt.Errorf("fix requires output fixture")
			}
			if err := validateExpect(c.Fix.Expect); err != nil {
				return err
			}
		}
		if c.Change != nil {
			if c.Change.Added == nil && c.Change.Removed == nil && !c.Change.NoNew && !c.Change.Unchanged {
				return fmt.Errorf("change requires an assertion")
			}
			if err := validateMatches(c.Change.Added); err != nil {
				return err
			}
			if err := validateMatches(c.Change.Removed); err != nil {
				return err
			}
		}
	}
	return nil
}
func validateExpect(e Expectations) error {
	if e.Diagnostics == nil && len(e.Execution) == 0 {
		return fmt.Errorf("expect requires diagnostics or execution assertions")
	}
	if err := validateMatches(e.Diagnostics); err != nil {
		return err
	}
	for id, e := range e.Execution {
		if id == "" {
			return fmt.Errorf("execution rule ID must not be empty")
		}
		switch e.Status {
		case "completed", "excluded":
		default:
			return fmt.Errorf("execution status must be completed or excluded")
		}
		for _, c := range []*Count{e.Matches, e.Invocations, e.Ignored, e.Fixed} {
			if c != nil {
				if c.Min == nil && c.Max == nil {
					return fmt.Errorf("count requires min or max")
				}
				if c.Min != nil && *c.Min < 0 || c.Max != nil && *c.Max < 0 {
					return fmt.Errorf("count must not be negative")
				}
				if c.Min != nil && c.Max != nil && *c.Min > *c.Max {
					return fmt.Errorf("count min exceeds max")
				}
			}
		}
	}
	return nil
}
func validateMatches(ms []DiagnosticMatch) error {
	for _, m := range ms {
		if m.Rule == "" {
			return fmt.Errorf("diagnostic assertion requires rule")
		}
		if m.Line < 0 || m.Column < 0 {
			return fmt.Errorf("source coordinates must not be negative")
		}
		switch m.Severity {
		case "", "error", "warn", "info", "hint":
		default:
			return fmt.Errorf("invalid severity %q", m.Severity)
		}
	}
	return nil
}
