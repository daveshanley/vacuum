// Copyright 2024 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT
// https://pb33f.io

package cmd

import (
	languageserver "github.com/daveshanley/vacuum/language-server"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"io"
	"log/slog"
)

func GetLanguageServerCommand() *cobra.Command {
	return &cobra.Command{
		SilenceErrors: true,
		Use:           "language-server",
		Short:         "Run a fully compliant LSP server for OpenAPI linting (Language Server Protocol)",
		Long: `Provides a fully compliant LSP backend for OpenAPI linting and validation. Connect up your favorite
IDE and start linting your OpenAPI documents in real-time.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// setup logging to be discarded, it will invalidate the LSP protocol
			handler := pterm.NewSlogHandler(&pterm.Logger{
				Writer: io.Discard,
			})
			logger := slog.New(handler)
			return languageserver.NewServer(Version, logger).Run()
		},
	}
}
