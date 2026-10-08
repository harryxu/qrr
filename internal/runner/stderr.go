package runner

import (
	"context"
	"io"
)

type cancellationWriter struct {
	ctx    context.Context
	output io.Writer
}

func (w cancellationWriter) Write(p []byte) (int, error) {
	if userInterrupted(w.ctx) {
		return len(p), nil
	}
	return w.output.Write(p)
}
