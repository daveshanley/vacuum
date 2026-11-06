package cmd

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/daveshanley/vacuum/color"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/testify/require"
)

func TestRepairConsoleTableKeepsStableChoicesAndFitsTerminal(t *testing.T) {
	options := []motor.FixOption{
		{Index: 0, Available: true, Result: &model.RuleFunctionResult{RuleId: "component-description", Path: "$.components.schemas['Cat']"}},
		{Index: 1, Available: false},
		{Index: 7, Available: true, Result: &model.RuleFunctionResult{RuleId: "operation-description", Path: "$.paths['/a/very/long/path/with/many/segments/and/more/segments'].post.requestBody"}},
	}
	for _, width := range []int{60, 100, 160} {
		plain := (repairConsole{width: width}).options(options)
		styled := (repairConsole{width: width, styled: true}).options(options)
		require.Equal(t, plain, ansi.Strip(styled))
		require.Contains(t, styled, color.ASCIIYellow+"  8"+color.ASCIIReset)
		require.Contains(t, styled, color.ASCIIPink+color.ASCIIBold+"Fix")
		// The prompt is prose; table rows must fit without terminal wrapping.
		for _, line := range strings.Split(strings.Split(plain, " Enter fix numbers")[0], "\n") {
			require.LessOrEqual(t, ansi.StringWidth(line), width)
		}
	}
	var out bytes.Buffer
	choices, err := selectCLIRepairs(strings.NewReader("8\n"), &out, options)
	require.NoError(t, err)
	require.Equal(t, []motor.FixChoice{{Index: 7}}, choices)
	require.NotContains(t, out.String(), "\x1b", "redirected review must be plain")
}

func TestRepairConsoleDiffPreservesCompleteSourceAcrossWrites(t *testing.T) {
	original := []byte("name: old\nvalue: " + strings.Repeat("x", 40000) + "\ncontrol: \x1b[2J\n")
	candidate := []byte("name: new\nvalue: " + strings.Repeat("y", 40000) + "\ncontrol: \x1b[2J\n")
	var plain, styled bytes.Buffer
	require.NoError(t, utils.WriteRepairDiff(&plain, original, candidate))
	writer := &repairConsoleDiffWriter{output: &styled, console: repairConsole{styled: true}}
	// Split inside prefixes and source lines, as well as at buffer boundaries.
	for _, chunk := range []int{1, 7, 16384} {
		styled.Reset()
		writer.inLine = false
		for offset := 0; offset < plain.Len(); offset += chunk {
			data := plain.Bytes()[offset:min(offset+chunk, plain.Len())]
			n, err := writer.Write(data)
			require.NoError(t, err)
			require.Equal(t, len(data), n)
		}
		require.Equal(t, plain.String(), ansi.Strip(styled.String()))
		require.Contains(t, styled.String(), color.ASCIIRed+repairDiffRedBackground+ansi.EraseLineRight+"-name: old")
		require.Contains(t, styled.String(), color.ASCIIGreen+repairDiffGreenBackground+ansi.EraseLineRight+"+name: new")
		require.Contains(t, styled.String(), "-name: old"+color.ASCIIReset+"\n")
		require.Contains(t, styled.String(), "+name: new"+color.ASCIIReset+"\n")
		require.Contains(t, styled.String(), " control: \\x1b[2J"+color.ASCIIReset+"\n", "context must not receive a row highlight")
		require.NotContains(t, styled.String(), "\x1b[2J")
	}
	for _, out := range []io.Writer{shortReviewWriter{}, errorWriter{}} {
		w := &repairConsoleDiffWriter{output: out, console: repairConsole{styled: true}}
		_, err := w.Write([]byte("+change\n"))
		require.Error(t, err)
	}
}

func TestRepairConsoleMetadataCannotInjectTerminalControls(t *testing.T) {
	require.Equal(t, "title [2J injected", repairConsoleText("title\x1b[2J\ninjected"))
	require.False(t, newRepairConsole(&bytes.Buffer{}).styled)
}

// A failing diff flush must stop review before the answer is read.
type repairConsoleCountingWriter struct {
	writes    int
	failAfter int
}

func (w *repairConsoleCountingWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.failAfter > 0 && w.writes > w.failAfter {
		return 0, io.ErrClosedPipe
	}
	return len(p), nil
}

func TestRepairConsoleDiffBatchesWritesAndRejectsFlushFailure(t *testing.T) {
	preview := &motor.FixPreview{Candidate: []byte(strings.Repeat("new: value\n", 10000)), Impact: motor.RepairImpact{Status: motor.RepairImpactNoBreaking}}
	output := &repairConsoleCountingWriter{}
	accepted, err := reviewCLIRepair(strings.NewReader("\r"), output, true, false, "api.yaml", []byte("old: value\n"), preview)
	require.NoError(t, err)
	require.True(t, accepted)
	require.Less(t, output.writes, 100, "diff writes must be batched")
	output = &repairConsoleCountingWriter{failAfter: 1}
	accepted, err = reviewCLIRepair(errorReader{}, output, true, false, "api.yaml", nil, &motor.FixPreview{Candidate: []byte("small: diff\n")})
	require.ErrorContains(t, err, "cannot display fix review")
	require.False(t, accepted)
	for _, data := range [][]byte{[]byte("small"), []byte(strings.Repeat("x", 32768))} {
		writer := bufio.NewWriter(repairConsoleCheckedWriter{shortReviewWriter{}})
		_, err = writer.Write(data)
		if err == nil {
			err = writer.Flush()
		}
		require.ErrorIs(t, err, io.ErrShortWrite)
	}
}

func TestRepairConsolePromptUsesBlueKeysAndIgnoresNavigation(t *testing.T) {
	var output bytes.Buffer
	keys := []string{"up", "home", "3", "backspace", "8", "enter"}
	next := func() (string, error) { key := keys[0]; keys = keys[1:]; return key, nil }
	answer, canceled, err := readRepairPrompt(context.Background(), &output, repairConsole{styled: true}, true, false, next)
	require.NoError(t, err)
	require.False(t, canceled)
	require.Equal(t, "8", answer)
	require.Contains(t, output.String(), color.ASCIIBlue+"enter: review")
	require.Contains(t, output.String(), color.ASCIIBlue+"esc: cancel")
	output.Reset()
	_, canceled, err = readRepairPrompt(context.Background(), &output, repairConsole{styled: true}, false, false, func() (string, error) { return "enter", nil })
	require.NoError(t, err)
	require.False(t, canceled)
	require.Contains(t, output.String(), color.ASCIIBlue+"enter: apply")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err = readRepairPrompt(ctx, &output, repairConsole{}, false, false, func() (string, error) { t.Fatal("must not read after interruption"); return "", nil })
	require.ErrorIs(t, err, context.Canceled)
}
