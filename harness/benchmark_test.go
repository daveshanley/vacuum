package harness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
)

// BenchmarkSuite measures complete library runs, including fixture validation,
// ruleset loading, reference resolution and report construction. CLI process
// startup and JSON output encoding are intentionally outside this measurement.
func BenchmarkSuite(b *testing.B) {
	for _, kind := range []string{"lint", "references", "javascript", "autofix"} {
		for _, cases := range []int{10, 100} {
			b.Run(fmt.Sprintf("%s/cases%d", kind, cases), func(b *testing.B) {
				path, opts, size := benchmarkSuite(b, kind, cases)
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					report := RunFile(path, opts)
					if report.ExitCode != 0 {
						b.Fatalf("benchmark suite failed: %+v %+v", report.Errors, report.Cases)
					}
				}
				b.ReportMetric(float64(size), "fixture-B")
				b.ReportMetric(float64(cases), "cases/op")
			})
		}
	}
}

func benchmarkSuite(b *testing.B, kind string, count int) (string, RunOptions, int) {
	b.Helper()
	dir := b.TempDir()
	write := func(name string, data []byte) {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			b.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0600); err != nil {
			b.Fatal(err)
		}
	}
	var spec strings.Builder
	spec.WriteString(good + "components:\n  schemas:\n")
	for i := 0; i < 50; i++ {
		fmt.Fprintf(&spec, "    Item%d:\n      type: object\n      properties:\n        name: {type: string}\n        count: {type: integer}\n", i)
	}
	if kind == "references" {
		for i := 0; i < 8; i++ {
			fmt.Fprintf(&spec, "    External%d: {$ref: './ref%d.yaml'}\n", i, i)
			write(fmt.Sprintf("ref%d.yaml", i), []byte("type: object\nproperties:\n  value: {type: string}\n"))
		}
	}
	write("input.yaml", []byte(spec.String()))
	rule := testRules
	expected := Expectations{Diagnostics: []DiagnosticMatch{}}
	suite := Suite{Version: 1, Ruleset: "rules.yaml"}
	opts := RunOptions{}
	if kind == "javascript" {
		suite.Functions = "functions"
		write("functions/check.js", []byte(`function getSchema(){return {name:"check"};} function runRule(input){return [];}`))
		rule = "rules:\n  title-required:\n    given: $.info\n    then: {function: check}\n"
	}
	if kind == "autofix" {
		rule = "rules:\n  title-required:\n    given: $.info.title\n    autoFixFunction: title\n    then: {function: falsy}\n"
		expected.Diagnostics = []DiagnosticMatch{{Rule: "title-required"}}
		write("fixed.yaml", []byte(strings.Replace(spec.String(), "title: Test", "title: ''", 1)))
		opts.AutoFixFunctions = map[string]model.AutoFixFunction{"title": func(node, doc *yaml.Node, context *model.RuleFunctionContext) (*yaml.Node, error) {
			node.Value = ""
			return node, nil
		}}
	}
	write("rules.yaml", []byte(rule))
	for i := 0; i < count; i++ {
		c := Case{ID: fmt.Sprintf("case-%d", i), Input: "input.yaml", Expect: expected}
		if kind == "autofix" {
			c.Fix = &FixExpectations{Output: "fixed.yaml", Expect: Expectations{Diagnostics: []DiagnosticMatch{}}}
		}
		suite.Cases = append(suite.Cases, c)
	}
	data, err := json.Marshal(suite)
	if err != nil {
		b.Fatal(err)
	}
	write("suite.json", data)
	return filepath.Join(dir, "suite.json"), opts, spec.Len()
}
