package prompt

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

const selectorDescription = "Type to filter."

// commandSelector keeps search active and prevents submission of empty results.
// Huh owns navigation, rendering, and viewport scrolling.
type commandSelector struct {
	*huh.Select[string]
	input         *huh.Input
	options       []huh.Option[string]
	value         *string
	rename        func(string, string) error
	oldName       string
	newName       string
	renameError   error
	renaming      bool
	theme         huh.Theme
	keymap        *huh.KeyMap
	width         int
	height        int
	position      huh.FieldPosition
	editRequested bool
}

func newCommandSelector(options []huh.Option[string], value *string) *commandSelector {
	s := &commandSelector{
		Select: huh.NewSelect[string]().Options(options...).Filtering(true).Title("Commands").
			Description(selectorDescription).Height(10).Value(value),
		options: options,
		value:   value,
		height:  10,
		theme:   huh.ThemeFunc(huh.ThemeCharm),
	}
	s.input = huh.NewInput().Title("Rename command").Description("Enter to save. Esc to return. Ctrl+C to cancel.").Value(&s.newName)
	return s
}

func (s *commandSelector) Update(msg tea.Msg) (huh.Model, tea.Cmd) {
	if s.renaming {
		if key, ok := msg.(tea.KeyPressMsg); ok {
			switch key.String() {
			case "esc":
				s.renaming = false
				s.renameError = nil
				s.input.Blur()
				return s, s.Select.Focus()
			case "enter":
				if err := s.rename(s.oldName, s.newName); err != nil {
					s.renameError = err
					return s, nil
				}
				for i, option := range s.options {
					if option.Value == s.oldName {
						s.options[i] = huh.NewOption(s.newName+strings.TrimPrefix(option.Key, s.oldName), s.newName)
					}
				}
				// Clear the old search so the renamed command remains visible.
				*s.value = s.newName
				s.Select = huh.NewSelect[string]().Options(s.options...).Filtering(true).Title("Commands").Description(selectorDescription).Value(s.value)
				s.WithTheme(s.theme)
				s.WithKeyMap(s.keymap)
				s.WithWidth(s.width)
				s.WithHeight(s.height)
				s.WithPosition(s.position)
				s.renaming = false
				s.renameError = nil
				s.input.Blur()
				return s, s.Select.Focus()
			case "tab", "shift+tab":
				return s, nil
			}
			s.renameError = nil
		}
		_, cmd := s.input.Update(msg)
		return s, cmd
	}
	if key, ok := msg.(tea.KeyPressMsg); ok {
		if key.String() == "ctrl+e" {
			name, found := s.Hovered()
			if !found {
				return s, nil
			}
			*s.value = name
			s.editRequested = true
			return s, huh.NextField
		}
		if key.String() == "ctrl+r" {
			name, found := s.Hovered()
			if !found || s.rename == nil {
				return s, nil
			}
			s.oldName, s.newName = name, name
			s.input.Title("Rename " + name).Value(&s.newName)
			s.renaming = true
			s.Select.Blur()
			return s, s.input.Focus()
		}
		if _, found := s.Hovered(); !found {
			switch key.String() {
			case "up", "down", "left", "right", "ctrl+p", "ctrl+n", "ctrl+k", "ctrl+j", "enter", "tab", "shift+tab":
				return s, nil
			}
		}
	}
	_, cmd := s.Select.Update(msg)
	return s, cmd
}

func (s *commandSelector) View() string {
	if s.renaming {
		view := s.input.View()
		if s.renameError != nil {
			view += "\n  " + s.renameError.Error()
		}
		return view
	}
	view := s.Select.View()
	if _, ok := s.Hovered(); !ok {
		lines := strings.Split(view, "\n")
		if len(lines) > 2 {
			lines[2] = "  No matching commands"
		} else {
			lines = append(lines, "  No matching commands")
		}
		return strings.Join(lines, "\n")
	}
	return view
}

func (s *commandSelector) WithKeyMap(k *huh.KeyMap) huh.Field {
	s.keymap = k
	s.input.WithKeyMap(k)
	k.Select.Next.SetEnabled(false)
	k.Select.Prev.SetEnabled(false)
	k.Select.Filter.SetEnabled(false)
	k.Select.SetFilter.SetEnabled(false)
	k.Select.ClearFilter.SetEnabled(false)
	s.Select.WithKeyMap(k)
	return s
}

func (s *commandSelector) WithTheme(theme huh.Theme) huh.Field {
	s.theme = theme
	s.Select.WithTheme(theme)
	s.input.WithTheme(theme)
	return s
}

func (s *commandSelector) WithWidth(width int) huh.Field {
	s.width = width
	s.Select.WithWidth(width)
	s.input.WithWidth(width)
	return s
}

func (s *commandSelector) WithHeight(height int) huh.Field {
	s.height = height
	s.Select.WithHeight(height)
	s.input.WithHeight(height)
	return s
}

func (s *commandSelector) WithPosition(position huh.FieldPosition) huh.Field {
	s.position = position
	s.Select.WithPosition(position)
	s.input.WithPosition(position)
	return s
}

func (s *commandSelector) KeyBinds() []key.Binding {
	if s.renaming {
		return []key.Binding{
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "save")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
			key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
		}
	}
	return []key.Binding{
		key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
		key.NewBinding(key.WithKeys("ctrl+r"), key.WithHelp("ctrl+r", "rename")),
		key.NewBinding(key.WithKeys("ctrl+e"), key.WithHelp("ctrl+e", "edit")),
	}
}
