package recipe

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestStoreRawFilesRemainRepairable(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	path, err := store.Create("broken")
	if err != nil {
		t.Fatal(err)
	}
	pathYML := strings.TrimSuffix(path, ".yaml") + ".yml"
	if err := os.Rename(path, pathYML); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pathYML, []byte("invalid: true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if got, err := store.Path("broken"); err != nil || got != pathYML {
		t.Fatalf("got %q, %v", got, err)
	}
	if got, err := store.Read("broken"); err != nil || string(got) != "invalid: true\n" {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := store.Validate("broken"); err == nil {
		t.Fatal("accepted invalid YAML")
	}
	if recipes, problems := store.Load(); len(recipes) != 0 || len(problems) != 1 {
		t.Fatalf("got %v, %v", recipes, problems)
	}
	if _, err := store.Create("broken"); err == nil {
		t.Fatal("overwrote .yml file")
	}
	if err := store.Remove("broken"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Path("broken"); err == nil {
		t.Fatal("removed file still found")
	}
}

func TestStoreCreationAndNameRules(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	if recipes, problems := store.Load(); len(recipes) != 0 || len(problems) != 0 {
		t.Fatalf("missing directory: %v %v", recipes, problems)
	}
	for _, name := range []string{"../escape", "help", "UPPER", ""} {
		if _, err := store.Create(name); err == nil {
			t.Fatalf("created invalid name %q", name)
		}
		if _, err := store.Path(name); err == nil {
			t.Fatalf("resolved invalid name %q", name)
		}
		if err := store.Remove(name); err == nil {
			t.Fatalf("removed invalid name %q", name)
		}
	}
	path, err := store.Create("demo")
	if err != nil {
		t.Fatal(err)
	}
	r, err := store.Validate("demo")
	if err != nil || r.Name != "demo" {
		t.Fatalf("created invalid template: %v %v", r, err)
	}
	for path, want := range map[string]os.FileMode{path: 0600, filepath.Dir(path): 0700} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != want {
			t.Fatalf("%s mode=%v, want %v", path, info.Mode().Perm(), want)
		}
	}
	if _, err := store.Create("demo"); err == nil {
		t.Fatal("overwrote existing recipe")
	}
}

func TestStoreConsistentFilenameValidationAndDiscovery(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	for _, name := range []string{"demo", "demo-extra"} {
		if _, err := store.Create(name); err != nil {
			t.Fatal(err)
		}
	}
	commands := filepath.Join(dir, "commands")
	if err := os.WriteFile(filepath.Join(commands, "wrong.yaml"), []byte(minimal), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Validate("wrong"); err == nil || !strings.Contains(err.Error(), "name must match filename") {
		t.Fatalf("got %v", err)
	}
	if err := os.WriteFile(filepath.Join(commands, "demo.yml"), []byte(minimal), 0600); err != nil {
		t.Fatal(err)
	}
	if got, err := store.Path("demo"); err != nil || filepath.Ext(got) != ".yaml" {
		t.Fatalf("extension preference: %q %v", got, err)
	}
	recipes, problems := store.Load()
	var names []string
	for _, r := range recipes {
		names = append(names, r.Name)
	}
	if !reflect.DeepEqual(names, []string{"demo", "demo-extra"}) || len(problems) != 2 {
		t.Fatalf("got %v, %v", names, problems)
	}
	if err := os.WriteFile(filepath.Join(commands, "new.yaml"), []byte(strings.ReplaceAll(minimal, "name: demo", "name: new")), 0600); err != nil {
		t.Fatal(err)
	}
	recipes, _ = store.Load()
	if len(recipes) != 3 {
		t.Fatal("store cached stale recipes")
	}
}

func TestStoreDoesNotReplaceDanglingSymlink(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	if _, err := store.Create("demo"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "commands", "dangling.yml")
	if err := os.Symlink(filepath.Join(dir, "missing"), path); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create("dangling"); err == nil {
		t.Fatal("ignored existing dangling symlink")
	}
}
