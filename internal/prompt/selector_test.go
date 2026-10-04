package prompt

import (
	"fmt"
	"strings"
	"testing"

	"qrr/internal/recipe"

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

func TestSelectorRenameAndReturn(t *testing.T) {
	var value string
	s := newCommandSelector([]huh.Option[string]{huh.NewOption("hello  Greeting", "hello"), huh.NewOption("download  Download", "download")}, &value)
	// Exercise the default theme used by runForm, including after rebuilding the list.
	s.WithKeyMap(huh.NewDefaultKeyMap())
	s.WithWidth(80)
	s.Focus()
	calls := 0
	s.rename = func(oldName, newName string) error {
		calls++
		if oldName != "download" || newName != "fetch" {
			t.Fatalf("rename received %q -> %q", oldName, newName)
		}
		return nil
	}
	s.Update(tea.KeyPressMsg{Code: 'd', Text: "download"})
	s.Update(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	if !s.renaming || s.newName != "download" {
		t.Fatal("rename did not prefill the highlighted command")
	}
	s.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if s.renaming || calls != 0 {
		t.Fatal("Escape saved a rename")
	}
	if hovered, ok := s.Hovered(); !ok || hovered != "download" || strings.Contains(s.View(), "hello  Greeting") {
		t.Fatal("Escape did not preserve the filter and selection")
	}
	s.Update(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	s.Update(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	s.Update(tea.KeyPressMsg{Code: 'f', Text: "fetch"})
	_, cmd := s.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("rename save submitted the form")
	}
	if s.renaming || calls != 1 || value != "fetch" {
		t.Fatalf("rename did not refresh selection: %q, calls=%d", value, calls)
	}
	// The save must focus the selector, rather than finish the form.
	if !strings.Contains(s.View(), "hello") || !strings.Contains(s.View(), "fetch  Download") {
		t.Fatal("rename did not clear the filter and return to the list")
	}
	s.Update(tea.KeyPressMsg{Code: 'x', Text: "missing"})
	s.Update(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	if s.renaming || calls != 1 {
		t.Fatal("empty search renamed a stale selection")
	}
}

func TestSelectorRenameErrorsAllowCorrection(t *testing.T) {
	store := recipe.NewStore(t.TempDir())
	if _, err := store.Create("hello"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create("taken"); err != nil {
		t.Fatal(err)
	}
	var value string
	s := newCommandSelector([]huh.Option[string]{huh.NewOption("hello", "hello")}, &value)
	s.WithTheme(huh.ThemeFunc(huh.ThemeCharm))
	s.WithKeyMap(huh.NewDefaultKeyMap())
	s.Focus()
	s.rename = store.Rename
	s.Update(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	for _, name := range []string{"hello", "", "../bad", "rename", "taken", "greet"} {
		s.input.Value(&s.newName)
		s.Update(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
		if name != "" {
			s.Update(tea.KeyPressMsg{Code: 'x', Text: name})
		}
		s.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		if name != "greet" {
			if !s.renaming || s.renameError == nil || !strings.Contains(s.View(), s.renameError.Error()) {
				t.Fatalf("missing inline error for %q", name)
			}
			if _, err := store.Path("hello"); err != nil {
				t.Fatal("failed rename removed source")
			}
		}
	}
	if s.renaming || value != "greet" {
		t.Fatal("corrected name did not save")
	}
	if _, err := store.Path("greet"); err != nil {
		t.Fatal(err)
	}
}

func TestSelectorEditUsesHighlightedMatch(t *testing.T) {
	var value string
	s := newCommandSelector([]huh.Option[string]{huh.NewOption("hello", "hello"), huh.NewOption("download", "download")}, &value)
	s.WithTheme(huh.ThemeFunc(huh.ThemeCharm))
	s.WithKeyMap(huh.NewDefaultKeyMap())
	s.Focus()
	s.Update(tea.KeyPressMsg{Code: 'd', Text: "download"})
	_, cmd := s.Update(tea.KeyPressMsg{Code: 'e', Mod: tea.ModCtrl})
	if !s.editRequested || value != "download" || cmd == nil {
		t.Fatal("Ctrl+E did not request editing the filtered match")
	}
	if cmd() != huh.NextField() {
		t.Fatal("editing did not finish the form to restore the terminal")
	}
	s.editRequested = false
	s.Update(tea.KeyPressMsg{Code: 'x', Text: "missing"})
	_, cmd = s.Update(tea.KeyPressMsg{Code: 'e', Mod: tea.ModCtrl})
	if s.editRequested || cmd != nil {
		t.Fatal("empty search requested editing a stale selection")
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
