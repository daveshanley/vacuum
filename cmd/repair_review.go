// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/daveshanley/vacuum/motor/autofix"

	"github.com/daveshanley/vacuum/cmd/internal/repairinput"
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
	output, outputIsFile := writer.(interface{ Fd() uintptr })
	interactive := execution.SpecFileName != "stdin" && term.IsTerminal(int(os.Stdin.Fd())) && outputIsFile && term.IsTerminal(int(output.Fd()))
	return runReviewedCLIRepairWithIO(execution, options, flags, policy, os.Stdin, writer, interactive)
}

func runReviewedCLIRepairWithIO(execution *motor.RuleSetExecution, options *motor.ExecutionOptions, flags *LintFlags, policy *repair.Policy, input io.Reader, writer io.Writer, interactive bool) *motor.RuleSetExecutionResult {
	fail := func(err error) *motor.RuleSetExecutionResult {
		return &motor.RuleSetExecutionResult{RuleSetExecution: execution, Errors: []error{err}}
	}
	if execution.ApplyAutoFixes && len(execution.AutoFixFunctions) != 0 {
		return fail(fmt.Errorf("custom fix callbacks need a reviewed proposal; no files written"))
	}
	quiet := flags.repairWarningsSilenced || flags.SilentFlag && !flags.FormattedOutput
	baselineWarning := func(reason string) *motor.RuleSetExecutionResult {
		baseline := motor.ApplyRulesToRuleSetWithOptions(execution.CopyConfig(), options)
		baseline.RepairWarnings = append(baseline.RepairWarnings, reason)
		return baseline
	}
	if !flags.DryRun {
		if !interactive || quiet {
			return baselineWarning("fixes require visible terminal review; no files written; use --fix --dry-run to preview")
		}
		info, err := validateRepairSource(execution.SpecFileName)
		if err != nil {
			return baselineWarning(err.Error() + "; no files written")
		}
		if flags.FixFileFlag != "" {
			source, err := filepath.Abs(execution.SpecFileName)
			if err != nil {
				return fail(err)
			}
			output, err := filepath.Abs(flags.FixFileFlag)
			if err != nil {
				return fail(err)
			}
			if err := validateRepairOutput(source, output, info, false); err != nil {
				return fail(err)
			}
		}
	}
	review, err := autofix.NewFixReviewWithPolicy(execution, options, policy)
	if err != nil {
		return baselineWarning("repairs declined: " + err.Error())
	}
	defer review.Close()
	available := review.Options()
	var choices []autofix.FixChoice
	for _, option := range available {
		if option.Available && !option.NeedsValue {
			choices = append(choices, autofix.FixChoice{Index: option.Index})
		}
	}
	if len(choices) == 0 {
		return review.TakeBaselineResult()
	}
	if quiet {
		writer = io.Discard
	}
	repairConsoleMu.Lock()
	defer repairConsoleMu.Unlock()
	reader := io.Reader(repairinput.ContextReader{Reader: input})
	if options != nil {
		reader = repairinput.ContextReader{Reader: input, Context: options.Context}
	}
	if interactive && !flags.DryRun && !quiet {
		choices, err = selectCLIRepairs(reader, writer, available)
		if err != nil {
			return fail(err)
		}
		if len(choices) == 0 {
			return review.TakeBaselineResult()
		}
	}
	preview, declined, err := previewCLIRepairs(review, choices, options, writer)
	if err != nil {
		baseline := review.TakeBaselineResult()
		baseline.RepairWarnings = append(baseline.RepairWarnings, fmt.Sprintf("repairs declined: %v", err))
		baseline.RepairWarnings = append(baseline.RepairWarnings, declined...)
		return baseline
	}
	defer preview.Close()
	for _, warning := range declined {
		if err := writeRepairReview(writer, "Warning: "+warning+"\n"); err != nil {
			return fail(err)
		}
	}

	accepted, err := reviewCLIRepair(reader, writer, interactive, flags.DryRun, execution.SpecFileName, execution.Spec, preview)
	if err != nil {
		return fail(err)
	}
	if flags.DryRun || !accepted {
		return review.TakeBaselineResult()
	}
	if options != nil && options.Context != nil && options.Context.Err() != nil {
		return fail(fmt.Errorf("fix review canceled: %w; no files written", options.Context.Err()))
	}
	return preview.TakeResult()
}

// reviewCLIRepair presents the exact verified bytes before reading acceptance.
// It has no write access. A dry run displays the projection without acceptance.
func reviewCLIRepair(input io.Reader, output io.Writer, interactive, dryRun bool, source string, original []byte, preview *autofix.FixPreview) (bool, error) {
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
	clipped := false
	if dryRun {
		if err := writeRepairReview(diffOutput, utils.RepairDiff(original, preview.Candidate)); err != nil {
			return false, fmt.Errorf("cannot display fix review: %w", err)
		}
	} else {
		var err error
		clipped, err = utils.WriteRepairDiffExcerpt(diffOutput, original, preview.Candidate)
		if err != nil {
			return false, fmt.Errorf("cannot display fix review: %w", err)
		}
	}
	if err := diffBuffer.Flush(); err != nil {
		return false, fmt.Errorf("cannot display fix review: %w", err)
	}
	switch preview.Impact.Status {
	case autofix.RepairImpactNoBreaking:
		display.WriteString(console.paint(color.ASCIIGreen, "\n No breaking changes detected.\n"))
	case autofix.RepairImpactBreaking:
		display.WriteString(console.paint(color.ASCIIRed, fmt.Sprintf("\n Breaking changes detected: %d\n", preview.Impact.BreakingChanges)))
	default:
		display.WriteString(console.paint(color.ASCIIYellow, fmt.Sprintf("\n Contract impact unknown: %s\n", repairConsoleText(preview.Impact.Reason))))
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
	mode := repairinput.Apply
	if clipped {
		mode = repairinput.ApplyOrExpand
	}
	answer, canceled, err := repairinput.ReadKeys(input, output, mode, console.paint)
	if err == nil && answer == "d" {
		if err := utils.WriteRepairDiff(diffOutput, original, preview.Candidate); err != nil {
			return false, fmt.Errorf("cannot display full diff: %w", err)
		}
		if err := diffBuffer.Flush(); err != nil {
			return false, fmt.Errorf("cannot display full diff: %w", err)
		}
		_, canceled, err = repairinput.ReadKeys(input, output, repairinput.Apply, console.paint)
	}
	if err != nil {
		return false, fmt.Errorf("fix review was not accepted: %w; no files written", err)
	}
	if canceled {
		if err := writeRepairReview(output, console.paint(color.ASCIIGrey, " Canceled. No files written.\n")); err != nil {
			return false, err
		}
		return false, nil
	}
	return true, nil
}

func selectCLIRepairs(input io.Reader, output io.Writer, options []autofix.FixOption) ([]autofix.FixChoice, error) {
	console := newRepairConsole(output)
	available := make(map[int]bool)
	var all []autofix.FixChoice
	for _, option := range options {
		if !option.Available || option.NeedsValue {
			continue
		}
		available[option.Index+1] = true
		all = append(all, autofix.FixChoice{Index: option.Index})
	}
	if err := writeRepairReview(output, console.options(options)); err != nil {
		return nil, err
	}
	answer, canceled, err := repairinput.ReadKeys(input, output, repairinput.Select, newRepairConsole(output).paint)
	if err != nil {
		return nil, fmt.Errorf("fix selection was not submitted: %w", err)
	}
	if canceled {
		return nil, nil
	}
	answer = strings.TrimSpace(answer)
	if answer == "all" || answer == "" {
		return all, nil
	}
	var selected []autofix.FixChoice
	seen := make(map[int]bool)
	for _, part := range strings.Fields(answer) {
		n, err := strconv.Atoi(part)
		if err != nil || !available[n] || seen[n] {
			return nil, fmt.Errorf("invalid or repeated fix number %q", part)
		}
		seen[n] = true
		selected = append(selected, autofix.FixChoice{Index: n - 1})
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
