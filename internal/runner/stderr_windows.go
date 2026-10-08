package runner

import (
	"context"
	"io"
)

// Windows has no Unix PTY relay; filter cleanup output through the normal stream.
func relayStderr(ctx context.Context, output io.Writer) (io.Writer, func() error, error) {
	return cancellationWriter{ctx: ctx, output: output}, func() error { return nil }, nil
}
