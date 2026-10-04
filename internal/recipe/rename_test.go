package recipe

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRenamePreservesRecipe(t *testing.T) {
	for _, ext := range []string{".yaml", ".yml"} {
		t.Run(ext, func(t *testing.T) {
			dir := t.TempDir()
			store := NewStore(dir)
			if _, err := store.Create("demo"); err != nil {
				t.Fatal(err)
			}
			original := filepath.Join(dir, "commands", "demo"+ext)
			if ext == ".yml" {
				if err := os.Rename(filepath.Join(dir, "commands", "demo.yaml"), original); err != nil {
					t.Fatal(err)
				}
			}
			data := `# Shared recipe
version: 1
name: demo # Recipe name
description: Keep demo in the description
type: command
params:
  - {name: text, type: input, default: 'demo'}
command: [echo, '{{ .text }}', demo]
optional_args:
  - {when: text, args: [--text, '{{ .text }}']}
args_tail: [--, '{{ .text }}']
`
			if err := os.WriteFile(original, []byte(data), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(original, 0640); err != nil {
				t.Fatal(err)
			}
			before, err := store.Validate("demo")
			if err != nil {
				t.Fatal(err)
			}
			if err := store.Rename("demo", "greet"); err != nil {
				t.Fatal(err)
			}
			after, err := store.Validate("greet")
			if err != nil {
				t.Fatal(err)
			}
			before.Name = "greet"
			if !reflect.DeepEqual(before, after) {
				t.Fatalf("recipe changed: %#v", after)
			}
			argv, err := after.Prepare(context.Background(), nil, nil)
			if err != nil || !reflect.DeepEqual(argv, []string{"echo", "demo", "demo", "--text", "demo", "--", "demo"}) {
				t.Fatalf("argv=%v err=%v", argv, err)
			}
			path, err := store.Path("greet")
			if err != nil || filepath.Ext(path) != ext {
				t.Fatalf("path=%q err=%v", path, err)
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != 0640 {
				t.Fatalf("permissions changed: %v", info.Mode())
			}
			if _, err := os.Stat(original); !os.IsNotExist(err) {
				t.Fatalf("old file remains: %v", err)
			}
			content, err := store.Read("greet")
			if err != nil {
				t.Fatal(err)
			}
			for _, comment := range []string{"# Shared recipe", "# Recipe name"} {
				if !strings.Contains(string(content), comment) {
					t.Fatalf("lost comment %q", comment)
				}
			}
			entries, err := os.ReadDir(filepath.Dir(path))
			if err != nil || len(entries) != 1 {
				t.Fatalf("temporary files remain: %v %v", entries, err)
			}
		})
	}
}

func TestRenameFailuresLeaveFilesUnchanged(t *testing.T) {
	for _, tc := range []struct {
		name, oldName, newName string
		files                  map[string]string
	}{
		{"missing", "missing", "new", nil},
		{"invalid source", "../demo", "new", nil},
		{"invalid target", "demo", "../new", nil},
		{"reserved target", "demo", "rename", nil},
		{"same name", "demo", "demo", nil},
		{"yaml collision", "demo", "new", map[string]string{"new.yaml": "invalid: true"}},
		{"yml collision", "demo", "new", map[string]string{"new.yml": "invalid: true"}},
		{"duplicate source", "demo", "new", map[string]string{"demo.yml": minimal}},
		{"invalid YAML", "demo", "new", map[string]string{"demo.yaml": "invalid: true"}},
		{"mismatched name", "demo", "new", map[string]string{"demo.yaml": strings.ReplaceAll(minimal, "name: demo", "name: other")}},
		{"name alias affects argv", "demo", "new", map[string]string{"demo.yaml": "version: 1\nname: &name demo\ntype: command\ncommand: [echo, *name]\n"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			store := NewStore(dir)
			path, err := store.Create("demo")
			if err != nil {
				t.Fatal(err)
			}
			for name, content := range tc.files {
				if err := os.WriteFile(filepath.Join(filepath.Dir(path), name), []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
			}
			snapshot := func() map[string]string {
				t.Helper()
				entries, err := os.ReadDir(filepath.Dir(path))
				if err != nil {
					t.Fatal(err)
				}
				files := map[string]string{}
				for _, entry := range entries {
					data, err := os.ReadFile(filepath.Join(filepath.Dir(path), entry.Name()))
					if err != nil {
						t.Fatal(err)
					}
					files[entry.Name()] = string(data)
				}
				return files
			}
			before := snapshot()
			if err := store.Rename(tc.oldName, tc.newName); err == nil {
				t.Fatal("expected rename failure")
			}
			if after := snapshot(); !reflect.DeepEqual(before, after) {
				t.Fatalf("files changed: %v", after)
			}
		})
	}
}

func TestRenameRejectsSymlinks(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	path, err := store.Create("demo")
	if err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(filepath.Dir(path), "linked.yaml")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if err := store.Rename("linked", "new"); err == nil {
		t.Fatal("renamed symlink")
	}
	if err := os.Symlink(filepath.Join(dir, "missing"), filepath.Join(filepath.Dir(path), "new.yml")); err != nil {
		t.Fatal(err)
	}
	if err := store.Rename("demo", "new"); err == nil {
		t.Fatal("ignored dangling target symlink")
	}
}
