//go:build !windows

package runner

import (
	"io"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

// The child inherits our process group, so a terminal interrupt already reaches it.
func sharesForegroundTerminal(in io.Reader) bool {
	file, ok := in.(interface{ Fd() uintptr })
	if !ok {
		return false
	}
	group, err := unix.IoctlGetInt(int(file.Fd()), unix.TIOCGPGRP)
	return err == nil && group == syscall.Getpgrp()
}

func interruptProcess(process *os.Process) error { return process.Signal(os.Interrupt) }
