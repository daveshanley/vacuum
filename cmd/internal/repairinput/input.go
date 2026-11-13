// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package repairinput

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/daveshanley/vacuum/color"
	"golang.org/x/term"
)

// ContextReader carries command cancellation through successive prompts.
type ContextReader struct {
	io.Reader
	Context context.Context
}

// Prompt identifies which keys the review can accept.
type Prompt uint8

const (
	Apply Prompt = iota
	Select
	ApplyOrExpand
)

// ReadKeys shares the dashboard's key decoder on real terminals. Only
// the prompt owns raw mode; each review starts with fresh keyboard input.
func ReadKeys(input io.Reader, output io.Writer, mode Prompt, paint func(string, string) string) (answer string, canceled bool, err error) {
	ctx := context.Background()
	if wrapped, ok := input.(ContextReader); ok {
		input = wrapped.Reader
		if wrapped.Context != nil {
			ctx = wrapped.Context
		}
	}
	if file, ok := input.(interface{ Fd() uintptr }); ok && term.IsTerminal(int(file.Fd())) {
		ctx, stop := signal.NotifyContext(ctx, repairPromptSignals()...)
		defer stop()
		state, rawErr := term.MakeRaw(int(file.Fd()))
		if rawErr != nil {
			return "", false, rawErr
		}
		defer func() {
			restoreErr := term.Restore(int(file.Fd()), state)
			if err == nil {
				err = restoreErr
			}
			newlineErr := writeText(output, "\n")
			if err == nil {
				err = newlineErr
			}
		}()
		reader, readerErr := uv.NewCancelReader(input)
		if readerErr != nil {
			return "", false, readerErr
		}
		if err = prepareRepairPrompt(int(file.Fd())); err != nil {
			reader.Close()
			return "", false, err
		}
		eventc := make(chan uv.Event)
		done := make(chan struct{})
		readCtx, cancel := context.WithCancel(ctx)
		var readErr error
		go func() {
			defer close(done)
			readErr = uv.NewTerminalReader(reader, os.Getenv("TERM")).StreamEvents(readCtx, eventc)
		}()
		defer func() {
			cancel()
			reader.Cancel()
			// Drain decoder output while it stops, including any queued pasted keys.
			for {
				select {
				case <-eventc:
				case <-done:
					closeErr := reader.Close()
					if err == nil {
						err = closeErr
					}
					return
				}
			}
		}()
		return readRepairPrompt(ctx, output, paint, mode, true, func() (string, error) {
			for {
				select {
				case <-ctx.Done():
					return "", ctx.Err()
				case <-done:
					if readErr != nil {
						return "", readErr
					}
					return "", io.EOF
				case event := <-eventc:
					if key, ok := event.(uv.KeyPressEvent); ok {
						return key.String(), nil
					}
				}
			}
		})
	}
	// Non-terminal readers are used by command tests; never prefetch a later answer.
	return readRepairPrompt(ctx, output, paint, mode, false, func() (string, error) {
		var key [1]byte
		if _, err := io.ReadFull(input, key[:]); err != nil {
			return "", err
		}
		switch key[0] {
		case '\r', '\n':
			return "enter", nil
		case 27:
			return "esc", nil
		case 3:
			return "ctrl+c", nil
		case 4:
			return "ctrl+d", nil
		case 8, 127:
			return "backspace", nil
		case ' ':
			return "space", nil
		}
		return string(key[:]), nil
	})
}

func readRepairPrompt(ctx context.Context, output io.Writer, paint func(string, string) string, mode Prompt, echo bool, next func() (string, error)) (string, bool, error) {
	selection := mode == Select
	action := "enter: apply"
	if selection {
		action = "enter: review"
	}
	prompt := " " + paint(color.ASCIIBlue, action) + paint(color.ASCIIGrey, " | ") + paint(color.ASCIIBlue, "esc: cancel") + "\n"
	if mode == ApplyOrExpand {
		prompt = prompt[:len(prompt)-1] + paint(color.ASCIIGrey, " | ") + paint(color.ASCIIBlue, "d: full diff") + "\n"
	}
	if echo {
		prompt = prompt[:len(prompt)-1] + "\r\n"
	}
	if selection {
		prompt += paint(color.ASCIIBlue, " > ")
	}
	if err := writeText(output, prompt); err != nil {
		return "", false, err
	}
	var value []byte
	for {
		if err := ctx.Err(); err != nil {
			return "", false, err
		}
		key, err := next()
		if err != nil {
			return "", false, err
		}
		if mode == ApplyOrExpand && key == "d" {
			return key, false, nil
		}
		switch key {
		case "enter":
			return string(value), false, nil
		case "esc", "ctrl+d":
			return "", true, nil
		case "ctrl+c":
			return "", false, context.Canceled
		case "backspace":
			if selection && len(value) > 0 {
				value = value[:len(value)-1]
				if echo {
					if err := writeText(output, "\b \b"); err != nil {
						return "", false, err
					}
				}
			}
		default:
			if key == "space" {
				key = " "
			}
			if !selection || len(key) != 1 || key[0] < 32 || key[0] > 126 {
				continue
			}
			if len(value) >= 1024 {
				return "", false, fmt.Errorf("fix selection exceeds 1024 bytes")
			}
			value = append(value, key[0])
			if echo {
				if err := writeText(output, paint(color.ASCIIBlue, key)); err != nil {
					return "", false, err
				}
			}
		}
	}
}

func writeText(output io.Writer, text string) error {
	n, err := io.WriteString(output, text)
	if err == nil && n != len(text) {
		return io.ErrShortWrite
	}
	return err
}
