package main

import (
	"github.com/daveshanley/vacuum/cmd"
	"time"
)

var version string
var commit string
var date string

func main() {
	if version == "" {
		version = "latest"
	}
	if commit == "" {
		commit = "latest"
	}
	if date == "" {
		date = time.Now().Format("2006-01-02 15:04:05 MST")
	}
	cmd.Execute(version, commit, date)
}
