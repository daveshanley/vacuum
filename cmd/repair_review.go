package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/daveshanley/vacuum/color"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/repair"
	"github.com/daveshanley/vacuum/utils"
	"golang.org/x/term"
)

// Concurrent file linting shares the terminal. Keep each diff and its answer
// together so an answer cannot approve a different file's candidate.
var repairConsoleMu sync.Mutex

func runReviewedCLIRepair(execution *motor.RuleSetExecution, options *motor.ExecutionOptions, flags *LintFlags, policy *repair.Policy) *motor.RuleSetExecutionResult {
	writer := flags.RepairWarningWriter
	if writer == nil {
		writer = os.Stderr
	}
	interactive := execution.SpecFileName != "stdin" && term.IsTerminal(int(os.Stdin.Fd()))
	return runReviewedCLIRepairWithIO(execution, options, flags, policy, os.Stdin, writer, interactive)
}

func runReviewedCLIRepairWithIO(execution *motor.RuleSetExecution, options *motor.ExecutionOptions, flags *LintFlags, policy *repair.Policy, input io.Reader, writer io.Writer, interactive bool) *motor.RuleSetExecutionResult {
	fail := func(err error) *motor.RuleSetExecutionResult {
		return &motor.RuleSetExecutionResult{RuleSetExecution: execution, Errors: []error{err}}
	}
	if execution.ApplyAutoFixes && len(execution.AutoFixFunctions) != 0 {
		return fail(fmt.Errorf("custom fix callbacks need a reviewed proposal; no files written"))
	}
	review, err := motor.NewFixReviewWithPolicy(execution, options, policy)
	if err != nil {
		return fail(err)
	}
	defer review.Close()
	available := review.Options()
	var choices []motor.FixChoice
	for _, option := range available {
		if option.Available {
			choices = append(choices, motor.FixChoice{Index: option.Index})
		}
	}
	if len(choices) == 0 {
		return review.TakeBaselineResult()
	}
	quiet := flags.repairWarningsSilenced || flags.SilentFlag && !flags.FormattedOutput
	if quiet {
		writer = io.Discard
	}
	repairConsoleMu.Lock()
	defer repairConsoleMu.Unlock()
	reader := bufio.NewReader(input)
	if interactive && !flags.DryRun && !quiet {
		choices, err = selectCLIRepairs(reader, writer, available)
		if err != nil {
			return fail(err)
		}
		if len(choices) == 0 {
			return review.TakeBaselineResult()
		}
	}
	preview, err := review.Preview(choices)
	if err != nil {
		baseline := review.TakeBaselineResult()
		baseline.RepairWarnings = append(baseline.RepairWarnings, fmt.Sprintf("repairs declined: %v; use dashboard to select a smaller set", err))
		return baseline
	}
	defer preview.Close()
	if quiet && !flags.DryRun {
		return fail(fmt.Errorf("fixes require visible review; use --fix --dry-run with --silent; no files written"))
	}
	accepted, err := reviewCLIRepair(reader, writer, interactive, flags.DryRun, execution.SpecFileName, execution.Spec, preview)
	if err != nil {
		return fail(err)
	}
	if !accepted && !flags.DryRun {
		return review.TakeBaselineResult()
	}
	if options != nil && options.Context != nil && options.Context.Err() != nil {
		return fail(fmt.Errorf("fix review canceled: %w; no files written", options.Context.Err()))
	}
	return preview.TakeResult()
}

// reviewCLIRepair presents the exact verified bytes before reading acceptance.
// It has no write access. A dry run returns projected results without acceptance.
func reviewCLIRepair(input io.Reader, output io.Writer, interactive, dryRun bool, source string, original []byte, preview *motor.FixPreview) (bool, error) {
	if preview == nil {
		return false, fmt.Errorf("verified fix preview is unavailable")
	}
	console := newRepairConsole(output)
	var display strings.Builder
	display.WriteString(console.heading("Review fixes: " + source))
	display.WriteString(console.paint(color.ASCIIYellow, fmt.Sprintf(" %d fixes; %d findings remain.", preview.Fixed, len(preview.Results))))
	display.WriteString(console.paint(color.ASCIIGrey, " No files written.\n\n"))
	if err := writeRepairReview(output, display.String()); err != nil {
		return false, fmt.Errorf("cannot display fix review: %w", err)
	}
	display.Reset()
	diffBuffer := bufio.NewWriter(repairConsoleCheckedWriter{output})
	diffOutput := &repairConsoleDiffWriter{output: diffBuffer, console: console}
	if dryRun {
		if err := writeRepairReview(diffOutput, utils.RepairDiff(original, preview.Candidate)); err != nil {
			return false, fmt.Errorf("cannot display fix review: %w", err)
		}
	} else if err := utils.WriteRepairDiff(diffOutput, original, preview.Candidate); err != nil {
		return false, fmt.Errorf("cannot display fix review: %w", err)
	}
	if err := diffBuffer.Flush(); err != nil {
		return false, fmt.Errorf("cannot display fix review: %w", err)
	}
	phrase := "yes"
	switch preview.Impact.Status {
	case motor.RepairImpactNoBreaking:
		display.WriteString(console.paint(color.ASCIIGreen, "\n No breaking changes detected.\n"))
	case motor.RepairImpactBreaking:
		display.WriteString(console.paint(color.ASCIIRed, fmt.Sprintf("\n Breaking changes detected: %d\n", preview.Impact.BreakingChanges)))
		phrase = "accept breaking changes"
	default:
		display.WriteString(console.paint(color.ASCIIYellow, fmt.Sprintf("\n Contract impact unknown: %s\n", repairConsoleText(preview.Impact.Reason))))
		phrase = "accept unknown impact"
	}
	for _, change := range preview.Impact.Changes {
		if change == nil || !change.Breaking {
			continue
		}
		fmt.Fprintf(&display, "  %s: %q -> %q", repairConsoleText(change.Property), change.Original, change.New)
		if change.Context != nil && change.Context.OriginalLine != nil {
			fmt.Fprintf(&display, " (original line %d)", *change.Context.OriginalLine)
		}
		display.WriteByte('\n')
	}
	if dryRun {
		display.WriteString(console.paint(color.ASCIIGrey, " Preview only. No files written.\n"))
	} else if interactive {
		display.WriteString(console.paint(color.ASCIIPink, " "+strings.Repeat("─", max(1, console.width-2))+"\n"))
		display.WriteString(console.paint(color.ASCIIGrey, fmt.Sprintf(" Type %q to apply these changes | enter: cancel\n", phrase)))
		display.WriteString(console.paint(color.ASCIIYellow, " > "))
	}
	if err := writeRepairReview(output, display.String()); err != nil {
		return false, fmt.Errorf("cannot display fix review: %w", err)
	}
	if dryRun {
		return false, nil
	}
	if !interactive {
		return false, fmt.Errorf("fixes require terminal review; use --fix --dry-run to preview or dashboard to review and apply; no files written")
	}
	answer, err := readRepairAnswer(input)
	if err != nil {
		return false, fmt.Errorf("fix review was not accepted: %w; no files written", err)
	}
	if strings.TrimSpace(answer) != phrase {
		if err := writeRepairReview(output, console.paint(color.ASCIIGrey, " Canceled. No files written.\n")); err != nil {
			return false, err
		}
		return false, nil
	}
	return true, nil
}

func readRepairAnswer(input io.Reader) (string, error) {
	reader, ok := input.(*bufio.Reader)
	if !ok {
		reader = bufio.NewReader(input)
	}
	var answer strings.Builder
	for answer.Len() < 1024 {
		b, err := reader.ReadByte()
		if err != nil {
			return "", err
		}
		if b == '\n' {
			return answer.String(), nil
		}
		answer.WriteByte(b)
	}
	return "", fmt.Errorf("review answer exceeds 1024 bytes")
}

func selectCLIRepairs(input io.Reader, output io.Writer, options []motor.FixOption) ([]motor.FixChoice, error) {
	console := newRepairConsole(output)
	available := make(map[int]bool)
	var all []motor.FixChoice
	for _, option := range options {
		if !option.Available {
			continue
		}
		available[option.Index+1] = true
		all = append(all, motor.FixChoice{Index: option.Index})
	}
	if err := writeRepairReview(output, console.options(options)); err != nil {
		return nil, err
	}
	answer, err := readRepairAnswer(input)
	if err != nil {
		return nil, fmt.Errorf("fix selection was not submitted: %w", err)
	}
	answer = strings.TrimSpace(answer)
	if answer == "all" {
		return all, nil
	}
	var selected []motor.FixChoice
	seen := make(map[int]bool)
	for _, part := range strings.Fields(answer) {
		n, err := strconv.Atoi(part)
		if err != nil || !available[n] || seen[n] {
			return nil, fmt.Errorf("invalid or repeated fix number %q", part)
		}
		seen[n] = true
		selected = append(selected, motor.FixChoice{Index: n - 1})
	}
	return selected, nil
}

func writeRepairReview(output io.Writer, text string) error {
	n, err := io.WriteString(output, text)
	if err == nil && n != len(text) {
		return io.ErrShortWrite
	}
	return err
}
