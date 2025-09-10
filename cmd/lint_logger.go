// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/lipgloss/v2/tree"
	"github.com/daveshanley/vacuum/cui"
)

// LogEntry represents a structured log entry
type LogEntry struct {
	Level   string
	Message string
	Fields  map[string]interface{}
}

// BufferedLogger collects log entries for later rendering
type BufferedLogger struct {
	mu       sync.Mutex
	entries  []LogEntry
	logLevel string // Minimum log level to display
}

// NewBufferedLogger creates a new buffered logger
func NewBufferedLogger() *BufferedLogger {
	return &BufferedLogger{
		entries:  make([]LogEntry, 0),
		logLevel: cui.LogLevelError, // Default to error level
	}
}

// NewBufferedLoggerWithLevel creates a new buffered logger with a specific log level
func NewBufferedLoggerWithLevel(level string) *BufferedLogger {
	return &BufferedLogger{
		entries:  make([]LogEntry, 0),
		logLevel: level,
	}
}

// SetLogLevel sets the minimum log level to display
func (l *BufferedLogger) SetLogLevel(level string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logLevel = level
}

// getSeverityPriority returns a priority value for log levels (lower = more severe)
func getSeverityPriority(level string) int {
	switch level {
	case cui.LogLevelError:
		return 1
	case cui.LogLevelWarn:
		return 2
	case cui.LogLevelInfo:
		return 3
	case cui.LogLevelDebug:
		return 4
	default:
		return 5
	}
}

// shouldLog determines if a log entry should be stored based on severity
func (l *BufferedLogger) shouldLog(level string) bool {
	// Get priorities (lower number = more severe)
	entryPriority := getSeverityPriority(level)
	configuredPriority := getSeverityPriority(l.logLevel)
	
	// Log if entry severity is equal or more severe than configured level
	return entryPriority <= configuredPriority
}

// Error logs an error level message
func (l *BufferedLogger) Error(msg string, fields ...interface{}) {
	l.log(cui.LogLevelError, msg, fields...)
}

// Warn logs a warning level message
func (l *BufferedLogger) Warn(msg string, fields ...interface{}) {
	l.log(cui.LogLevelWarn, msg, fields...)
}

// Info logs an info level message
func (l *BufferedLogger) Info(msg string, fields ...interface{}) {
	l.log(cui.LogLevelInfo, msg, fields...)
}

// Debug logs a debug level message
func (l *BufferedLogger) Debug(msg string, fields ...interface{}) {
	l.log(cui.LogLevelDebug, msg, fields...)
}

// log is the internal logging method
func (l *BufferedLogger) log(level, msg string, fields ...interface{}) {
	// Check if we should log this based on severity
	if !l.shouldLog(level) {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	entry := LogEntry{
		Level:   level,
		Message: msg,
		Fields:  make(map[string]interface{}),
	}

	// Parse fields as key-value pairs
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			if key, ok := fields[i].(string); ok {
				entry.Fields[key] = fields[i+1]
			}
		}
	}

	l.entries = append(l.entries, entry)
}

// GetEntries returns all collected log entries
func (l *BufferedLogger) GetEntries() []LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.entries
}

// Clear removes all log entries
func (l *BufferedLogger) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = l.entries[:0]
}

// RenderTree renders all log entries as a tree structure using lipgloss v2
func (l *BufferedLogger) RenderTree(noStyle bool) string {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.entries) == 0 {
		return ""
	}

	var output strings.Builder

	// Define styles for tree rendering
	errorText := lipgloss.NewStyle().Foreground(cui.RGBRed)
	debugText := lipgloss.NewStyle().Foreground(cui.RGBGrey)
	treeStyleOriginal := lipgloss.NewStyle().Foreground(cui.RGBPink)
	keyStyleOriginal := lipgloss.NewStyle().Bold(true).Foreground(cui.RGBBlue) // blue bold for keys

	keyStyle := keyStyleOriginal
	treeStyle := treeStyleOriginal
	for i, entry := range l.entries {
		keyStyle = keyStyleOriginal
		treeStyle = treeStyleOriginal
		severityPrefix, severityColor := getLogSeverityInfo(entry.Level)

		switch entry.Level {
		case cui.LogLevelError:
			keyStyle = lipgloss.NewStyle().Bold(true).
				Foreground(lipgloss.Color(severityColor))
			treeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(severityColor))

		case cui.LogLevelInfo:
		case cui.LogLevelWarn:
			keyStyle = lipgloss.NewStyle().Bold(true).
				Foreground(lipgloss.Color(severityColor))
			treeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(severityColor))
		case cui.LogLevelDebug:
			keyStyle = lipgloss.NewStyle().Bold(true).
				Foreground(cui.RGBGrey)
			treeStyle = lipgloss.NewStyle().Foreground(cui.RGBGrey)
		}

		// Create the main message with severity
		var mainMsg string
		if noStyle {
			mainMsg = fmt.Sprintf("%s %s", severityPrefix, entry.Message)
		} else {
			severityStyle := lipgloss.NewStyle().
				Background(lipgloss.Color(severityColor)).
				Foreground(cui.RGBBlack).
				Bold(true)

			// split the severity prefix and message with a space
			severityPrefix = severityStyle.Render(fmt.Sprintf(" %s ", severityPrefix))

			m := cui.ColorizeLogMessage(entry.Message, entry.Level)
			mainMsg = fmt.Sprintf("%s %s", severityPrefix, m)
		}
		output.WriteString(mainMsg + "\n")

		// Create tree for fields if present
		if len(entry.Fields) > 0 {
			// Build tree nodes for fields
			var nodes []any

			for key, value := range entry.Fields {
				// Use reflection to check if value is a slice/array
				rv := reflect.ValueOf(value)

				if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
					// Handle slices/arrays
					if rv.Len() > 0 {
						// Create a node with the key and sub-items
						var subNodes []any
						for i := 0; i < rv.Len(); i++ {
							item := rv.Index(i).Interface()
							// Convert item to string
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
						// Empty array
						var fieldNode string
						if noStyle {
							fieldNode = fmt.Sprintf(" %s: []", key)
						} else {
							fieldNode = fmt.Sprintf(" %s: []", keyStyle.Render(key))
						}
						nodes = append(nodes, fieldNode)
					}
				} else {
					// Regular single value field
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
						if entry.Level == cui.LogLevelError {
							fieldNode = fmt.Sprintf(" %s: %s", keyStyle.Render(key), errorText.Render(valueStr))
						}
						if entry.Level == cui.LogLevelDebug {
							fieldNode = fmt.Sprintf(" %s: %s", keyStyle.Render(key), debugText.Render(valueStr))
						}
					}
					nodes = append(nodes, fieldNode)
				}
			}

			// Create and render the tree
			t := tree.New().Child(nodes...)
			if !noStyle {
				t = t.EnumeratorStyle(treeStyle)
			}

			// Add indentation for the tree
			treeOutput := t.String()
			lines := strings.Split(treeOutput, "\n")
			for _, line := range lines {
				if line != "" {
					output.WriteString("  " + line + "\n")
				}
			}
		}

		// Add spacing between entries except for the last one
		if i < len(l.entries)-1 {
			output.WriteString("\n")
		}
	}

	return output.String()
}

// getLogSeverityInfo returns the prefix and color code for a log severity level
func getLogSeverityInfo(level string) (string, string) {
	switch level {
	case cui.LogLevelError:
		return "ERR", "196" // red
	case cui.LogLevelWarn:
		return "WRN", "220" // yellow
	case cui.LogLevelInfo:
		return "INF", "39" // blue
	case cui.LogLevelDebug:
		return "DEV", "244" // grey
	default:
		return "•", "15" // white
	}
}

// BufferedLogHandler implements slog.Handler interface for BufferedLogger
type BufferedLogHandler struct {
	logger *BufferedLogger
}

// NewBufferedLogHandler creates a new slog.Handler that writes to BufferedLogger
func NewBufferedLogHandler(logger *BufferedLogger) *BufferedLogHandler {
	return &BufferedLogHandler{logger: logger}
}

// Enabled reports whether the handler handles records at the given level
func (h *BufferedLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true // Accept all levels, we'll filter later if needed
}

// Handle handles the Record
func (h *BufferedLogHandler) Handle(ctx context.Context, record slog.Record) error {
	// Convert slog level to our level
	var level string
	switch record.Level {
	case slog.LevelError:
		level = cui.LogLevelError
	case slog.LevelWarn:
		level = cui.LogLevelWarn
	case slog.LevelInfo:
		level = cui.LogLevelInfo
	case slog.LevelDebug:
		level = cui.LogLevelDebug
	default:
		level = cui.LogLevelInfo
	}

	// Collect attributes
	var fields []interface{}
	record.Attrs(func(attr slog.Attr) bool {
		fields = append(fields, attr.Key, attr.Value.Any())
		return true
	})

	// Log to our buffered logger
	h.logger.log(level, record.Message, fields...)
	return nil
}

// WithAttrs returns a new Handler with attributes
func (h *BufferedLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// For simplicity, return the same handler
	// In a full implementation, you'd store and merge attributes
	return h
}

// WithGroup returns a new Handler with a group name
func (h *BufferedLogHandler) WithGroup(name string) slog.Handler {
	// For simplicity, return the same handler
	// In a full implementation, you'd handle grouping
	return h
}
