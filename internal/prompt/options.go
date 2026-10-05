package prompt

import (
	"strings"

	"qrr/internal/recipe"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

func menuOptions(options []recipe.Option) []huh.Option[string] {
	labels := make([]string, len(options))
	counts := map[string]int{}
	for i, option := range options {
		label := option.Label
		if label == "" {
			label = option.Value
		}
		labels[i] = label
		counts[label]++
	}
	result := make([]huh.Option[string], 0, len(options))
	used := map[string]bool{}
	for i, option := range options {
		label := labels[i]
		// Huh's multiselect uses display keys when toggling. Make duplicate labels
		// distinguishable so toggling one value cannot toggle another option.
		if counts[label] > 1 {
			label += " [" + option.Value + "]"
		}
		for used[label] {
			label += " [" + option.Value + "]"
		}
		used[label] = true
		result = append(result, huh.NewOption(label, option.Value))
	}
	return result
}

// optionSelect keeps typing in the search field and rejects stale selections.
type optionSelect struct {
	*huh.Select[string]
}

func (s *optionSelect) Update(msg tea.Msg) (huh.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		if _, found := s.Hovered(); !found && selectionKey(key.String()) {
			return s, nil
		}
	}
	_, cmd := s.Select.Update(msg)
	return s, cmd
}

func (s *optionSelect) View() string {
	view := s.Select.View()
	if _, found := s.Hovered(); !found {
		view = emptyOptionsView(view)
	}
	return view
}

func (s *optionSelect) WithKeyMap(k *huh.KeyMap) huh.Field {
	k.Select.Next.SetEnabled(false)
	k.Select.Prev.SetEnabled(false)
	k.Select.Filter.SetEnabled(false)
	k.Select.SetFilter.SetEnabled(false)
	k.Select.ClearFilter.SetEnabled(false)
	s.Select.WithKeyMap(k)
	return s
}

// optionMultiSelect retains Huh's slash-to-filter and space-to-toggle controls.
type optionMultiSelect struct {
	*huh.MultiSelect[string]
}

func (s *optionMultiSelect) Update(msg tea.Msg) (huh.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		if _, found := s.Hovered(); !found && (selectionKey(key.String()) || (!s.GetFiltering() && key.String() == "space")) {
			return s, nil
		}
	}
	_, cmd := s.MultiSelect.Update(msg)
	return s, cmd
}

func (s *optionMultiSelect) View() string {
	view := s.MultiSelect.View()
	if _, found := s.Hovered(); !found {
		view = emptyOptionsView(view)
	}
	return view
}

func emptyOptionsView(view string) string {
	lines := strings.Split(view, "\n")
	// Replace a blank viewport row so the group's fixed height cannot clip the hint.
	lines[len(lines)-1] = "  No matching options"
	return strings.Join(lines, "\n")
}

func selectionKey(key string) bool {
	switch key {
	case "up", "down", "left", "right", "ctrl+p", "ctrl+n", "ctrl+k", "ctrl+j", "enter", "tab", "shift+tab":
		return true
	}
	return false
}
