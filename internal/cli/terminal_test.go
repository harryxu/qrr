package cli

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"qrr/internal/runner"

	"github.com/creack/pty"
)

func TestPromptCancellationChild(t *testing.T) {
	mode := os.Getenv("QRR_PROMPT_CANCEL_CHILD")
	if mode == "" {
		return
	}
	args := []string{"--config-dir", os.Getenv("QRR_PROMPT_CANCEL_CONFIG"), "--dry-run"}
	if mode == "parameter" {
		args = append(args, "demo")
	}
	// A background context makes this exercise prompt cancellation independently
	// of the executable's SIGINT handler.
	err := Execute(context.Background(), args, os.Stdout, os.Stderr)
	os.Exit(runner.ExitCode(err))
}

func TestTerminalPromptCancellation(t *testing.T) {
	for _, mode := range []string{"selector", "parameter"} {
		t.Run(mode, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestPromptCancellationChild$")
			cmd.Env = append(os.Environ(), "QRR_PROMPT_CANCEL_CHILD="+mode, "QRR_PROMPT_CANCEL_CONFIG="+fixture(t), "TERM=xterm-256color")
			terminal, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 80})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = cmd.Process.Kill(); _ = terminal.Close() })
			ready := make(chan struct{})
			output := make(chan string, 1)
			readyHint := "enter submit"
			if mode == "selector" {
				readyHint = "ctrl+r rename"
			}
			go func() {
				var text strings.Builder
				buffer := make([]byte, 4096)
				notified := false
				for {
					n, err := terminal.Read(buffer)
					text.Write(buffer[:n])
					if !notified && strings.Contains(text.String(), readyHint) {
						close(ready)
						notified = true
					}
					if err != nil {
						output <- text.String()
						return
					}
				}
			}()
			finished := make(chan error, 1)
			go func() { finished <- cmd.Wait() }()
			select {
			case <-ready:
			case text := <-output:
				t.Fatalf("prompt exited before becoming ready: %q", text)
			case <-time.After(10 * time.Second):
				t.Fatal("prompt did not become ready")
			}
			if _, err := terminal.Write([]byte{3}); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-finished:
				var exit *exec.ExitError
				if !errors.As(err, &exit) || exit.ExitCode() != 130 {
					t.Fatalf("cancellation exit: %v", err)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("prompt did not exit after Ctrl+C")
			}
			select {
			case text := <-output:
				if strings.Contains(text, "Running:") {
					t.Fatalf("executed after cancellation: %q", text)
				}
				if mode == "selector" {
					if !strings.Contains(text, "ctrl+c quit") || !strings.Contains(text, "ctrl+r rename") {
						t.Fatalf("missing selector actions: %q", text)
					}
					for _, hint := range []string{"↑ up", "↓ down", "enter submit"} {
						if strings.Contains(text, hint) {
							t.Fatalf("unexpected selector hint %q: %q", hint, text)
						}
					}
				}
			case <-time.After(time.Second):
				t.Fatal("terminal did not close after cancellation")
			}
		})
	}
}
