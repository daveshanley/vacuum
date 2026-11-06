package cmd

import "golang.org/x/sys/unix"

func prepareRepairPrompt(fd int) error {
	state, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return err
	}
	state.Lflag |= unix.ISIG // Keep Ctrl-C connected to the command's signal context.
	if err = unix.IoctlSetTermios(fd, unix.TCSETS, state); err != nil {
		return err
	}
	return unix.IoctlSetInt(fd, unix.TCFLSH, 0) // TCIFLUSH is zero on Linux.
}
