package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func fixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "commands"), 0700); err != nil {
		t.Fatal(err)
	}
	data := `version: 1
name: demo
description: A test recipe
type: command
params:
  - {name: text, type: input, required: true}
  - {name: enabled, type: confirm, default: true}
  - name: languages
    type: multiselect
    default: [en]
    options: [{label: English, value: en}, {label: Japanese, value: ja}]
  - name: quality
    type: select
    default: high
    options: [{value: high}, {value: low}]
command: [echo, '{{ .text }}']
optional_args:
  - {when: enabled, args: [--enabled]}
  - {when: languages, args: [--languages, '{{ join .languages "," }}']}
`
	if err := os.WriteFile(filepath.Join(dir, "commands", "demo.yaml"), []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	return dir
}
func invoke(dir string, args ...string) (string, string, error) {
	var out, stderr bytes.Buffer
	err := Execute(context.Background(), append([]string{"--config-dir", dir}, args...), &out, &stderr)
	return out.String(), stderr.String(), err
}
func TestDryRunExplicitEmptyAndFalse(t *testing.T) {
	dir := fixture(t)
	for _, name := range [][]string{{"demo"}, {"run", "demo"}} {
		input := "a b 世界 ' $(echo unexpected)"
		args := append(append([]string{}, name...), "--text", input, "--enabled=false", "--languages=", "--non-interactive", "--dry-run", "--json")
		out, _, err := invoke(dir, args...)
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, []string{"echo", input}) {
			t.Fatalf("unexpected argv %#v", got)
		}
	}
}
func TestInvalidArgumentsFailWithoutExecution(t *testing.T) {
	dir := fixture(t)
	for _, args := range [][]string{{"demo", "--non-interactive"}, {"demo", "--text", "ok", "--quality", "missing", "--non-interactive"}, {"demo", "--text", "ok", "--languages", "en,en", "--non-interactive"}, {"demo", "--text", "ok", "--typo"}, {"--non-interactive"}} {
		if _, _, err := invoke(dir, args...); err == nil {
			t.Fatalf("expected error for %v", args)
		}
	}
}
func TestCompletionAndDynamicDiscovery(t *testing.T) {
	dir := fixture(t)
	for _, tc := range []struct {
		args []string
		want string
	}{{[]string{"__complete", "d"}, "demo"}, {[]string{"__complete", "run", "d"}, "demo"}, {[]string{"__complete", "demo", "--quality", "h"}, "high"}, {[]string{"__complete", "demo", "--languages", "en,j"}, "en,ja"}, {[]string{"__complete", "show", "d"}, "demo"}} {
		out, _, err := invoke(dir, tc.args...)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out, tc.want) {
			t.Fatalf("missing %q in %q", tc.want, out)
		}
	}
	if _, _, err := invoke(dir, "add", "new"); err != nil {
		t.Fatal(err)
	}
	out, _, err := invoke(dir, "__complete", "n")
	if err != nil || !strings.Contains(out, "new") {
		t.Fatalf("new recipe absent: %q %v", out, err)
	}
}
func TestManagementAndInvalidFile(t *testing.T) {
	dir := fixture(t)
	if _, _, err := invoke(dir, "add", "demo"); err == nil {
		t.Fatal("add overwrote existing recipe")
	}
	if _, _, err := invoke(dir, "add", "../escape"); err == nil {
		t.Fatal("accepted invalid name")
	}
	if _, _, err := invoke(dir, "remove", "demo", "--non-interactive"); err == nil {
		t.Fatal("removed without confirmation")
	}
	if err := os.WriteFile(filepath.Join(dir, "commands", "broken.yaml"), []byte("invalid: true"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := invoke(dir, "validate"); err == nil {
		t.Fatal("validate ignored invalid file")
	}
	out, stderr, err := invoke(dir, "__complete", "d")
	if err != nil || !strings.Contains(out, "demo") || strings.Contains(stderr, "Warning:") {
		t.Fatalf("completion polluted: %q %q %v", out, stderr, err)
	}
	if _, _, err := invoke(dir, "remove", "demo", "--yes"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "commands", "demo.yaml")); !os.IsNotExist(err) {
		t.Fatal("recipe was not deleted")
	}
}
