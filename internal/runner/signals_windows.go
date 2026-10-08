package runner

import (
	"io"
	"os"
)

func sharesForegroundTerminal(io.Reader) bool { return false }

// Go cannot send os.Interrupt to a Windows process. Kill it on cancellation.
func interruptProcess(process *os.Process) error { return process.Kill() }
