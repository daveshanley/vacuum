// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/spf13/cobra"
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
	return nil
}
