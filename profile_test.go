package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"testing"

	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
)

// ProfileSpec runs turbo+hard-mode profiling for a single spec and writes
// CPU, heap, and allocs profiles to /tmp/vacuum_turbo_<name>_*.prof
func profileSpec(t *testing.T, name, path string) {
	t.Helper()
	spec, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	rs := rulesets.BuildDefaultRuleSets()
	defaultRS := rs.GenerateOpenAPIRecommendedRuleSet()
	hardRS := rs.GenerateOpenAPIDefaultRuleSet()
	for k, v := range hardRS.Rules {
		defaultRS.Rules[k] = v
	}
	rulesets.FilterRulesForTurbo(defaultRS)

	prefix := fmt.Sprintf("/tmp/vacuum_turbo_%s", name)

	cpuf, _ := os.Create(prefix + "_cpu.prof")
	pprof.StartCPUProfile(cpuf)

	motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
		RuleSet:     defaultRS,
		Spec:        spec,
		AllowLookup: false,
		TurboMode:   true,
	})

	pprof.StopCPUProfile()
	cpuf.Close()

	runtime.GC()

	f, _ := os.Create(prefix + "_heap.prof")
	pprof.WriteHeapProfile(f)
	f.Close()

	af, _ := os.Create(prefix + "_allocs.prof")
	pprof.Lookup("allocs").WriteTo(af, 0)
	af.Close()

	t.Logf("%s: profiles written to %s_*.prof", name, prefix)
}

func TestProfileVacuumTurboHardMode_AllSpecs(t *testing.T) {
	specs := []struct {
		name string
		path string
	}{
		{"petstore", "../demo/speed-test/petstore.yaml"},
		{"mistral", "../demo/speed-test/mistral.yaml"},
		{"neon", "../demo/speed-test/neon.yaml"},
		{"ld", "../demo/speed-test/ld.yaml"},
		{"plaid", "../demo/speed-test/plaid.yml"},
		{"stripe", "../demo/speed-test/stripe.yaml"},
	}

	for _, s := range specs {
		t.Run(s.name, func(t *testing.T) {
			profileSpec(t, s.name, s.path)
		})
	}
}

func TestProfileVacuumTurboCPU(t *testing.T) {
	spec, err := os.ReadFile("../demo/speed-test/stripe.yaml")
	if err != nil {
		t.Fatal(err)
	}

	rs := rulesets.BuildDefaultRuleSets()
	defaultRS := rs.GenerateOpenAPIRecommendedRuleSet()

	cpuf, _ := os.Create("/tmp/vacuum_turbo_cpu.prof")
	pprof.StartCPUProfile(cpuf)

	motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
		RuleSet:     defaultRS,
		Spec:        spec,
		AllowLookup: false,
		TurboMode:   true,
	})

	pprof.StopCPUProfile()
	cpuf.Close()

	runtime.GC()

	f, _ := os.Create("/tmp/vacuum_turbo_heap.prof")
	pprof.WriteHeapProfile(f)
	f.Close()

	af, _ := os.Create("/tmp/vacuum_turbo_allocs.prof")
	pprof.Lookup("allocs").WriteTo(af, 0)
	af.Close()
}
