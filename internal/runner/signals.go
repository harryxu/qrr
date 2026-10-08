package runner

import (
	"context"
	"errors"
	"os"
	"os/signal"
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
