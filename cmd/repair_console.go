package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/daveshanley/vacuum/color"
	"github.com/daveshanley/vacuum/motor"
	"golang.org/x/term"
)

// Console reviews use the lint palette, but only on the review output terminal.
// In particular, redirected stderr must not inherit stdout's color capability.
type repairConsole struct {
	styled bool
	width  int
}

func newRepairConsole(output io.Writer) repairConsole {
	c := repairConsole{width: 100}
	if file, ok := output.(interface{ Fd() uintptr }); ok && term.IsTerminal(int(file.Fd())) {
		c.styled = !color.AreColorsDisabled() && os.Getenv("NO_COLOR") == ""
		if width, _, err := term.GetSize(int(file.Fd())); err == nil && width > 0 {
			c.width = width
		}
	}
	return c
}

func (c repairConsole) paint(style, text string) string {
	if !c.styled {
		return text
	}
	return style + text + color.ASCIIReset
}

func repairConsoleText(text string) string {
	return strings.Map(func(r rune) rune {
		if r < 32 || r == 127 || r >= 0x80 && r <= 0x9f {
			return ' '
		}
		return r
	}, text)
}

func (c repairConsole) heading(title string) string {
	return "\n " + c.paint(color.ASCIIBlue+color.ASCIIBold, repairConsoleText(title)) + "\n " + c.paint(color.ASCIIPink, strings.Repeat("─", max(1, c.width-2))) + "\n"
}

func (c repairConsole) options(options []motor.FixOption) string {
	var out strings.Builder
	out.WriteString(c.heading("Available fixes"))
	idWidth, ruleWidth := 3, 4
	for _, option := range options {
		if option.Available {
			idWidth = max(idWidth, len(fmt.Sprint(option.Index+1)))
			ruleWidth = max(ruleWidth, ansi.StringWidth(repairConsoleText(option.Result.RuleId)))
		}
	}
	ruleWidth = min(ruleWidth, max(8, c.width/3))
	pathWidth := max(8, c.width-idWidth-ruleWidth-7)
	fmt.Fprintf(&out, " %s  %s  %s\n", c.paint(color.ASCIIPink+color.ASCIIBold, fmt.Sprintf("%-*s", idWidth, "Fix")), c.paint(color.ASCIIPink+color.ASCIIBold, fmt.Sprintf("%-*s", ruleWidth, "Rule")), c.paint(color.ASCIIPink+color.ASCIIBold, "Path"))
	fmt.Fprintf(&out, " %s\n", c.paint(color.ASCIIPink, strings.Repeat("─", max(1, c.width-2))))
	for _, option := range options {
		if !option.Available {
			continue
		}
		rules := strings.Split(ansi.Hardwrap(repairConsoleText(option.Result.RuleId), ruleWidth, true), "\n")
		paths := strings.Split(ansi.Hardwrap(repairConsoleText(option.Result.Path), pathWidth, true), "\n")
		for line := 0; line < max(len(rules), len(paths)); line++ {
			id, rule, path := "", "", ""
			if line == 0 {
				id = fmt.Sprint(option.Index + 1)
			}
			if line < len(rules) {
				rule = rules[line]
			}
			if line < len(paths) {
				path = paths[line]
			}
			fmt.Fprintf(&out, " %s  %s  %s\n", c.paint(color.ASCIIYellow, fmt.Sprintf("%*s", idWidth, id)), c.paint(color.ASCIIBlue, rule+strings.Repeat(" ", ruleWidth-ansi.StringWidth(rule))), c.paint(color.ASCIIGrey, path))
		}
	}
	fmt.Fprintf(&out, " %s\n", c.paint(color.ASCIIPink, strings.Repeat("─", max(1, c.width-2))))
	out.WriteString(c.paint(color.ASCIIGrey, " Enter fix numbers separated by spaces | all: select all | enter: cancel\n"))
	out.WriteString(c.paint(color.ASCIIYellow, " > "))
	return out.String()
}

// Stream color around diff lines without buffering or wrapping source text.
// A line can cross any number of writes, including a large minified document.
type repairConsoleDiffWriter struct {
	output  io.Writer
	console repairConsole
	inLine  bool
}

func (w *repairConsoleDiffWriter) Write(data []byte) (int, error) {
	if !w.console.styled {
		n, err := w.output.Write(data)
		if err == nil && n != len(data) {
			err = io.ErrShortWrite
		}
		return n, err
	}
	consumed := 0
	for len(data) > 0 {
		if !w.inLine {
			style := ""
			switch data[0] {
			case '-':
				style = color.ASCIIBlue
			case '+':
				style = color.ASCIIYellow
			case '@':
				style = color.ASCIIPink
			}
			if w.console.styled {
				if err := writeRepairReview(w.output, style); err != nil {
					return consumed, err
				}
			}
			w.inLine = true
		}
		size := len(data)
		newline := bytes.IndexByte(data, '\n')
		if newline >= 0 {
			size = newline
		}
		n, err := w.output.Write(data[:size])
		consumed += n
		if err != nil {
			return consumed, err
		}
		if n != size {
			return consumed, io.ErrShortWrite
		}
		data = data[size:]
		if newline >= 0 {
			if w.console.styled {
				if err := writeRepairReview(w.output, color.ASCIIReset); err != nil {
					return consumed, err
				}
			}
			if err := writeRepairReview(w.output, "\n"); err != nil {
				return consumed, err
			}
			consumed++
			data = data[1:]
			w.inLine = false
		}
	}
	return consumed, nil
}

// Check the underlying writer before bufio can retry a short write.
type repairConsoleCheckedWriter struct{ io.Writer }

func (w repairConsoleCheckedWriter) Write(data []byte) (int, error) {
	n, err := w.Writer.Write(data)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	return n, err
}
