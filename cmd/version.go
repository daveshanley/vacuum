// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func GetGenerateVersionCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:     "version",
		Short:   "Prints the current version of vacuum",
		Long:    "Prints out the current version of vacuum to the terminal",
		Example: "vacuum version",
		Run: func(cmd *cobra.Command, args []string) {
			pterm.Println(Version)
		},
	}
	return cmd
}
