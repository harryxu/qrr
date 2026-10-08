package recipe

import "os/exec"

// CommandContext kills the provider on cancellation. WaitDelay bounds pipe waits.
func configureOptionsProcess(cmd *exec.Cmd) func() { return func() {} }
