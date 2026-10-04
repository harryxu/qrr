package recipe

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"go.yaml.in/yaml/v3"
)

// Rename updates a valid recipe's name and filename, retaining its extension,
// comments, permissions, and execution behavior. Existing targets are never replaced.
func (s *Store) Rename(oldName, newName string) error {
	if !ValidName(newName) {
		return fmt.Errorf("invalid or reserved command name %q", newName)
	}
	path, err := s.Path(oldName)
	if err != nil {
		return err
	}
	if err := s.requireAvailable(newName); err != nil {
		return err
	}
	// Two files with the old name would leave an old command after renaming.
	for _, ext := range []string{".yaml", ".yml"} {
		other := filepath.Join(s.dir, oldName+ext)
		if other == path {
			continue
		}
		if _, err := os.Lstat(other); err == nil {
			return fmt.Errorf("command %q has both .yaml and .yml files; remove the duplicate before renaming", oldName)
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("cannot rename non-regular recipe file %q", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	r, err := Parse(data)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if r.Name != oldName {
		return fmt.Errorf("name must match filename")
	}
	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return err
	}
	mapping := document.Content[0]
	for i := 0; i < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == "name" {
			name := mapping.Content[i+1]
			name.Kind, name.Tag, name.Value, name.Alias = yaml.ScalarNode, "!!str", newName, nil
			break
		}
	}
	var rendered bytes.Buffer
	encoder := yaml.NewEncoder(&rendered)
	encoder.SetIndent(2)
	if err := encoder.Encode(&document); err != nil {
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}
	renamed, err := Parse(rendered.Bytes())
	if err != nil {
		return err
	}
	r.Name = newName
	if !reflect.DeepEqual(r, renamed) {
		return fmt.Errorf("renaming would change other recipe values; remove aliases referencing the name first")
	}
	temporary, err := os.CreateTemp(s.dir, ".qrr-rename-*")
	if err != nil {
		return err
	}
	defer os.Remove(temporary.Name())
	defer temporary.Close()
	if err := temporary.Chmod(info.Mode().Perm()); err != nil {
		return err
	}
	if _, err := temporary.Write(rendered.Bytes()); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	target := filepath.Join(s.dir, newName+filepath.Ext(path))
	// Link publishes the complete file and fails if the target appeared meanwhile.
	if err := os.Link(temporary.Name(), target); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return errors.Join(err, os.Remove(target))
	}
	return nil
}
