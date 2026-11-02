// Command check verifies that the autofix audit covers the current built-in rules.
// Run it from the repository root with go run ./design/autofix/check.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
)

func main() {
	if err := check(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func check() error {
	content, err := os.ReadFile("design/autofix/inventory.json")
	if err != nil {
		return err
	}
	var inventory struct {
		Rules []struct {
			Family         string         `json:"family"`
			ID             string         `json:"id"`
			Classification string         `json:"classification"`
			Repair         string         `json:"repair"`
			Guard          string         `json:"guard"`
			Test           string         `json:"test"`
			Sources        []string       `json:"sources"`
			Rule           map[string]any `json:"rule"`
		} `json:"rules"`
	}
	if err := json.Unmarshal(content, &inventory); err != nil {
		return err
	}
	families := map[string]map[string]*model.Rule{
		"openapi": rulesets.GetAllBuiltInRules(), "owasp": rulesets.GetAllOWASPRules(),
		"asyncapi": rulesets.GetAllAsyncAPIRules(), "jsonschema": rulesets.GetJSONSchemaRecommendedRules(),
		"arazzo": rulesets.GetAllArazzoRules(),
	}
	seen := make(map[string]bool)
	counts := make(map[string]int)
	for _, row := range inventory.Rules {
		key := row.Family + "/" + row.ID
		if seen[key] {
			return fmt.Errorf("duplicate audit row %s", key)
		}
		seen[key] = true
		current := families[row.Family][row.ID]
		if current == nil {
			return fmt.Errorf("unknown rule %s", key)
		}
		if row.Repair == "" || row.Guard == "" || row.Test == "" || len(row.Sources) == 0 {
			return fmt.Errorf("incomplete audit row %s", key)
		}
		switch row.Classification {
		case "A", "P", "R", "I", "O":
		default:
			return fmt.Errorf("invalid class for %s", key)
		}
		for _, source := range row.Sources {
			if _, err := os.Stat(source); err != nil {
				return err
			}
		}
		// Normalize both values through JSON to compare the public ruleset representation.
		actual, err := json.Marshal(current)
		if err != nil {
			return err
		}
		var normalized map[string]any
		if err := json.Unmarshal(actual, &normalized); err != nil {
			return err
		}
		actual, err = json.Marshal(normalized)
		if err != nil {
			return err
		}
		expected, err := json.Marshal(row.Rule)
		if err != nil {
			return err
		}
		if !bytes.Equal(actual, expected) {
			return fmt.Errorf("rule changed since audit: %s", key)
		}
		counts[row.Classification]++
	}
	var missing []string
	for family, rules := range families {
		for id := range rules {
			if !seen[family+"/"+id] {
				missing = append(missing, family+"/"+id)
			}
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		return fmt.Errorf("missing rules: %v", missing)
	}
	fmt.Printf("Verified %d built-in rules: A=%d P=%d R=%d I=%d O=%d\n", len(seen), counts["A"], counts["P"], counts["R"], counts["I"], counts["O"])
	return nil
}
