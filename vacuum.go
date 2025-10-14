package main

import (
	"github.com/daveshanley/vacuum/cmd"
)

// These variables can be set via ldflags during build time
// Example: go build -ldflags "-X main.version=v1.0.0 -X main.commit=abc1234 -X 'main.date=2025-01-14'"
var (
	version string
	commit  string
	date    string
)

func main() {
	// Pass ldflags values to cmd package
	cmd.Execute(version, commit, date)
}
