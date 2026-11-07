// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

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
	template := execution.CopyConfig()
	var configMu sync.RWMutex
	return &tui.RepairConfig{
		SourcePath:     source,
		DisabledReason: disabled,
		Reload: func(rules *rulesets.RuleSet, currentIgnores model.IgnoredItems) {
			next := rules.CopyForExecution()
			configMu.Lock()
			template.RuleSet = next
			ignored = currentIgnores
			configMu.Unlock()
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
			configMu.RLock()
			next := template.CopyConfig()
			next.RepairIgnores = ignored
			configMu.RUnlock()
			next.Spec = append([]byte(nil), snapshot...)
			next.SpecFileName = source
			return motor.NewFixReview(next, options)
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
		PreviewOutput: func(review *motor.FixReview, choices []motor.FixChoice, save tui.RepairSaveOptions) (*motor.FixPreview, error) {
			if disabled != "" {
				return nil, fmt.Errorf("%s", disabled)
			}
			output, err := dashboardRepairOutputPath(source, save.OutputPath)
			if err != nil {
				return nil, err
			}
			format := ""
			if output != "" {
				switch strings.ToLower(filepath.Ext(output)) {
				case ".yaml", ".yml":
					format = "yaml"
				case ".json":
					format = "json"
				}
			}
			return review.PreviewOutput(choices, motor.FixOutputOptions{OutputPath: output, Format: format, BundleMode: save.BundleMode})
		},
		Save: func(original, candidate []byte, save tui.RepairSaveOptions) error {
			if disabled != "" {
				return fmt.Errorf("%s", disabled)
			}
			output, err := dashboardRepairOutputPath(source, save.OutputPath)
			if err != nil {
				return err
			}
			if output != "" {
				return writeNewRepairFile(source, output, original, candidate)
			}
			return writeRepairFile(source, "", original, candidate)
		},
	}
}

// Relative copy names belong beside the source, not in the process directory.
// The writer repeats these checks and enforces no-clobber publication at save.
func dashboardRepairOutputPath(source, output string) (string, error) {
	if output == "" {
		return "", nil
	}
	if strings.TrimSpace(output) == "" {
		return "", fmt.Errorf("enter a file name for the new specification")
	}
	absoluteSource, err := filepath.Abs(source)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(output) {
		output = filepath.Join(filepath.Dir(absoluteSource), output)
	}
	output = filepath.Clean(output)
	if output == absoluteSource {
		return "", fmt.Errorf("new file must have a different path from the source")
	}
	if _, err = os.Lstat(output); err == nil {
		return "", fmt.Errorf("new file already exists: %s", output)
	} else if !os.IsNotExist(err) {
		return "", err
	}
	parent, err := os.Stat(filepath.Dir(output))
	if err != nil {
		return "", fmt.Errorf("new file directory: %w", err)
	}
	if !parent.IsDir() {
		return "", fmt.Errorf("new file parent must be a directory")
	}
	return output, nil
}
