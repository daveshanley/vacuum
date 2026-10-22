package harness

import (
	"github.com/pb33f/testify/require"
	"path/filepath"
	"testing"
)

func TestJavaScriptRulesetEvidence(t *testing.T) {
	s, path := testSuite(t)
	dir := filepath.Dir(path)
	fixture(t, "helper.cjs", `module.exports=v=>v.title?undefined:[{message:'missing title'}]`, dir)
	fixture(t, "rules.cjs", `const fn=require('./helper.cjs');module.exports={rules:{'title-required':{given:'$.info',severity:'error',then:{function:fn}}}}`, dir)
	s.Ruleset = "rules.cjs"
	r := Run(s, path, RunOptions{})
	require.Equal(t, 0, r.ExitCode, "%+v %+v", r.Errors, r.Cases)
	require.False(t, r.Manifest.Reproducible)
	found := false
	for _, file := range r.Manifest.Files {
		if file.Path == "helper.cjs" {
			found = true
		}
	}
	require.True(t, found)
	require.Error(t, WriteBundle(r, path, filepath.Join(dir, "bundle")))
	fixture(t, "helper.cjs", `module.exports=()=>{throw Error('failure')}`, dir)
	r = Run(s, path, RunOptions{})
	require.Equal(t, 2, r.ExitCode)
	require.NotEmpty(t, r.Cases[0].Errors)
}
