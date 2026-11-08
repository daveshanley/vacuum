// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"os"
)

func ruleContract(cmd *cobra.Command) model.Contract {
	enabled, _ := cmd.Flags().GetBool("openlint")
	if enabled {
		return model.OpenLintV1Draft
	}
	return model.NativeContract
}

func validateOpenLintCommand(cmd *cobra.Command) error {
	if ruleContract(cmd) == model.NativeContract {
		return nil
	}
	switch cmd.Name() {
	case "lint", "report", "spectral-report", "html-report", "dashboard", "language-server":
	default:
		return fmt.Errorf("--openlint is not supported by %s", cmd.CommandPath())
	}
	if fix, _ := cmd.Flags().GetBool("fix"); fix {
		return fmt.Errorf("--openlint cannot be combined with --fix")
	}
	if path, _ := cmd.Flags().GetString("ruleset"); path == "" {
		return fmt.Errorf("--openlint requires --ruleset (-r); supply an OpenLint ruleset")
	}
	return nil
}

func renderOpenLintContract(cmd *cobra.Command) {
	if ruleContract(cmd) == model.NativeContract || cmd.Name() != "lint" || !term.IsTerminal(int(os.Stdout.Fd())) {
		return
	}
	for _, flag := range []string{"silent", "pipeline-output", "github-annotations"} {
		if enabled, _ := cmd.Flags().GetBool(flag); enabled {
			return
		}
	}
	if formats, _ := cmd.Flags().GetStringSlice("format"); len(formats) > 0 {
		return
	}
	tui.RenderInfo("OpenLint contract: %s", model.OpenLintV1Draft)
}
