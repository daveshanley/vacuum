package main

import (
	"os"
	"runtime"
	"runtime/pprof"
	"testing"

	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
)

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
