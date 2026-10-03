package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"

	"github.com/creack/pty"
	"golang.org/x/term"
	"testing"
	"time"
)

func TestHelperProcess(t *testing.T) {
	if os.Getenv("QRR_TEST_PROCESS") != "1" {
		return
	}
	mode := os.Args[len(os.Args)-1]
	if mode == "cleanup" {
		signals := make(chan os.Signal, 2)
		signal.Notify(signals, os.Interrupt)
		fmt.Fprintln(os.Stderr, "ready")
		<-signals
		time.Sleep(50 * time.Millisecond)
		fmt.Fprintln(os.Stderr, "Traceback: cleanup interrupted")
		fmt.Fprintln(os.Stderr, "ERROR: Interrupted by user")
		os.Exit(1)
	}
	if mode == "failure" {
		fmt.Fprintln(os.Stderr, "ERROR: actual download failure")
		os.Exit(23)
	}
	if mode == "terminal" {
		if !term.IsTerminal(int(os.Stderr.Fd())) {
			os.Exit(31)
		}
		fmt.Fprint(os.Stderr, "terminal output\n")
		os.Exit(0)
	}
	if os.Args[len(os.Args)-1] == "wait" {
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}
	os.Exit(23)
}
func TestExitCode(t *testing.T) {
	t.Setenv("QRR_TEST_PROCESS", "1")
	err := Run(context.Background(), []string{os.Args[0], "-test.run=TestHelperProcess", "--", "exit"})
	if ExitCode(err) != 23 {
		t.Fatalf("got %v / %d", err, ExitCode(err))
	}
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		t.Fatal("expected process exit error")
	}
}
func TestCancellation(t *testing.T) {
	t.Setenv("QRR_TEST_PROCESS", "1")
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := Run(ctx, []string{os.Args[0], "-test.run=TestHelperProcess", "--", "wait"})
	if err == nil || time.Since(start) > 5*time.Second {
		t.Fatalf("cancellation failed: %v", err)
	}
}

type readyWriter struct {
	bytes.Buffer
	ready chan struct{}
	once  sync.Once
}

func (w *readyWriter) Write(p []byte) (int, error) {
	n, err := w.Buffer.Write(p)
	if strings.Contains(w.Buffer.String(), "ready") {
		w.once.Do(func() { close(w.ready) })
	}
	return n, err
}

func TestUserCancellationSuppressesCleanupDiagnostics(t *testing.T) {
	t.Setenv("QRR_TEST_PROCESS", "1")
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	stderr := &readyWriter{ready: make(chan struct{})}
	done := make(chan error, 1)
	go func() {
		done <- run(ctx, []string{os.Args[0], "-test.run=^TestHelperProcess$", "--", "cleanup"}, nil, io.Discard, stderr)
	}()
	select {
	case <-stderr.ready:
	case <-time.After(5 * time.Second):
		t.Fatal("helper did not start")
	}
	cancel(ErrInterrupted)
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) || ExitCode(err) != 130 {
			t.Fatalf("unexpected cancellation: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cancelled process did not exit")
	}
	if stderr.String() != "ready\n" {
		t.Fatalf("cleanup leaked: %q", stderr.String())
	}
}

func TestRealFailuresRemainVisible(t *testing.T) {
	t.Setenv("QRR_TEST_PROCESS", "1")
	var stderr bytes.Buffer
	err := run(context.Background(), []string{os.Args[0], "-test.run=^TestHelperProcess$", "--", "failure"}, nil, io.Discard, &stderr)
	if ExitCode(err) != 23 || stderr.String() != "ERROR: actual download failure\n" {
		t.Fatalf("failure suppressed: %v %q", err, stderr.String())
	}
}

func TestStderrRemainsTerminalAndDrains(t *testing.T) {
	t.Setenv("QRR_TEST_PROCESS", "1")
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()
	received := make(chan []byte, 1)
	go func() { buffer := make([]byte, 4096); n, _ := master.Read(buffer); received <- buffer[:n] }()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = run(ctx, []string{os.Args[0], "-test.run=^TestHelperProcess$", "--", "terminal"}, nil, io.Discard, slave)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case data := <-received:
		if !bytes.Contains(data, []byte("terminal output")) {
			t.Fatalf("terminal output lost: %q", data)
		}
	case <-ctx.Done():
		t.Fatal("terminal output did not arrive")
	}
}
