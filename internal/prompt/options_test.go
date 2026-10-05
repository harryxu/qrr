package prompt

import (
	"reflect"
	"testing"

	"qrr/internal/recipe"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

func TestDuplicateOptionLabelsToggleIndependently(t *testing.T) {
	options := menuOptions([]recipe.Option{{Label: "Same", Value: "one"}, {Label: "Same", Value: "two"}, {Label: "Same [two]", Value: "three"}})
	var values []string
	s := &optionMultiSelect{MultiSelect: huh.NewMultiSelect[string]().Options(options...).Value(&values)}
	s.WithTheme(huh.ThemeFunc(huh.ThemeCharm))
	s.WithKeyMap(huh.NewDefaultKeyMap())
	s.Focus()
	s.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	s.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if !reflect.DeepEqual(values, []string{"two"}) {
		t.Fatalf("duplicate label toggled other values: %v", values)
	}
}
