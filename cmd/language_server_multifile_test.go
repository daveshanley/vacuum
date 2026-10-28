// Copyright 2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package cmd

import (
	"path/filepath"
	"testing"

	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/testify/require"
)

func TestLanguageServerCommandRootSpecification(t *testing.T) {
	original := runLanguageServer
	t.Cleanup(func() { runLanguageServer = original })
	absoluteRoot := filepath.Join(t.TempDir(), "API spec.yaml")
	relativeRoot, err := filepath.Abs(filepath.Join("test_data", "openapi.yaml"))
	require.NoError(t, err)

	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "independent documents"},
		{name: "relative root", args: []string{filepath.Join("test_data", "openapi.yaml")}, want: relativeRoot},
		{name: "absolute root", args: []string{absoluteRoot}, want: absoluteRoot},
		{name: "HTTPS root", args: []string{"https://example.com/apis/root.yaml?revision=1"}, want: "https://example.com/apis/root.yaml?revision=1"},
		{name: "HTTP root", args: []string{"http://example.com/root.yaml"}, want: "http://example.com/root.yaml"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			runLanguageServer = func(_ string, request *utils.LintFileRequest, _ *motor.ExecutionOptions) error {
				called = true
				require.Equal(t, tc.want, request.MainSpecPath)
				return nil
			}
			command := GetLanguageServerCommand()
			registerPersistentFlags(command)
			command.SetArgs(tc.args)
			require.NoError(t, command.Execute())
			require.True(t, called)
		})
	}
}

func TestLanguageServerCommandRejectsMultipleRoots(t *testing.T) {
	original := runLanguageServer
	t.Cleanup(func() { runLanguageServer = original })
	called := false
	runLanguageServer = func(_ string, _ *utils.LintFileRequest, _ *motor.ExecutionOptions) error {
		called = true
		return nil
	}
	command := GetLanguageServerCommand()
	registerPersistentFlags(command)
	command.SetArgs([]string{"first.yaml", "second.yaml"})
	require.ErrorContains(t, command.Execute(), "accepts at most 1 arg(s)")
	require.False(t, called)
}
