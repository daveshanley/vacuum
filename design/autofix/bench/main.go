// Command bench compares lint, repair planning, verified repair and a repeat run.
// It never writes specification files. Allocation totals are process-wide, so
// run it alone and compare the phases within the same invocation.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
)

type sample struct {
	Milliseconds   float64 `json:"milliseconds"`
	AllocatedBytes uint64  `json:"allocatedBytes"`
	Allocations    uint64  `json:"allocations"`
	Fixed          int     `json:"fixed"`
	Errors         int     `json:"errors"`
}
type record struct {
	File    string   `json:"file"`
	Bytes   int      `json:"bytes"`
	Phase   string   `json:"phase"`
	Samples []sample `json:"samples"`
}

func main() {
	corpus := flag.String("corpus", "", "benchmark directory")
	runs := flag.Int("runs", 3, "samples per phase")
	flag.Parse()
	var paths []string
	if *corpus != "" {
		entries, err := os.ReadDir(*corpus)
		must(err)
		for _, entry := range entries {
			ext := filepath.Ext(entry.Name())
			if !entry.IsDir() && (ext == ".yaml" || ext == ".yml" || ext == ".json") {
				paths = append(paths, filepath.Join(*corpus, entry.Name()))
			}
		}
	}
	sort.Strings(paths)
	paths = append(paths, "")
	encoder := json.NewEncoder(os.Stdout)
	for _, path := range paths {
		var data []byte
		name := filepath.Base(path)
		if path == "" {
			name = "synthetic-repair"
			data = []byte(`{"openapi":"3.1.0","info":{"title":"Synthetic","version":"1"},"paths":{},"components":{"schemas":{"Pet":{"type":"string","enum":["a","a","b"]}}}}`)
		} else {
			var err error
			data, err = os.ReadFile(path)
			must(err)
		}
		candidate := data
		for _, phase := range []string{"lint", "plan", "repair", "repeat"} {
			row := record{File: name, Bytes: len(data), Phase: phase}
			for i := 0; i < *runs; i++ {
				source := data
				if phase == "repeat" {
					source = candidate
				}
				e := &motor.RuleSetExecution{Spec: source, SpecFileName: path, SilenceLogs: true, RuleSet: rulesets.BuildDefaultRuleSets().GenerateOpenAPIRecommendedRuleSet()}
				runtime.GC()
				var before, after runtime.MemStats
				runtime.ReadMemStats(&before)
				start := time.Now()
				var result *motor.RuleSetExecutionResult
				switch phase {
				case "lint":
					result = motor.ApplyRulesToRuleSet(e)
				case "plan":
					_, result = motor.RunRepairs(e, nil, nil, false)
				default:
					_, result = motor.RunRepairs(e, nil, nil, true)
				}
				elapsed := time.Since(start)
				runtime.ReadMemStats(&after)
				row.Samples = append(row.Samples, sample{float64(elapsed.Microseconds()) / 1000, after.TotalAlloc - before.TotalAlloc, after.Mallocs - before.Mallocs, len(result.FixedResults), len(result.Errors)})
				if phase == "repair" && len(result.ModifiedSpec) > 0 {
					candidate = append([]byte(nil), result.ModifiedSpec...)
				}
				result.ReleaseOwnedResources()
			}
			must(encoder.Encode(row))
		}
	}
}
func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
