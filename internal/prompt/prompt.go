// Package prompt provides inline terminal forms.
package prompt

import (
	"context"
	"errors"
	"fmt"
	"os"

	"qrr/internal/recipe"

	"charm.land/huh/v2"
	"golang.org/x/term"
)

func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}
func Choose(ctx context.Context, recipes []*recipe.Recipe, rename func(string, string) error) (*recipe.Recipe, error) {
	if len(recipes) == 0 {
		return nil, fmt.Errorf("no commands configured; use qrr add <name> to create one")
	}
	var name string
	options := make([]huh.Option[string], 0, len(recipes))
	for _, r := range recipes {
		options = append(options, huh.NewOption(r.Name+"  "+r.Description, r.Name))
	}
	field := newCommandSelector(options, &name)
	field.rename = func(oldName, newName string) error {
		if err := rename(oldName, newName); err != nil {
			return err
		}
		for _, r := range recipes {
			if r.Name == oldName {
				r.Name = newName
				break
			}
		}
		return nil
	}
	if err := runForm(ctx, field); err != nil {
		return nil, err
	}
	for _, r := range recipes {
		if r.Name == name {
			return r, nil
		}
	}
	return nil, fmt.Errorf("no command selected")
}
func Ask(ctx context.Context, p recipe.Param, value any) (any, error) {
	title := p.Prompt
	if title == "" {
		title = p.Name
	}
	var field huh.Field
	switch p.Type {
	case "input":
		v := value.(string)
		field = huh.NewInput().Title(title).Value(&v).Validate(func(s string) error { return recipe.ValidateValue(p, s) })
		err := runForm(ctx, field)
		return v, err
	case "select":
		v := value.(string)
		var opts []huh.Option[string]
		for _, o := range p.Options {
			label := o.Label
			if label == "" {
				label = o.Value
			}
			opts = append(opts, huh.NewOption(label, o.Value))
		}
		field = huh.NewSelect[string]().Title(title).Options(opts...).Value(&v).Validate(func(s string) error { return recipe.ValidateValue(p, s) })
		err := runForm(ctx, field)
		return v, err
	case "multiselect":
		v := value.([]string)
		var opts []huh.Option[string]
		for _, o := range p.Options {
			label := o.Label
			if label == "" {
				label = o.Value
			}
			opts = append(opts, huh.NewOption(label, o.Value))
		}
		field = huh.NewMultiSelect[string]().Title(title).Options(opts...).Value(&v).Validate(func(s []string) error { return recipe.ValidateValue(p, s) })
		err := runForm(ctx, field)
		return v, err
	case "confirm":
		v := value.(bool)
		field = huh.NewConfirm().Title(title).Value(&v).Validate(func(b bool) error { return recipe.ValidateValue(p, b) })
		err := runForm(ctx, field)
		return v, err
	}
	return nil, fmt.Errorf("unsupported parameter type")
}

// runForm keeps terminal-library cancellation errors inside the prompt module.
func runForm(ctx context.Context, field huh.Field) error {
	err := huh.NewForm(huh.NewGroup(field)).WithAccessible(false).RunWithContext(ctx)
	if errors.Is(err, huh.ErrUserAborted) {
		return context.Canceled
	}
	return err
}
