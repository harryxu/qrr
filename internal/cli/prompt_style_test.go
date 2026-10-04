package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/creack/pty"
)

func TestEntryPromptTerminalStyleKeepsOneLogicalLine(t *testing.T) {
	for _, text := range []string{
		"Run qrr prompt -v 'flutter upgrade' and follow its instructions to add a qrr YAML recipe.",
		"Run qrr prompt -v 'echo " + strings.Repeat("long-command-argument", 20) + "' and follow its instructions to add a qrr YAML recipe.",
	} {
		rendered := renderEntryPrompt(text + "\n")
		if !strings.Contains(rendered, "\x1b[") {
			t.Fatal("missing terminal colors")
		}
		if strings.ContainsAny(rendered, "\n\r") {
			t.Fatal("styling inserted a hard line break")
		}
		if !strings.Contains(rendered, text) {
			t.Fatal("styling changed prompt text")
		}
	}
}

func TestEntryPromptPipeOmitsReminder(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()
	var stderr bytes.Buffer
	text := "Run qrr prompt -v 'flutter upgrade' and follow its instructions to add a qrr YAML recipe.\n"
	if err := writeEntryPrompt(writer, &stderr, text); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(reader)
	if err != nil || string(got) != text || stderr.Len() != 0 {
		t.Fatalf("pipe output=%q stderr=%q err=%v", got, stderr.String(), err)
	}
}

func TestEntryPromptTerminalReminderIndependentOfColor(t *testing.T) {
	for _, mode := range []string{"color", "no-color", "dumb"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("NO_COLOR", "")
			t.Setenv("TERM", "xterm-256color")
			if mode == "no-color" {
				t.Setenv("NO_COLOR", "1")
			}
			if mode == "dumb" {
				t.Setenv("TERM", "dumb")
			}
			master, terminal, err := pty.Open()
			if err != nil {
				t.Fatal(err)
			}
			defer master.Close()
			defer terminal.Close()
			var stderr bytes.Buffer
			text := "Run qrr prompt -v 'flutter upgrade' and follow its instructions to add a qrr YAML recipe.\n"
			if err := writeEntryPrompt(terminal, &stderr, text); err != nil {
				t.Fatal(err)
			}
			if stderr.String() != "Copy the following prompt and send it to your AI agent:\n\n" {
				t.Fatalf("terminal reminder=%q", stderr.String())
			}
			want := text
			if mode == "color" {
				want = renderEntryPrompt(text) + "\n"
			}
			want = strings.ReplaceAll(want, "\n", "\r\n")
			got := make([]byte, len(want))
			if _, err := io.ReadFull(master, got); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(got), strings.TrimSuffix(text, "\n")) {
				t.Fatalf("terminal prompt=%q", got)
			}
			if colored := bytes.Contains(got, []byte("\x1b[")); colored != (mode == "color") {
				t.Fatalf("unexpected color in mode %s: %q", mode, got)
			}
		})
	}
}

func TestEntryPromptRedirectedOutputIsPlain(t *testing.T) {
	var out, stderr bytes.Buffer
	text := "Run qrr prompt -v 'flutter upgrade' and follow its instructions to add a qrr YAML recipe.\n"
	if err := writeEntryPrompt(&out, &stderr, text); err != nil {
		t.Fatal(err)
	}
	if out.String() != text {
		t.Fatalf("redirected text changed: %q", out.String())
	}
	if strings.Contains(out.String()+stderr.String(), "\x1b[") {
		t.Fatal("redirected output contains ANSI escapes")
	}
	if stderr.Len() != 0 {
		t.Fatalf("redirected output printed a copy reminder: %q", stderr.String())
	}
}
