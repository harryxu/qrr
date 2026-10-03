package prompt

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

const selectorDescription = "Type to filter. Use up/down to navigate. Ctrl+C to cancel."

// commandSelector keeps search active and prevents submission of empty results.
// Huh owns navigation, rendering, and viewport scrolling.
type commandSelector struct {
	*huh.Select[string]
}

func newCommandSelector(options []huh.Option[string], value *string) *commandSelector {
	return &commandSelector{
		Select: huh.NewSelect[string]().Options(options...).Filtering(true).Title("Commands").
			Description(selectorDescription).Height(10).Value(value),
	}
}

func (s *commandSelector) Update(msg tea.Msg) (huh.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
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
	k.Select.Next.SetEnabled(false)
	k.Select.Prev.SetEnabled(false)
	k.Select.Filter.SetEnabled(false)
	k.Select.SetFilter.SetEnabled(false)
	k.Select.ClearFilter.SetEnabled(false)
	s.Select.WithKeyMap(k)
	return s
}
