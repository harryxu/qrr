package recipe

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const minimal = "version: 1\nname: demo\ntype: command\ncommand: [echo]\n"

func TestStrictValidation(t *testing.T) {
	cases := map[string]string{
		"unknown field":      minimal + "unknown: true\n",
		"duplicate key":      minimal + "name: another\n",
		"multiple documents": minimal + "---\n" + minimal,
		"missing reference":  strings.ReplaceAll(minimal, "[echo]", "[echo, '{{ .missing }}']"),
		"wrong default":      minimal + "params: [{name: text, type: input, default: 10}]\n",
		"wrong multiselect":  minimal + "params: [{name: languages, type: multiselect, default: [10], options: [{value: en}]}]\n",
		"unknown condition":  minimal + "optional_args: [{when: missing, args: [test]}]\n",
		"reserved name":      strings.ReplaceAll(minimal, "name: demo", "name: completion"),
		"reserved flag":      minimal + "params: [{name: dry-run, type: confirm}]\n",
		"handler":            strings.ReplaceAll(minimal, "type: command", "type: handler"),
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse([]byte(data)); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
func TestRenderPreservesBoundariesAndBranchOrder(t *testing.T) {
	r, err := Parse([]byte(`version: 1
name: demo
type: command
params:
  - {name: text, type: input}
  - name: languages
    type: multiselect
    options: [{value: en}, {value: ja}]
command: [echo, '{{ .text }}']
optional_args:
  - when: languages
    args: [--languages, '{{ join .languages "," }}']
args_tail: [--, '{{ .text }}']
`))
	if err != nil {
		t.Fatal(err)
	}
	input := "hello 世界 ' \" $(touch /tmp/should-not-exist)"
	for _, languages := range [][]string{{"en", "ja"}, {}} {
		argv, err := r.Render(map[string]any{"text": input, "languages": languages})
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"echo", input}
		if len(languages) > 0 {
			want = append(want, "--languages", "en,ja")
		}
		want = append(want, "--", input)
		if !reflect.DeepEqual(argv, want) {
			t.Fatalf("got %#v, want %#v", argv, want)
		}
	}
}
func TestLoadSkipsInvalidAndReloads(t *testing.T) {
	dir := t.TempDir()
	commands := filepath.Join(dir, "commands")
	if err := os.Mkdir(commands, 0700); err != nil {
		t.Fatal(err)
	}
	write := func(name, data string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(commands, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("demo.yaml", minimal)
	write("broken.yaml", "bad: true")
	recipes, problems := NewStore(dir).Load()
	if len(recipes) != 1 || len(problems) != 1 {
		t.Fatalf("got %d recipes and %d errors", len(recipes), len(problems))
	}
	write("new.yaml", strings.ReplaceAll(minimal, "name: demo", "name: new"))
	recipes, _ = NewStore(dir).Load()
	if len(recipes) != 2 {
		t.Fatal("new recipe not discovered")
	}
}
