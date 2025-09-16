// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package utils

import "regexp"

var (
	LocationRegex    = regexp.MustCompile(`((?:[a-zA-Z]:)?[^\s│]*?[/\\]?[^\s│/\\]+\.[a-zA-Z]+):(\d+):(\d+)`)
	JsonPathRegex    = regexp.MustCompile(`\$\.\S+`)
	CircularRefRegex = regexp.MustCompile(`\b[a-zA-Z0-9_-]+(?:\s*->\s*[a-zA-Z0-9_-]+)+\b`)
	PartRegex        = regexp.MustCompile(`([a-zA-Z0-9_-]+)|(\s*->\s*)`)
	BacktickRegex    = regexp.MustCompile("`([^`]+)`")
	SingleQuoteRegex = regexp.MustCompile(`'([^']+)'`)
	LogPrefixRegex   = regexp.MustCompile(`\[([^]]+)]`)
)
