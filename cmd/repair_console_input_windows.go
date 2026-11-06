package cmd

import "golang.org/x/sys/windows"

func prepareRepairPrompt(fd int) error {
	handle := windows.Handle(fd)
	var mode uint32
	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		return err
	}
	if err := windows.SetConsoleMode(handle, mode|windows.ENABLE_PROCESSED_INPUT); err != nil {
		return err
	}
	return windows.FlushConsoleInputBuffer(handle)
}
