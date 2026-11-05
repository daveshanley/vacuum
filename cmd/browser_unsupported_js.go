//go:build js

// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"

	"github.com/daveshanley/vacuum/utils"
	wcModel "github.com/pb33f/libopenapi/what-changed/model"
	"github.com/spf13/cobra"
)

// The browser build leaves out the printing press and the change renderer: both
// depend on SQLite, which cannot be compiled to js/wasm.

// BrowserInstallURL is where browser users are sent for the full CLI.
const BrowserInstallURL = "https://quobix.com/vacuum/installing/"

func GetDocsCommand() *cobra.Command {
	return &cobra.Command{
		Use:    "docs",
		Short:  "Generate API documentation (full CLI only)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("vacuum docs is not available in the browser; install vacuum to use it: %s", BrowserInstallURL)
		},
	}
}

func renderComparisonModeSummary(_ *utils.ChangeResult, _ *wcModel.DocumentChanges, _ bool, _ bool) {}
