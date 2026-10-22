package harness

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/pb33f/testify/require"
)

func TestAutoFixRejectsTrailingExpectedDocument(t *testing.T) {
	s, path := testSuite(t)
	fixture(t, "fixed.yaml", good+"---\nother: document\n", filepath.Dir(path))
	s.Cases[0].Fix = &FixExpectations{Output: "fixed.yaml", Expect: Expectations{Diagnostics: []DiagnosticMatch{}}}
	r := Run(s, path, RunOptions{})
	require.Equal(t, 2, r.ExitCode, "%+v", r.Cases)
}

func TestDiagnosticPathsRoundTripRetainsEmptyAssertion(t *testing.T) {
	s, path := testSuite(t)
	s.Cases[0].Input = "bad.yaml"
	s.Cases[0].Expect.Diagnostics = []DiagnosticMatch{{Rule: "title-required", Paths: []string{}}}
	encoded, err := json.Marshal(s)
	require.NoError(t, err)
	decoded, err := Parse(encoded)
	require.NoError(t, err)
	require.NotNil(t, decoded.Cases[0].Expect.Diagnostics[0].Paths)
	require.Equal(t, Run(s, path, RunOptions{}).ExitCode, Run(decoded, path, RunOptions{}).ExitCode)
}

func TestFixtureRejectsTrailingDocument(t *testing.T) {
	s, path := testSuite(t)
	fixture(t, "good.yaml", good+"---\nopenapi: broken\n", filepath.Dir(path))
	require.Equal(t, 2, Run(s, path, RunOptions{}).ExitCode)
}
