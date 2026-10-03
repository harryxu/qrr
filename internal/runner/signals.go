package runner

import (
	"context"
	"errors"
	"io"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sys/unix"
)

// ErrInterrupted identifies a deliberate user interrupt rather than an execution failure.
var ErrInterrupted = errors.New("interrupted by user")

// InterruptContext cancels on SIGINT and records why cancellation occurred.
func InterruptContext(parent context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancelCause(parent)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	go func() {
		select {
		case <-signals:
			cancel(ErrInterrupted)
		case <-ctx.Done():
		}
	}()
	return ctx, func() { signal.Stop(signals); cancel(context.Canceled) }
}

func userInterrupted(ctx context.Context) bool { return errors.Is(context.Cause(ctx), ErrInterrupted) }

// The child inherits our process group, so a terminal interrupt already reaches it.
func sharesForegroundTerminal(in io.Reader) bool {
	file, ok := in.(interface{ Fd() uintptr })
	if !ok {
		return false
	}
	group, err := unix.IoctlGetInt(int(file.Fd()), unix.TIOCGPGRP)
	return err == nil && group == syscall.Getpgrp()
}
