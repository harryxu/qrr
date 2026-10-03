package cli

import (
	"bytes"
	"strings"
	"testing"
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
}
