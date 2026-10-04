package recipe

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Store owns recipe file naming and storage. Each operation reads current files;
// raw lookup and reading do not require valid YAML, so broken recipes are repairable.
type Store struct {
	dir string
}

func NewStore(dir string) *Store { return &Store{dir: filepath.Join(dir, "commands")} }

// Path finds a raw recipe file, preferring .yaml over .yml.
func (s *Store) Path(name string) (string, error) {
	if !ValidName(name) {
		return "", fmt.Errorf("invalid command name %q", name)
	}
	for _, ext := range []string{".yaml", ".yml"} {
		path := filepath.Join(s.dir, name+ext)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
	}
	return "", fmt.Errorf("unknown command %q", name)
}

func (s *Store) Read(name string) ([]byte, error) {
	path, err := s.Path(name)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

// Validate checks one file without executing it or prompting.
func (s *Store) Validate(name string) (*Recipe, error) {
	path, err := s.Path(name)
	if err != nil {
		return nil, err
	}
	return parseFile(path)
}

func parseFile(path string) (*Recipe, error) {
	data, err := os.ReadFile(path)
	var r *Recipe
	if err == nil {
		r, err = Parse(data)
	}
	if err == nil && strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)) != r.Name {
		err = fmt.Errorf("name must match filename")
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return r, nil
}

// Load returns valid recipes in name order and reports invalid files separately.
func (s *Store) Load() ([]*Recipe, []error) {
	entries, err := os.ReadDir(s.dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, []error{err}
	}
	var recipes []*Recipe
	var problems []error
	seen := map[string]bool{}
	for _, entry := range entries {
		ext := filepath.Ext(entry.Name())
		if entry.IsDir() || (ext != ".yaml" && ext != ".yml") {
			continue
		}
		path := filepath.Join(s.dir, entry.Name())
		r, err := parseFile(path)
		if err == nil && seen[r.Name] {
			err = fmt.Errorf("%s: duplicate command %q", path, r.Name)
		}
		if err != nil {
			problems = append(problems, err)
			continue
		}
		seen[r.Name] = true
		recipes = append(recipes, r)
	}
	sort.Slice(recipes, func(i, j int) bool { return recipes[i].Name < recipes[j].Name })
	return recipes, problems
}

// Create writes a private example template without overwriting either extension.
func (s *Store) Create(name string) (string, error) {
	if !ValidName(name) {
		return "", fmt.Errorf("invalid or reserved command name %q", name)
	}
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return "", err
	}
	for _, ext := range []string{".yaml", ".yml"} {
		if _, err := os.Lstat(filepath.Join(s.dir, name+ext)); err == nil {
			return "", fmt.Errorf("command %q already exists", name)
		} else if !os.IsNotExist(err) {
			return "", err
		}
	}
	path := filepath.Join(s.dir, name+".yaml")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", err
	}
	_, writeErr := fmt.Fprintf(f, "version: 1\nname: %s\ndescription: An example command\ntype: command\nparams:\n  - name: message\n    type: input\n    prompt: Message\n    default: Hello from qrr\ncommand:\n  - echo\n  - '{{ .message }}'\n", name)
	closeErr := f.Close()
	if writeErr != nil {
		return "", writeErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	return path, nil
}

// Remove deletes a raw recipe file, including invalid YAML.
func (s *Store) Remove(name string) error {
	path, err := s.Path(name)
	if err != nil {
		return err
	}
	return os.Remove(path)
}
