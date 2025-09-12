// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package cui

import (
	"fmt"
	"regexp"
	"strings"
)

// ShortcodeHandler defines how to transform a shortcode into markdown
type ShortcodeHandler struct {
	// Pattern is the regex pattern to match the shortcode
	Pattern *regexp.Regexp
	// Transform is the function that converts matched shortcode to markdown
	Transform func(matches []string) string
}

// ShortcodeParser holds the configuration for parsing shortcodes
type ShortcodeParser struct {
	handlers []ShortcodeHandler
}

// NewShortcodeParser creates a new shortcode parser with default handlers
func NewShortcodeParser() *ShortcodeParser {
	return &ShortcodeParser{
		handlers: []ShortcodeHandler{
			// warn-box shortcode handler
			{
				Pattern: regexp.MustCompile(`{{<\s*warn-box\s*>}}([\s\S]*?){{</\s*warn-box\s*>}}`),
				Transform: func(matches []string) string {
					if len(matches) > 1 {
						content := strings.TrimSpace(matches[1])
						return fmt.Sprintf("**\u21e8\u21e8\u21e8 WARNING \u21e6\u21e6\u21e6** \n\n%s\n", content)
					}
					return ""
				},
			},

			// Generic shortcode with parameters (e.g., {{< shortcode param="value" >}})
			{
				Pattern: regexp.MustCompile(`{{<\s*(\w+)\s+([^>]+?)\s*>}}`),
				Transform: func(matches []string) string {
					if len(matches) > 2 {
						shortcodeName := matches[1]
						params := matches[2]
						return fmt.Sprintf("[%s: %s]", strings.ToUpper(shortcodeName), params)
					}
					return ""
				},
			},
			// Simple shortcode without content (e.g., {{< br >}} or {{< hr >}})
			{
				Pattern: regexp.MustCompile(`{{<\s*(br|hr)\s*>}}`),
				Transform: func(matches []string) string {
					if len(matches) > 1 {
						switch matches[1] {
						case "br":
							return "\n"
						case "hr":
							return "\n---\n"
						}
					}
					return ""
				},
			},
		},
	}
}

// AddHandler adds a custom shortcode handler to the parser
func (p *ShortcodeParser) AddHandler(pattern *regexp.Regexp, transform func([]string) string) {
	p.handlers = append(p.handlers, ShortcodeHandler{
		Pattern:   pattern,
		Transform: transform,
	})
}

// Parse processes the input text and replaces all shortcodes with their markdown equivalents
func (p *ShortcodeParser) Parse(input string) string {
	result := input

	// Process each handler in order
	for _, handler := range p.handlers {
		matches := handler.Pattern.FindAllStringSubmatch(result, -1)
		for _, match := range matches {
			replacement := handler.Transform(match)
			result = strings.Replace(result, match[0], replacement, 1)
		}
	}

	return result
}

// ConvertHugoShortcodesToMarkdown is a convenience function that converts Hugo shortcodes to markdown
// with highlight syntax using the default parser configuration
func ConvertHugoShortcodesToMarkdown(content string) string {
	parser := NewShortcodeParser()
	return parser.Parse(content)
}

// ConvertHugoShortcodesToMarkdownWithCustomHandlers allows adding custom handlers before parsing
func ConvertHugoShortcodesToMarkdownWithCustomHandlers(content string, customHandlers []ShortcodeHandler) string {
	parser := NewShortcodeParser()

	// Add custom handlers
	for _, handler := range customHandlers {
		parser.handlers = append(parser.handlers, handler)
	}

	return parser.Parse(content)
}
