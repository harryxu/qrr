package cli

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestRenameCommandAndDynamicDiscovery(t *testing.T) {
	dir := fixture(t)
	out, _, err := invoke(dir, "__complete", "rename", "d")
	if err != nil || !strings.Contains(out, "demo") {
		t.Fatalf("source completion: %q %v", out, err)
	}
	out, _, err = invoke(dir, "__complete", "rename", "demo", "")
	if err != nil || strings.Contains(out, "demo") || !strings.Contains(out, ":4") {
		t.Fatalf("target completion: %q %v", out, err)
	}
	out, _, err = invoke(dir, "rename", "demo", "greet", "--non-interactive")
	if err != nil || out != "Renamed: demo -> greet\n" {
		t.Fatalf("rename: %q %v", out, err)
	}
	if _, _, err := invoke(dir, "validate"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := invoke(dir, "validate", "greet"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := invoke(dir, "demo", "--non-interactive"); err == nil {
		t.Fatal("old command still exists")
	}
	for _, command := range [][]string{{"greet"}, {"run", "greet"}} {
		args := append(command, "--text", "a b", "--enabled=false", "--languages=", "--non-interactive", "--dry-run", "--json")
		out, _, err := invoke(dir, args...)
		if err != nil {
			t.Fatal(err)
		}
		var argv []string
		if err := json.Unmarshal([]byte(out), &argv); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(argv, []string{"echo", "a b"}) {
			t.Fatalf("argv=%v", argv)
		}
	}
	for _, args := range [][]string{{"__complete", "g"}, {"__complete", "run", "g"}, {"__complete", "rename", "g"}, {"__complete", "greet", "--quality", "h"}} {
		out, _, err := invoke(dir, args...)
		want := "greet"
		if args[1] == "greet" {
			want = "high"
		}
		if err != nil || !strings.Contains(out, want) || strings.Contains(out, "demo") {
			t.Fatalf("completion after rename: %q %v", out, err)
		}
	}
}

func TestRenameCommandRejectsInvalidRequests(t *testing.T) {
	dir := fixture(t)
	before, _, err := invoke(dir, "show", "demo")
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"rename"}, {"rename", "demo"}, {"rename", "demo", "new", "extra"},
		{"rename", "demo", "rename"}, {"rename", "demo", "../new"},
		{"rename", "missing", "new"}, {"rename", "demo", "demo"},
		{"add", "rename"},
	} {
		if _, _, err := invoke(dir, args...); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	after, _, err := invoke(dir, "show", "demo")
	if err != nil || before != after {
		t.Fatalf("source changed: %q %v", after, err)
	}
}
