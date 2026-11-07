// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package repairinput

import (
	"bytes"
	"context"
	"github.com/daveshanley/vacuum/color"
	"github.com/pb33f/testify/require"
	"testing"
)

func TestRepairConsolePromptUsesBlueKeysAndIgnoresNavigation(t *testing.T) {
	paint := func(style, text string) string { return style + text + color.ASCIIReset }
	var output bytes.Buffer
	keys := []string{"up", "home", "3", "backspace", "8", "enter"}
	next := func() (string, error) { key := keys[0]; keys = keys[1:]; return key, nil }
	answer, canceled, err := readRepairPrompt(context.Background(), &output, paint, true, false, next)
	require.NoError(t, err)
	require.False(t, canceled)
	require.Equal(t, "8", answer)
	require.Contains(t, output.String(), color.ASCIIBlue+"enter: review")
	require.Contains(t, output.String(), color.ASCIIBlue+"esc: cancel")
	output.Reset()
	_, canceled, err = readRepairPrompt(context.Background(), &output, paint, false, false, func() (string, error) { return "enter", nil })
	require.NoError(t, err)
	require.False(t, canceled)
	require.Contains(t, output.String(), color.ASCIIBlue+"enter: apply")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err = readRepairPrompt(ctx, &output, paint, false, false, func() (string, error) { t.Fatal("must not read after interruption"); return "", nil })
	require.ErrorIs(t, err, context.Canceled)
}
