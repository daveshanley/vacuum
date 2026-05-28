// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package upgrade

import (
	"errors"
	"syscall"
)

func processAlive(pid int) (bool, bool) {
	if pid <= 0 {
		return false, true
	}
	err := syscall.Kill(pid, 0)
	if err == nil || errors.Is(err, syscall.EPERM) {
		return true, true
	}
	if errors.Is(err, syscall.ESRCH) {
		return false, true
	}
	return true, true
}
