//go:build !windows

package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// relayStderr keeps isatty true for terminal-aware programs while allowing us to
// discard cleanup diagnostics after a user interrupt. Stdout and stdin stay direct.
func relayStderr(ctx context.Context, output io.Writer) (io.Writer, func() error, error) {
	sink := cancellationWriter{ctx: ctx, output: output}
	file, ok := output.(*os.File)
	if !ok || !term.IsTerminal(int(file.Fd())) {
		return sink, func() error { return nil }, nil
	}
	master, slave, err := pty.Open()
	if err != nil {
		return nil, nil, fmt.Errorf("open stderr terminal: %w", err)
	}
	// A nonblocking duplicate lets closing the reader interrupt a blocked Go read.
	fd, err := unix.Dup(int(master.Fd()))
	if err != nil {
		master.Close()
		slave.Close()
		return nil, nil, err
	}
	if err := unix.SetNonblock(fd, true); err != nil {
		unix.Close(fd)
		master.Close()
		slave.Close()
		return nil, nil, err
	}
	unix.CloseOnExec(fd)
	reader := os.NewFile(uintptr(fd), "qrr-stderr")
	master.Close()
	if _, err := term.MakeRaw(int(slave.Fd())); err != nil {
		reader.Close()
		slave.Close()
		return nil, nil, err
	}
	if err := pty.InheritSize(file, slave); err != nil {
		reader.Close()
		slave.Close()
		return nil, nil, err
	}
	resize := make(chan os.Signal, 1)
	signal.Notify(resize, syscall.SIGWINCH)
	stopResize := make(chan struct{})
	resizeDone := make(chan struct{})
	go func() {
		defer close(resizeDone)
		for {
			select {
			case <-resize:
				_ = pty.InheritSize(file, slave)
			case <-stopResize:
				return
			}
		}
	}()
	done := make(chan error, 1)
	go func() {
		_, err := io.Copy(sink, reader)
		if errors.Is(err, unix.EIO) {
			err = nil
		}
		done <- err
	}()
	finish := func() error {
		signal.Stop(resize)
		close(stopResize)
		<-resizeDone
		slave.Close()
		defer reader.Close()
		select {
		case err := <-done:
			return err
		case <-time.After(3 * time.Second):
			reader.Close()
			<-done
			return fmt.Errorf("stderr terminal did not close after command exit")
		}
	}
	return slave, finish, nil
}
