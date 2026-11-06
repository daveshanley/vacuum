//go:build darwin || freebsd || openbsd || netbsd || dragonfly

package cmd

import "golang.org/x/sys/unix"

func prepareRepairPrompt(fd int) error {
	state, err := unix.IoctlGetTermios(fd, unix.TIOCGETA)
	if err != nil {
		return err
	}
	state.Lflag |= unix.ISIG
	if err = unix.IoctlSetTermios(fd, unix.TIOCSETA, state); err != nil {
		return err
	}
	return unix.IoctlSetPointerInt(fd, unix.TIOCFLUSH, 1) // FREAD: discard pending input only.
}
