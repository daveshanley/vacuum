package cmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/tui"
)

// dashboardRepairConfig binds review to the same lint settings and the shared
// source-checked writer. A persisted report never grants access to its source.
func dashboardRepairConfig(source string, execution *motor.RuleSetExecution, options *motor.ExecutionOptions, ignored model.IgnoredItems, disabled string) *tui.RepairConfig {
	if execution == nil && disabled == "" {
		disabled = "Open a local specification to review fixes; saved reports are read-only."
	}
	if strings.Contains(source, "://") {
		disabled = "Download the specification and open its local file to review fixes."
	}
	var template motor.RuleSetExecution
	if execution != nil {
		template = *execution
	}
	return &tui.RepairConfig{
		Reload: func(rules *rulesets.RuleSet, currentIgnores model.IgnoredItems) {
			template.RuleSet = rules.CopyForExecution()
			ignored = currentIgnores
		},
		Open: func(snapshot []byte) (*motor.FixReview, error) {
			if disabled != "" {
				return nil, fmt.Errorf("%s", disabled)
			}
			info, err := os.Lstat(source)
			if err != nil {
				return nil, err
			}
			if !info.Mode().IsRegular() {
				return nil, fmt.Errorf("repair source must be a regular local file")
			}
			current, err := os.ReadFile(source)
			if err != nil {
				return nil, err
			}
			if !bytes.Equal(current, snapshot) {
				return nil, fmt.Errorf("source changed; refresh the dashboard before reviewing fixes")
			}
			copy := template
			copy.Spec = append([]byte(nil), snapshot...)
			copy.SpecFileName = source
			copy.RepairIgnores = ignored
			return motor.NewFixReview(&copy, options)
		},
		Write: func(original, candidate []byte) error {
			if disabled != "" {
				return fmt.Errorf("%s", disabled)
			}
			if bytes.Equal(original, candidate) {
				return nil
			}
			return writeRepairFile(source, "", original, candidate)
		},
	}
}
