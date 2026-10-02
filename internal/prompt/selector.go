package prompt

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

// commandSelector keeps Huh's keyboard handling and displays an empty-state message.
type commandSelector struct{ *huh.Select[string] }

func (s *commandSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		if _, found := s.Hovered(); !found {
			switch key.Type {
			case tea.KeyEnter, tea.KeyUp, tea.KeyDown, tea.KeyTab, tea.KeyShiftTab:
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
		view = strings.Join(lines, "\n")
	}
	return view
}

// Keep search active while navigating and reserve Enter for selection.
func (s *commandSelector) WithKeyMap(k *huh.KeyMap) huh.Field {
	k.Select.Next.SetEnabled(false)
	k.Select.Prev.SetEnabled(false)
	k.Select.Filter.SetEnabled(false)
	k.Select.SetFilter.SetEnabled(false)
	k.Select.ClearFilter.SetEnabled(false)
	s.Select.WithKeyMap(k)
	return s
}
