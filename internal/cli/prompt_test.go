package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"qrr/internal/recipe"

	"github.com/google/shlex"
)

func TestPromptPreservesTargetArguments(t *testing.T) {
	dir := fixture(t)
	target := []string{"yt-dlp", "-f", "bestvideo[height<=2160]+bestaudio/best[height<=2160]", "--write-subs", "--write-auto-subs", "--sub-langs", "zh-Hans,ja,en", "--embed-subs", "--sleep-subtitles", "60", "https://www.youtube.com/watch?v=cD-Z2A2zuiY", "--config-dir", "target config", "--dry-run", "--non-interactive", "--json", "--help", "", "a b '世界' $(echo test)"}
	out, stderr, err := invoke(dir, append([]string{"prompt"}, target...)...)
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := strings.CutSuffix(strings.TrimSpace(out), " and follow its instructions to add a qrr YAML recipe.")
	if !ok {
		t.Fatalf("unexpected entry prompt: %q", out)
	}
	invocation, err := shlex.Split(strings.TrimPrefix(entry, "Run "))
	if err != nil {
		t.Fatal(err)
	}
	if len(invocation) != 4 || !reflect.DeepEqual(invocation[:3], []string{"qrr", "prompt", "-v"}) {
		t.Fatalf("unexpected entry command: %#v", invocation)
	}
	got, err := shlex.Split(invocation[3])
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, target) {
		t.Fatalf("target arguments changed: %#v", got)
	}
	for _, detail := range []string{dir, "schema", "qrr add --help"} {
		if strings.Contains(out, detail) {
			t.Fatalf("entry contains authoring detail %q", detail)
		}
	}
	verbose, _, err := invoke(dir, "prompt", "-v", invocation[3])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(verbose, invocation[3]) {
		t.Fatal("detailed prompt lost command")
	}
	for _, want := range []string{"qrr --help", "schema", "qrr add --help", "validate", "dry-run", Display([]string{dir})} {
		if !strings.Contains(verbose, want) {
			t.Fatalf("missing %q in detailed prompt", want)
		}
	}
	if !strings.Contains(stderr, "Copy the following prompt") {
		t.Fatal("missing copy reminder")
	}
	if len(strings.Split(strings.TrimSpace(out), "\n")) != 1 {
		t.Fatal("prompt should be a single paragraph")
	}
}

func TestPromptDoesNotExecuteOrCreateRecipes(t *testing.T) {
	dir := fixture(t)
	marker := filepath.Join(t.TempDir(), "executed")
	exe := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\ntouch '"+marker+"'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadDir(filepath.Join(dir, "commands"))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := invoke(dir, "prompt", exe); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("prompt executed the target command")
	}
	after, err := os.ReadDir(filepath.Join(dir, "commands"))
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != len(after) {
		t.Fatal("prompt created a recipe")
	}
}

func TestPromptHelpAndMissingCommand(t *testing.T) {
	dir := fixture(t)
	for _, args := range [][]string{{"prompt"}, {"prompt", "--"}, {"prompt", ""}, {"prompt", "--unknown"}} {
		if _, _, err := invoke(dir, args...); err == nil {
			t.Fatalf("expected missing-command error for %v", args)
		}
	}
	for _, args := range [][]string{{"prompt", "--help"}, {"prompt", "-h"}, {"help", "prompt"}} {
		out, _, err := invoke(dir, args...)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out, "prompt [-v] <command>") {
			t.Fatalf("missing prompt help: %q", out)
		}
	}
	out, _, err := invoke(dir, "prompt", "--", "echo", "--help")
	if err != nil || !strings.Contains(out, "echo --help") {
		t.Fatalf("separator changed arguments: %q %v", out, err)
	}
}

func TestSchemaHelpWorksWithInvalidConfiguration(t *testing.T) {
	dir := fixture(t)
	if err := os.WriteFile(filepath.Join(dir, "commands", "broken.yaml"), []byte("invalid: true"), 0600); err != nil {
		t.Fatal(err)
	}
	out, stderr, err := invoke(dir, "schema")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{filepath.Join(dir, "commands"), "version: 1", "type: command", "optional_args", "args_tail", "required", "validate <name>", "--dry-run --json"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing schema guidance: %q", want)
		}
	}
	if stderr != "" {
		t.Fatalf("schema help polluted: %q", stderr)
	}
	if _, stderr, err := invoke(dir, "prompt", "echo", "hello"); err != nil || strings.Contains(stderr, "Warning:") {
		t.Fatalf("prompt depends on valid configuration: %q %v", stderr, err)
	}
	for _, name := range []string{"schema", "prompt"} {
		if recipe.ValidName(name) {
			t.Fatalf("%s must be reserved", name)
		}
		if _, _, err := invoke(dir, "add", name); err == nil {
			t.Fatalf("allowed reserved name %q", name)
		}
	}
}

func TestConfigDirStopsAtPromptBoundary(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{{"--config-dir", dir, "prompt", "echo", "--config-dir", "target"}, {"--config-dir=" + dir, "prompt", "echo", "--config-dir="}, {"--config-dir", dir, "__complete", "prompt", "echo", "--config-dir", "target"}} {
		got, err := ConfigDir(args)
		if err != nil || got != dir {
			t.Fatalf("wrong qrr config for %v: %q %v", args, got, err)
		}
	}
}

func TestPromptEntryAndVerboseCommand(t *testing.T) {
	dir := fixture(t)
	out, _, err := invoke(dir, "prompt", "flutter", "upgrade")
	if err != nil {
		t.Fatal(err)
	}
	want := "Run qrr prompt -v 'flutter upgrade' and follow its instructions to add a qrr YAML recipe.\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
	for _, args := range [][]string{{"prompt", "-v", "flutter upgrade"}, {"prompt", "--verbose", "flutter upgrade"}, {"prompt", "-v", "flutter", "upgrade"}} {
		out, stderr, err := invoke(dir, args...)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out, "\n\nflutter upgrade\n\n") || !strings.Contains(out, "schema") {
			t.Fatalf("unexpected detailed prompt: %q", out)
		}
		if stderr != "" {
			t.Fatalf("verbose mode printed a copy reminder: %q", stderr)
		}
	}
	for _, args := range [][]string{{"prompt", "-v"}, {"prompt", "-v", ""}, {"prompt", "--verbose", "--"}} {
		if _, _, err := invoke(dir, args...); err == nil {
			t.Fatalf("accepted empty verbose command: %v", args)
		}
	}
	out, _, err = invoke(dir, "prompt", "flutter", "-v")
	if err != nil || !strings.Contains(out, "'flutter -v'") {
		t.Fatalf("target -v consumed: %q %v", out, err)
	}
}
