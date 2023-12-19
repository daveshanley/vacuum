// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package cmd

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func GetLanguageServerCommand() *cobra.Command {
	return &cobra.Command{
		SilenceErrors: true,
		SilenceUsage:  true,
		Use:           "language-server",
		Short:         "Language server",
		Long:          `Language server for linting an OpenAPI specification in real time`,
		RunE: func(cmd *cobra.Command, args []string) error {
			pterm.Info.Println("starting vacuum language server")
			//return languageserver.NewServer(Version).Run()
			return nil
		},
	}
}
