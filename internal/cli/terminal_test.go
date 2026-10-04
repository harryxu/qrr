package cli

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
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

func TestRecipeEditorChild(t *testing.T) {
	if os.Getenv("QRR_TEST_EDITOR") != "1" {
		return
	}
	path := os.Args[len(os.Args)-1]
	if path != os.Getenv("QRR_TEST_EDITOR_PATH") || os.Args[len(os.Args)-2] != "argument with spaces" {
		os.Exit(9)
	}
	// Verify that qrr restored cooked mode before handing the terminal to the editor.
	state := exec.Command("stty", "-a")
	state.Stdin = os.Stdin
	terminalState, err := state.Output()
	if err != nil || strings.Contains(string(terminalState), "-icanon") || strings.Contains(string(terminalState), "-echo ") {
		os.Exit(8)
	}
	if os.Getenv("QRR_TEST_EDITOR_FAIL") == "1" {
		os.Exit(7)
	}
	if err := os.WriteFile(path, []byte(os.Getenv("QRR_TEST_EDITOR_CONTENT")), 0600); err != nil {
		os.Exit(6)
	}
	os.Exit(0)
}

func TestTerminalSelectorEditAndReload(t *testing.T) {
	for _, mode := range []string{"valid", "invalid", "editor-failure"} {
		t.Run(mode, func(t *testing.T) {
			dir := fixture(t)
			path := filepath.Join(dir, "commands", "demo.yaml")
			original := "version: 1\nname: demo\ndescription: Original recipe\ntype: command\ncommand: [echo, original]\n"
			if err := os.WriteFile(path, []byte(original), 0600); err != nil {
				t.Fatal(err)
			}
			content := strings.ReplaceAll(original, "original", "edited")
			content = strings.ReplaceAll(content, "Original recipe", "Edited recipe")
			readyAfterEdit := "Edited recipe"
			if mode == "invalid" {
				content = "version: [invalid"
				readyAfterEdit = "yaml:"
				spare := "version: 1\nname: spare\ndescription: Spare recipe\ntype: command\ncommand: [echo, spare]\n"
				if err := os.WriteFile(filepath.Join(dir, "commands", "spare.yaml"), []byte(spare), 0600); err != nil {
					t.Fatal(err)
				}
			} else if mode == "editor-failure" {
				readyAfterEdit = "editor: exit status 7"
			}
			cmd := exec.Command(os.Args[0], "-test.run=^TestPromptCancellationChild$")
			editor := Display([]string{os.Args[0], "-test.run=^TestRecipeEditorChild$", "--", "argument with spaces"})
			cmd.Env = append(os.Environ(), "TERM=xterm-256color", "QRR_PROMPT_CANCEL_CHILD=selector", "QRR_PROMPT_CANCEL_CONFIG="+dir,
				"EDITOR="+editor, "QRR_TEST_EDITOR=1", "QRR_TEST_EDITOR_PATH="+path, "QRR_TEST_EDITOR_CONTENT="+content)
			if mode == "editor-failure" {
				cmd.Env = append(cmd.Env, "QRR_TEST_EDITOR_FAIL=1")
			}
			terminal, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 120})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = cmd.Process.Kill(); _ = terminal.Close() })
			output := make(chan string, 1)
			edited := make(chan struct{})
			go func() {
				var text strings.Builder
				buffer := make([]byte, 4096)
				started, reloaded := false, false
				for {
					n, err := terminal.Read(buffer)
					text.Write(buffer[:n])
					if !started && strings.Contains(text.String(), "ctrl+e edit") {
						started = true
						_, _ = terminal.Write([]byte{5})
					}
					if !reloaded && strings.Contains(text.String(), readyAfterEdit) {
						reloaded = true
						close(edited)
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
			case <-edited:
			case text := <-output:
				t.Fatalf("exited before returning from the editor: %q", text)
			case <-time.After(10 * time.Second):
				t.Fatal("editor did not return to a refreshed selector")
			}
			if mode == "editor-failure" {
				_, err = terminal.Write([]byte{3})
			} else {
				_, err = terminal.Write([]byte{'\r'})
			}
			if err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-finished:
				if mode == "editor-failure" {
					var exit *exec.ExitError
					if !errors.As(err, &exit) || exit.ExitCode() != 130 {
						t.Fatalf("cancellation after editor failure: %v", err)
					}
				} else if err != nil {
					t.Fatal(err)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("selector did not finish after editing")
			}
			select {
			case text := <-output:
				if strings.Contains(text, "Running:") || strings.Contains(text, "'original'") {
					t.Fatalf("ran a recipe or prepared stale configuration: %q", text)
				}
				if mode == "valid" && !strings.Contains(text, "'echo' 'edited'") {
					t.Fatalf("did not prepare edited configuration: %q", text)
				}
				if mode == "invalid" && !strings.Contains(text, "'echo' 'spare'") {
					t.Fatalf("invalid edit prevented selecting another recipe: %q", text)
				}
			case <-time.After(time.Second):
				t.Fatal("terminal did not close")
			}
		})
	}
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
