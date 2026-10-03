package prompt

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

func TestSelectorFilteringNavigationAndEmptyResults(t *testing.T) {
	var value string
	s := newCommandSelector([]huh.Option[string]{huh.NewOption("hello  Greeting", "hello"), huh.NewOption("download  Greeting", "download")}, &value)
	s.WithTheme(huh.ThemeFunc(huh.ThemeCharm))
	s.WithKeyMap(huh.NewDefaultKeyMap())
	s.Focus()
	send := func(msg tea.KeyPressMsg) {
		t.Helper()
		model, _ := s.Update(msg)
		if model != s {
			t.Fatal("selector adapter lost during update")
		}
	}
	send(tea.KeyPressMsg{Code: 'G', Text: "GREETING"})
	send(tea.KeyPressMsg{Code: tea.KeyDown})
	if v, ok := s.Hovered(); !ok || v != "download" {
		t.Fatalf("arrow navigation failed: %q %v", v, ok)
	}
	send(tea.KeyPressMsg{Code: 'G', Text: "missing"})
	if _, ok := s.Hovered(); ok {
		t.Fatal("unexpected match")
	}
	send(tea.KeyPressMsg{Code: tea.KeyUp})
	_, cmd := s.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("empty filter submitted a stale selection")
	}

	if !strings.Contains(s.View(), "No matching commands") {
		t.Fatal("missing empty-state hint")
	}
	for i := 0; i < len("GREETINGmissing"); i++ {
		send(tea.KeyPressMsg{Code: tea.KeyBackspace})
	}
	if _, ok := s.Hovered(); !ok {
		t.Fatal("clearing filter did not restore list")
	}
	_, cmd = s.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter did not submit selection")
	}
}

func TestSelectorKeepsVisibleRowsUntilBoundary(t *testing.T) {
	for _, count := range []int{3, 12} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			var value string
			var options []huh.Option[string]
			for i := 0; i < count; i++ {
				name := fmt.Sprintf("recipe-%02d", i)
				options = append(options, huh.NewOption(name, name))
			}
			s := newCommandSelector(options, &value)
			s.WithTheme(huh.ThemeFunc(huh.ThemeCharm))
			s.WithWidth(80)
			s.WithHeight(6)
			s.WithKeyMap(huh.NewDefaultKeyMap())
			s.Focus()
			s.View() // Two header rows leave four visible recipe rows.
			if !strings.Contains(s.View(), "recipe-00") {
				t.Fatal("initial list scrolled")
			}
			for i := 1; i < min(count, 4); i++ {
				s.Update(tea.KeyPressMsg{Code: tea.KeyDown})
				view := s.View()
				if !strings.Contains(view, "recipe-00") || value != fmt.Sprintf("recipe-%02d", i) {
					t.Fatalf("list moved before boundary: value=%s", value)
				}
			}
			if count > 4 {
				s.Update(tea.KeyPressMsg{Code: tea.KeyDown})
				view := s.View()
				if strings.Contains(view, "recipe-00") || !strings.Contains(view, "recipe-04") {
					t.Fatalf("did not scroll at bottom: %q", view)
				}
				for i := 0; i < 3; i++ {
					s.Update(tea.KeyPressMsg{Code: tea.KeyUp})
					s.View()
					if strings.Contains(s.View(), "recipe-00") {
						t.Fatal("scrolled while selection was still visible")
					}
				}
				s.Update(tea.KeyPressMsg{Code: tea.KeyUp})
				s.View()
				if !strings.Contains(s.View(), "recipe-00") {
					t.Fatal("did not scroll at top boundary")
				}
			}
		})
	}
}
func TestSelectorFilterAndResizeResetViewport(t *testing.T) {
	var value string
	var options []huh.Option[string]
	for i := 0; i < 12; i++ {
		name := fmt.Sprintf("recipe-%02d", i)
		options = append(options, huh.NewOption(name, name))
	}
	s := newCommandSelector(options, &value)
	s.WithTheme(huh.ThemeFunc(huh.ThemeCharm))
	s.WithHeight(6)
	s.WithKeyMap(huh.NewDefaultKeyMap())
	s.Focus()
	for i := 0; i < 8; i++ {
		s.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		s.View()
	}
	if strings.Contains(s.View(), "recipe-00") {
		t.Fatal("test did not reach scrolled list")
	}
	s.WithHeight(20)
	s.View()
	if !strings.Contains(s.View(), "recipe-00") {
		t.Fatal("larger viewport did not restore full list")
	}
	s.Update(tea.KeyPressMsg{Code: 'G', Text: "recipe-00"})
	s.View()
	if hovered, ok := s.Hovered(); !ok || hovered != "recipe-00" {
		t.Fatal("filter kept stale selection or scroll offset")
	}
}
