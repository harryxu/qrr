// Package runner executes argv directly, preserving process streams.
package runner

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func Run(ctx context.Context, args []string) error {
	return run(ctx, args, os.Stdin, os.Stdout, os.Stderr)
}

func run(ctx context.Context, args []string, in io.Reader, out, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdin = in
	cmd.Stdout = out
	relay, finish, err := relayStderr(ctx, stderr)
	if err != nil {
		return err
	}
	cmd.Stderr = relay
	cmd.Cancel = func() error {
		if userInterrupted(ctx) && sharesForegroundTerminal(in) {
			// Ctrl+C was delivered to the whole foreground group. Sending it
			// again can interrupt Python finalizers and produce a traceback.
			return nil
		}
		return interruptProcess(cmd.Process)
	}
	cmd.WaitDelay = 3 * time.Second
	runErr := cmd.Run()
	stderrErr := finish()
	if userInterrupted(ctx) {
		return context.Canceled
	}
	if runErr != nil {
		return runErr
	}
	return stderrErr
}

func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	if errors.Is(err, context.Canceled) {
		return 130
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		if status, ok := exit.Sys().(syscall.WaitStatus); ok && status.Signaled() {
			return 128 + int(status.Signal())
		}
		return exit.ExitCode()
	}
	return 1
}
