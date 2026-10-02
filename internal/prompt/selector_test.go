package prompt

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

func TestSelectorFilteringNavigationAndEmptyResults(t *testing.T) {
	var value string
	field := huh.NewSelect[string]().Options(huh.NewOption("hello  Greeting", "hello"), huh.NewOption("download  Greeting", "download")).Filtering(true).Value(&value)
	s := &commandSelector{field}
	s.WithTheme(huh.ThemeCharm())
	s.WithKeyMap(huh.NewDefaultKeyMap())
	s.Focus()
	send := func(msg tea.KeyMsg) {
		t.Helper()
		model, _ := s.Update(msg)
		if model != s {
			t.Fatal("selector adapter lost during update")
		}
	}
	send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("GREETING")})
	send(tea.KeyMsg{Type: tea.KeyDown})
	if v, ok := s.Hovered(); !ok || v != "download" {
		t.Fatalf("arrow navigation failed: %q %v", v, ok)
	}
	send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("missing")})
	if _, ok := s.Hovered(); ok {
		t.Fatal("unexpected match")
	}
	send(tea.KeyMsg{Type: tea.KeyUp})
	_, cmd := s.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("empty filter submitted a stale selection")
	}
	if !strings.Contains(s.View(), "No matching commands") {
		t.Fatal("missing empty-state hint")
	}
	for i := 0; i < len("GREETINGmissing"); i++ {
		send(tea.KeyMsg{Type: tea.KeyBackspace})
	}
	if _, ok := s.Hovered(); !ok {
		t.Fatal("clearing filter did not restore list")
	}
	_, cmd = s.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter did not submit selection")
	}
}
