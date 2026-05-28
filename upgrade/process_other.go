// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

//go:build !(aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris)

package upgrade

func processAlive(pid int) (bool, bool) {
	return false, false
}
