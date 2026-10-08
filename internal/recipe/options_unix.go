//go:build !windows

package recipe

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

func configureOptionsProcess(cmd *exec.Cmd) func() {
	// Isolate the provider so cancellation also stops pipeline descendants.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	return func() {
		if cmd.Process != nil {
			// Clean up descendants that outlive the provider or keep its pipes open.
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
	}
}
