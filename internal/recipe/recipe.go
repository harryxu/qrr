// Package recipe loads and validates declarative command recipes.
package recipe

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"go.yaml.in/yaml/v3"
)

type Option struct {
	Label string `yaml:"label" json:"label"`
	Value string `yaml:"value" json:"value"`
}
type Param struct {
	Name           string   `yaml:"name"`
	Type           string   `yaml:"type"`
	Prompt         string   `yaml:"prompt"`
	Required       bool     `yaml:"required"`
	Default        any      `yaml:"default"`
	Options        []Option `yaml:"options,omitempty"`
	OptionsCommand []string `yaml:"options_command,omitempty"`
}
type OptionalArgs struct {
	When string   `yaml:"when"`
	Args []string `yaml:"args"`
}
type Recipe struct {
	Version      int            `yaml:"version"`
	Name         string         `yaml:"name"`
	Description  string         `yaml:"description"`
	Type         string         `yaml:"type"`
	Params       []Param        `yaml:"params"`
	Command      []string       `yaml:"command"`
	OptionalArgs []OptionalArgs `yaml:"optional_args"`
	ArgsTail     []string       `yaml:"args_tail"`
}

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
var reserved = map[string]bool{"run": true, "list": true, "show": true, "add": true, "edit": true, "rename": true, "remove": true, "validate": true, "completion": true, "help": true, "prompt": true, "schema": true}
var reservedFlags = map[string]bool{"help": true, "config-dir": true, "non-interactive": true, "dry-run": true, "json": true}

func ValidName(name string) bool { return namePattern.MatchString(name) && !reserved[name] }
func DefaultDir() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "qrr"), nil
	}
	home, err := os.UserHomeDir()
	return filepath.Join(home, ".config", "qrr"), err
}
func Parse(data []byte) (*Recipe, error) {
	d := yaml.NewDecoder(bytes.NewReader(data))
	d.KnownFields(true)
	var r Recipe
	if err := d.Decode(&r); err != nil {
		return nil, err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("expected exactly one YAML document")
	}
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return &r, nil
}
func DefaultValue(p Param) any {
	if p.Default != nil {
		if p.Type == "multiselect" {
			var values []string
			for _, v := range p.Default.([]any) {
				values = append(values, v.(string))
			}
			return values
		}
		return p.Default
	}
	switch p.Type {
	case "confirm":
		return false
	case "multiselect":
		return []string{}
	default:
		return ""
	}
}
func Empty(v any) bool {
	switch x := v.(type) {
	case string:
		return x == ""
	case []string:
		return len(x) == 0
	case bool:
		return !x
	default:
		return v == nil
	}
}
func ValidateValue(p Param, v any) error {
	if p.Required && Empty(v) {
		return fmt.Errorf("%s: a value is required", p.Name)
	}
	allowed := map[string]bool{}
	for _, o := range p.Options {
		allowed[o.Value] = true
	}
	switch p.Type {
	case "input", "select":
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("%s: expected a string", p.Name)
		}
		if p.Type == "select" && s != "" && len(p.OptionsCommand) == 0 && !allowed[s] {
			return fmt.Errorf("%s: unknown option %q", p.Name, s)
		}
	case "confirm":
		if _, ok := v.(bool); !ok {
			return fmt.Errorf("%s: expected a boolean", p.Name)
		}
	case "multiselect":
		values, ok := v.([]string)
		if !ok {
			return fmt.Errorf("%s: expected a string array", p.Name)
		}
		seen := map[string]bool{}
		for _, s := range values {
			if s == "" || strings.Contains(s, ",") || (len(p.OptionsCommand) == 0 && !allowed[s]) || seen[s] {
				return fmt.Errorf("%s: invalid or duplicate option %q", p.Name, s)
			}
			seen[s] = true
		}
	}
	return nil
}
func (r *Recipe) Validate() error {
	if r.Version != 1 {
		return fmt.Errorf("version must be 1")
	}
	if !ValidName(r.Name) {
		return fmt.Errorf("invalid or reserved command name %q", r.Name)
	}
	if r.Type != "command" {
		return fmt.Errorf("type must be command; handlers are not supported")
	}
	if len(r.Command) == 0 || strings.TrimSpace(r.Command[0]) == "" {
		return fmt.Errorf("command must contain an executable")
	}
	values := map[string]any{}
	for _, p := range r.Params {
		if !namePattern.MatchString(p.Name) || reservedFlags[p.Name] {
			return fmt.Errorf("invalid or reserved parameter name %q", p.Name)
		}
		if _, ok := values[p.Name]; ok {
			return fmt.Errorf("duplicate parameter %q", p.Name)
		}
		switch p.Type {
		case "input", "select", "multiselect", "confirm":
		default:
			return fmt.Errorf("%s: unknown parameter type %q", p.Name, p.Type)
		}
		if (p.Type == "select" || p.Type == "multiselect") && len(p.Options) == 0 && p.OptionsCommand == nil {
			return fmt.Errorf("%s: options or options_command are required", p.Name)
		}
		if p.Type != "select" && p.Type != "multiselect" && (len(p.Options) > 0 || p.OptionsCommand != nil) {
			return fmt.Errorf("%s: options are not supported for this type", p.Name)
		}
		if p.OptionsCommand != nil {
			if p.Options != nil {
				return fmt.Errorf("%s: options and options_command are mutually exclusive", p.Name)
			}
			if len(p.OptionsCommand) == 0 || strings.TrimSpace(p.OptionsCommand[0]) == "" {
				return fmt.Errorf("%s: options_command must contain an executable", p.Name)
			}
		}
		if err := validateOptions(p); err != nil {
			return err
		}
		if p.Default != nil {
			if p.Type == "multiselect" {
				a, ok := p.Default.([]any)
				if !ok {
					return fmt.Errorf("%s: default must be a string array", p.Name)
				}
				for _, v := range a {
					if _, ok := v.(string); !ok {
						return fmt.Errorf("%s: default must be a string array", p.Name)
					}
				}
			}
			check := p
			check.Required = false
			if err := ValidateValue(check, DefaultValue(p)); err != nil {
				return err
			}
		}
		values[p.Name] = DefaultValue(p)
	}
	for _, o := range r.OptionalArgs {
		if _, ok := values[o.When]; !ok {
			return fmt.Errorf("unknown condition parameter %q", o.When)
		}
	}
	// Render every branch during validation to detect missing references.
	parts := append([]string{}, r.Command...)
	for _, o := range r.OptionalArgs {
		parts = append(parts, o.Args...)
	}
	parts = append(parts, r.ArgsTail...)
	_, err := render(parts, values)
	return err
}

func validateOptions(p Param) error {
	seen := map[string]bool{}
	for _, o := range p.Options {
		if o.Value == "" || seen[o.Value] {
			return fmt.Errorf("%s: empty or duplicate option", p.Name)
		}
		if p.Type == "multiselect" && strings.Contains(o.Value, ",") {
			return fmt.Errorf("%s: multiselect option values must not contain commas", p.Name)
		}
		seen[o.Value] = true
	}
	return nil
}
func render(parts []string, values map[string]any) ([]string, error) {
	result := make([]string, 0, len(parts))
	for i, part := range parts {
		t, err := template.New("arg").Option("missingkey=error").Funcs(template.FuncMap{"join": strings.Join}).Parse(part)
		if err != nil {
			return nil, fmt.Errorf("argument %d: %w", i, err)
		}
		var b bytes.Buffer
		if err = t.Execute(&b, values); err != nil {
			return nil, fmt.Errorf("argument %d: %w", i, err)
		}
		result = append(result, b.String())
	}
	return result, nil
}
func (r *Recipe) Render(values map[string]any) ([]string, error) {
	parts := append([]string{}, r.Command...)
	for _, o := range r.OptionalArgs {
		if !Empty(values[o.When]) {
			parts = append(parts, o.Args...)
		}
	}
	parts = append(parts, r.ArgsTail...)
	args, err := render(parts, values)
	if err == nil && strings.TrimSpace(args[0]) == "" {
		return nil, fmt.Errorf("rendered executable is empty")
	}
	return args, err
}
