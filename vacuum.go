package main

import (
	"github.com/daveshanley/vacuum/cmd"
)

var version string
var commit string
var date string

func main() {
	if version == "" {
		version = "latest"
	}
	cmd.Execute(version, commit, date)
}
