// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package logging

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/lipgloss/v2/tree"
	"github.com/daveshanley/vacuum/color"
	"github.com/daveshanley/vacuum/utils"
)

// Log level constants
const (
	LogLevelError = "ERROR"
	LogLevelWarn  = "WARN"
	LogLevelInfo  = "INFO"
	LogLevelDebug = "DEBUG"
)

type LogEntry struct {
	Level   string
	Message string
	Fields  map[string]interface{}
}

type BufferedLogger struct {
	mu         sync.Mutex
	entries    []LogEntry
	logLevel   string
	maxEntries int  // maximum entries to keep, 0 = unlimited
	discardLog bool // if true, don't store entries at all
}

func NewBufferedLogger() *BufferedLogger {
	return &BufferedLogger{
		entries:    make([]LogEntry, 0),
		logLevel:   LogLevelError,
		maxEntries: 0, // unlimited by default
	}
}

func NewBufferedLoggerWithLevel(level string) *BufferedLogger {
	return &BufferedLogger{
		entries:    make([]LogEntry, 0),
		logLevel:   level,
		maxEntries: 0,
	}
}

// NewDiscardLogger creates a logger that discards all entries (for LSP)
func NewDiscardLogger() *BufferedLogger {
	return &BufferedLogger{
		entries:    make([]LogEntry, 0),
		logLevel:   LogLevelError,
		maxEntries: 0,
		discardLog: true,
	}
}

func (l *BufferedLogger) SetLogLevel(level string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logLevel = level
}

// SetMaxEntries sets the maximum number of entries to keep (0 = unlimited)
// when the limit is reached, oldest entries are removed
func (l *BufferedLogger) SetMaxEntries(max int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.maxEntries = max
	// trim existing entries if needed
	if max > 0 && len(l.entries) > max {
		l.entries = l.entries[len(l.entries)-max:]
	}
}

// SetDiscardMode sets whether to discard all log entries (useful for LSP)
func (l *BufferedLogger) SetDiscardMode(discard bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.discardLog = discard
	if discard {
		l.entries = l.entries[:0] // clear existing entries
	}
}

func getSeverityPriority(level string) int {
	switch level {
	case LogLevelError:
		return 1
	case LogLevelWarn:
		return 2
	case LogLevelInfo:
		return 3
	case LogLevelDebug:
		return 4
	default:
		return 5
	}
}

func (l *BufferedLogger) shouldLog(level string) bool {
	entryPriority := getSeverityPriority(level)
	configuredPriority := getSeverityPriority(l.logLevel)
	return entryPriority <= configuredPriority
}

func (l *BufferedLogger) Error(msg string, fields ...interface{}) {
	l.log(LogLevelError, msg, fields...)
}

func (l *BufferedLogger) Warn(msg string, fields ...interface{}) {
	l.log(LogLevelWarn, msg, fields...)
}

func (l *BufferedLogger) Info(msg string, fields ...interface{}) {
	l.log(LogLevelInfo, msg, fields...)
}

func (l *BufferedLogger) Debug(msg string, fields ...interface{}) {
	l.log(LogLevelDebug, msg, fields...)
}

func (l *BufferedLogger) log(level, msg string, fields ...interface{}) {
	if !l.shouldLog(level) {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// if in discard mode, don't store anything
	if l.discardLog {
		return
	}

	entry := LogEntry{
		Level:   level,
		Message: msg,
		Fields:  make(map[string]interface{}),
	}

	// parse fields as key-value pairs
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			if key, ok := fields[i].(string); ok {
				entry.Fields[key] = fields[i+1]
			}
		}
	}

	l.entries = append(l.entries, entry)

	// enforce max entries limit using ring buffer approach
	if l.maxEntries > 0 && len(l.entries) > l.maxEntries {
		// remove oldest entry by shifting slice
		l.entries = l.entries[1:]
	}
}

func (l *BufferedLogger) GetEntries() []LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.entries
}

func (l *BufferedLogger) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = l.entries[:0]
}

func (l *BufferedLogger) RenderTree(noStyle bool) string {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.entries) == 0 {
		return ""
	}

	var output strings.Builder

	errorText := lipgloss.NewStyle().Foreground(color.RGBRed)
	debugText := lipgloss.NewStyle().Foreground(color.RGBGrey)
	treeStyleOriginal := lipgloss.NewStyle().Foreground(color.RGBPink)
	keyStyleOriginal := lipgloss.NewStyle().Bold(true).Foreground(color.RGBBlue)

	keyStyle := keyStyleOriginal
	treeStyle := treeStyleOriginal
	for i, entry := range l.entries {
		keyStyle = keyStyleOriginal
		treeStyle = treeStyleOriginal
		severityPrefix, severityColor := getLogSeverityInfo(entry.Level)

		switch entry.Level {
		case LogLevelError:
			keyStyle = lipgloss.NewStyle().Bold(true).
				Foreground(lipgloss.Color(severityColor))
			treeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(severityColor))

		case LogLevelInfo:
		case LogLevelWarn:
			keyStyle = lipgloss.NewStyle().Bold(true).
				Foreground(lipgloss.Color(severityColor))
			treeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(severityColor))
		case LogLevelDebug:
			keyStyle = lipgloss.NewStyle().Bold(true).
				Foreground(color.RGBGrey)
			treeStyle = lipgloss.NewStyle().Foreground(color.RGBGrey)
		}

		var mainMsg string
		if noStyle {
			mainMsg = fmt.Sprintf("%s %s", severityPrefix, entry.Message)
		} else {
			severityStyle := lipgloss.NewStyle().
				Background(lipgloss.Color(severityColor)).
				Foreground(color.RGBBlack).
				Bold(true)

			severityPrefix = severityStyle.Render(fmt.Sprintf(" %s ", severityPrefix))

			m := ColorizeLogMessage(entry.Message, entry.Level)
			mainMsg = fmt.Sprintf("%s %s", severityPrefix, m)
		}
		output.WriteString(mainMsg + "\n")

		if len(entry.Fields) > 0 {
			var nodes []any

			for key, value := range entry.Fields {
				rv := reflect.ValueOf(value)

				if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
					if rv.Len() > 0 {
						var subNodes []any
						for i := 0; i < rv.Len(); i++ {
							item := rv.Index(i).Interface()
							var itemStr string
							switch v := item.(type) {
							case error:
								itemStr = v.Error()
							case string:
								itemStr = v
							default:
								itemStr = fmt.Sprintf(" %v", v)
							}

							subStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(severityColor))
							subNodes = append(subNodes, fmt.Sprintf(" %s", subStyle.Render(itemStr)))
						}

						var keyNode string
						if noStyle {
							keyNode = fmt.Sprintf("%s", key)
						} else {
							keyNode = " " + keyStyle.Render(key)
						}

						nodes = append(nodes, tree.Root(keyNode).Child(subNodes...))
					} else {
						var fieldNode string
						if noStyle {
							fieldNode = fmt.Sprintf(" %s: []", key)
						} else {
							fieldNode = fmt.Sprintf(" %s: []", keyStyle.Render(key))
						}
						nodes = append(nodes, fieldNode)
					}
				} else {
					var valueStr string
					switch v := value.(type) {
					case error:
						valueStr = v.Error()
					case string:
						valueStr = v
					default:
						valueStr = fmt.Sprintf("%v", v)
					}

					var fieldNode string
					if noStyle {
						fieldNode = fmt.Sprintf(" %s: %s", key, valueStr)
					} else {
						fieldNode = fmt.Sprintf(" %s: %s", keyStyle.Render(key), valueStr)
						if entry.Level == LogLevelError {
							fieldNode = fmt.Sprintf(" %s: %s", keyStyle.Render(key), errorText.Render(valueStr))
						}
						if entry.Level == LogLevelDebug {
							fieldNode = fmt.Sprintf(" %s: %s", keyStyle.Render(key), debugText.Render(valueStr))
						}
					}
					nodes = append(nodes, fieldNode)
				}
			}

			t := tree.New().Child(nodes...)
			if !noStyle {
				t = t.EnumeratorStyle(treeStyle)
			}

			treeOutput := t.String()
			lines := strings.Split(treeOutput, "\n")
			for _, line := range lines {
				if line != "" {
					output.WriteString("  " + line + "\n")
				}
			}
		}

		if i < len(l.entries)-1 {
			output.WriteString("\n")
		}
	}

	return output.String()
}

func getLogSeverityInfo(level string) (string, string) {
	switch level {
	case LogLevelError:
		return "ERR", "196"
	case LogLevelWarn:
		return "WRN", "220"
	case LogLevelInfo:
		return "INF", "39"
	case LogLevelDebug:
		return "DEV", "244"
	default:
		return "•", "15"
	}
}

type BufferedLogHandler struct {
	logger *BufferedLogger
}

func NewBufferedLogHandler(logger *BufferedLogger) *BufferedLogHandler {
	return &BufferedLogHandler{logger: logger}
}

func (h *BufferedLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (h *BufferedLogHandler) Handle(ctx context.Context, record slog.Record) error {
	var level string
	switch record.Level {
	case slog.LevelError:
		level = LogLevelError
	case slog.LevelWarn:
		level = LogLevelWarn
	case slog.LevelInfo:
		level = LogLevelInfo
	case slog.LevelDebug:
		level = LogLevelDebug
	default:
		level = LogLevelInfo
	}

	var fields []interface{}
	record.Attrs(func(attr slog.Attr) bool {
		fields = append(fields, attr.Key, attr.Value.Any())
		return true
	})

	h.logger.log(level, record.Message, fields...)
	return nil
}

func (h *BufferedLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *BufferedLogHandler) WithGroup(name string) slog.Handler {
	return h
}

func ColorizeLogMessage(message, severity string) string {
	if severity == LogLevelError {
		return color.StyleLogError.Render(message)
	}
	if severity == LogLevelDebug {
		return color.StyleLogDebug.Render(message)
	}

	if !strings.HasPrefix(message, "[") {
		return message
	}

	// Use regex to find and colorize backtick-enclosed text
	res := utils.LogPrefixRegex.ReplaceAllStringFunc(message, func(match string) string {
		// Extract the content between backticks
		content := utils.LogPrefixRegex.FindStringSubmatch(match)
		if len(content) > 1 {
			// Apply styling using lipgloss
			switch severity {
			case LogLevelError:
				return color.StyleLogError.Copy().Italic(true).Render("[" + content[1] + "]")
			case LogLevelWarn:
				return color.StyleLogWarn.Copy().Render("[" + content[1] + "]")
			case LogLevelInfo:
				return color.StyleLogInfo.Copy().Render("[" + content[1] + "]")
			case LogLevelDebug:
				return color.StyleLogDebug.Copy().Render("[" + content[1] + "]")
			}

		}
		return match
	})
	return res
}
