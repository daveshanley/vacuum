package rulesets

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/testify/require"
	"sync"
	"testing"
)

func TestSuppliedRulesetsDoNotChangeBuiltins(t *testing.T) {
	for _, alias := range []string{"vacuum:oas", "vacuum:json-schema", "vacuum:asyncapi", "vacuum:all"} {
		for _, mode := range []string{"all", "recommended"} {
			if alias == "vacuum:all" && mode == "recommended" {
				continue
			}
			t.Run(alias+"/"+mode, func(t *testing.T) {
				sets := BuildDefaultRuleSets()
				fresh := func() *RuleSet {
					supplied, err := CreateRuleSetFromData([]byte(fmt.Sprintf("extends: [[%s, %s]]", alias, mode)))
					require.NoError(t, err)
					return sets.GenerateRuleSetFromSuppliedRuleSet(supplied)
				}
				initial := fresh()
				require.NotEmpty(t, initial.Rules)
				var id string
				for key := range initial.Rules {
					id = key
					break
				}
				severity := initial.Rules[id].Severity
				overrideSeverity := model.SeverityHint
				if severity == overrideSeverity {
					overrideSeverity = model.SeverityError
				}
				for _, override := range []any{false, "off", overrideSeverity} {
					supplied, err := CreateRuleSetFromData([]byte(fmt.Sprintf("extends: [[%s, %s]]", alias, mode)))
					require.NoError(t, err)
					supplied.RuleDefinitions = map[string]any{id: override}
					changed := sets.GenerateRuleSetFromSuppliedRuleSet(supplied)
					if override == overrideSeverity {
						require.Equal(t, overrideSeverity, changed.Rules[id].Severity)
					} else {
						require.NotContains(t, changed.Rules, id)
					}
					restored := fresh()
					require.Len(t, restored.Rules, len(initial.Rules))
					require.Contains(t, restored.Rules, id)
					require.Equal(t, severity, restored.Rules[id].Severity)
					require.Equal(t, severity, initial.Rules[id].Severity)
				}
			})
		}
	}
}

func TestCombinedRulesetsDoNotPolluteOpenAPIOnConcurrentReload(t *testing.T) {
	sets := BuildDefaultRuleSets()
	baseline := sets.GenerateOpenAPIDefaultRuleSet()
	count, description, uri := len(baseline.Rules), baseline.Description, baseline.DocumentationURI
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Go(func() {
			supplied := &RuleSet{Extends: VacuumAllRulesets, RuleDefinitions: map[string]any{"info-contact": "off"}}
			result := sets.GenerateRuleSetFromSuppliedRuleSet(supplied)
			require.NoError(t, result.LoadError())
			require.Greater(t, len(result.Rules), count)
		})
	}
	wg.Wait()
	require.Len(t, baseline.Rules, count)
	require.Equal(t, description, baseline.Description)
	require.Equal(t, uri, baseline.DocumentationURI)
	for name := range GetAllOWASPRules() {
		require.NotContains(t, baseline.Rules, name)
	}
}
