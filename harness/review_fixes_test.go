package harness

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepairEvidenceAndOutputAcrossFormats(t *testing.T) {
	for _, spec := range []string{good,
		"asyncapi: 3.0.0\ninfo: {title: Test, version: '1'}\nchannels: {}\n",
		"arazzo: 1.0.1\ninfo: {title: Test, version: '1'}\nsourceDescriptions: []\nworkflows: []\n"} {
		t.Run(strings.Split(spec, ":")[0], func(t *testing.T) {
			s, path := testSuite(t)
			dir := filepath.Dir(path)
			fixture(t, "good.yaml", spec, dir)
			fixture(t, "fixed.yaml", strings.Replace(spec, "title: Test", "title: ''", 1), dir)
			formats := ""
			if strings.HasPrefix(spec, "asyncapi:") {
				formats = "    formats: [asyncapi3]\n"
			}
			fixture(t, "rules.yaml", "rules:\n  title-required:\n"+formats+"    given: $.info.title\n    severity: error\n    autoFixFunction: title\n    then: {function: falsy}\n", dir)
			one := 1
			s.Cases[0].Expect = Expectations{Diagnostics: []DiagnosticMatch{{Rule: "title-required"}}}
			s.Cases[0].Fix = &FixExpectations{Output: "fixed.yaml", Expect: Expectations{Diagnostics: []DiagnosticMatch{}, Execution: map[string]ExecutionMatch{"title-required": {Status: "completed", Fixed: &Count{Min: &one, Max: &one}}}}}
			opts := RunOptions{AutoFixFunctions: map[string]model.AutoFixFunction{"title": func(n, doc *yaml.Node, ctx *model.RuleFunctionContext) (*yaml.Node, error) {
				n.Value = ""
				return n, nil
			}}}
			report := Run(s, path, opts)
			require.Equal(t, 0, report.ExitCode, fmt.Sprintf("%+v", report.Cases))
			require.Equal(t, 1, report.Cases[0].Fixed.Evidence[0].Fixed)
			require.Equal(t, 0, report.Cases[0].After.Evidence[0].Fixed)
			require.Equal(t, 0, report.Cases[0].SecondFix.Evidence[0].Fixed)
			s.Cases[0].Fix.Expect.Execution = map[string]ExecutionMatch{"typo": {Status: "completed"}}
			report = Run(s, path, opts)
			require.Equal(t, 2, report.ExitCode)
			require.Contains(t, report.Cases[0].Errors[0].Message, "unselected rule")
		})
	}
}
