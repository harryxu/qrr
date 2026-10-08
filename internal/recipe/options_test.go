package recipe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestOptionsCommandChild(t *testing.T) {
	index := -1
	for i, arg := range os.Args {
		if arg == "--" {
			index = i
			break
		}
	}
	if index < 0 {
		return
	}
	args := os.Args[index+1:]
	switch args[0] {
	case "output":
		fmt.Print(args[1])
	case "value":
		_ = json.NewEncoder(os.Stdout).Encode([]Option{{Label: "A label", Value: args[1]}})
	case "failure":
		fmt.Fprint(os.Stderr, "Docker is unavailable")
		os.Exit(7)
	case "large":
		fmt.Print(strings.Repeat("x", optionsOutputLimit+1))
	case "wait":
		time.Sleep(30 * time.Second)
	}
	os.Exit(0)
}

func optionsHelper(args ...string) []string {
	return append([]string{os.Args[0], "-test.run=^TestOptionsCommandChild$", "--"}, args...)
}

func dynamicRecipe(command []string) *Recipe {
	return &Recipe{Version: 1, Name: "demo", Type: "command",
		Params:  []Param{{Name: "target", Type: "select", Required: true, OptionsCommand: command}},
		Command: []string{"echo", "{{ .target }}"}}
}

func TestDynamicOptionsValidation(t *testing.T) {
	for _, param := range []string{
		"{name: target, type: select, options_command: [missing-program]}",
		"{name: target, type: multiselect, options_command: [missing-program], default: [one, two]}",
	} {
		if _, err := Parse([]byte(minimal + "params: [" + param + "]\n")); err != nil {
			t.Fatal(err)
		}
	}
	for _, param := range []string{
		"{name: target, type: select, options_command: []}",
		"{name: target, type: select, options_command: ['  ']}",
		"{name: target, type: select, options_command: echo}",
		"{name: target, type: input, options_command: [echo]}",
		"{name: target, type: confirm, options_command: [echo]}",
		"{name: target, type: select, options: [{value: one}], options_command: [echo]}",
		"{name: target, type: select, options: [], options_command: [echo]}",
		"{name: target, type: select, options_command: [echo], default: 1}",
		"{name: target, type: multiselect, options_command: [echo], default: [one, one]}",
		"{name: target, type: multiselect, options_command: [echo], default: ['one,two']}",
	} {
		if _, err := Parse([]byte(minimal + "params: [" + param + "]\n")); err == nil {
			t.Fatalf("accepted %s", param)
		}
	}
}

func TestPrepareDynamicOptionsAndLiteralArguments(t *testing.T) {
	value := "space ' 世界 {{ .Names }} $(echo unexpected)"
	r := dynamicRecipe(optionsHelper("value", value))
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		argv, err := r.Prepare(context.Background(), nil, func(_ context.Context, p Param, initial any) (any, error) {
			if len(p.Options) != 1 || p.Options[0].Label != "A label" || p.Options[0].Value != value || initial != "" {
				t.Fatalf("bad resolved options: %#v, initial=%v", p, initial)
			}
			return p.Options[0].Value, nil
		})
		if err != nil || !reflect.DeepEqual(argv, []string{"echo", value}) {
			t.Fatalf("argv=%v err=%v", argv, err)
		}
	}
	if r.Params[0].Options != nil || len(r.Params[0].OptionsCommand) == 0 {
		t.Fatal("resolution mutated stored recipe")
	}
	if _, err := r.Prepare(context.Background(), nil, func(context.Context, Param, any) (any, error) { return "not-listed", nil }); err == nil {
		t.Fatal("accepted a prompted value outside resolved options")
	}
}

func TestDynamicOptionsSkipForExplicitAndNonInteractive(t *testing.T) {
	r := dynamicRecipe([]string{"qrr-provider-must-not-run"})
	ask := func(context.Context, Param, any) (any, error) { t.Fatal("prompted explicit value"); return nil, nil }
	for _, prompt := range []AskFunc{nil, ask} {
		argv, err := r.Prepare(context.Background(), map[string]any{"target": "external-id"}, prompt)
		if err != nil || argv[1] != "external-id" {
			t.Fatalf("argv=%v err=%v", argv, err)
		}
	}
	if _, err := r.Prepare(context.Background(), nil, nil); err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("missing required: %v", err)
	}
	r.Params[0].Default = "default-id"
	if _, err := r.Prepare(context.Background(), nil, nil); err != nil {
		t.Fatal(err)
	}
	r.Params[0].Type, r.Params[0].Required, r.Params[0].Default = "multiselect", false, []any{"default-id"}
	r.Command = []string{"echo", `{{ join .target "," }}`}
	argv, err := r.Prepare(context.Background(), map[string]any{"target": []string{}}, ask)
	if err != nil || argv[1] != "" {
		t.Fatalf("explicit empty: %v %v", argv, err)
	}
}

func TestDynamicOptionsOutputErrors(t *testing.T) {
	for _, output := range []string{"", "null", "[]", "{}", "not json", `["one"]`, `[{"value": 1}]`,
		`[{"label":"missing value"}]`, `[{"value":"one"},{"value":"one"}]`, `[{"value":"one","unknown":true}]`,
		`[{"value":"one"}] []`, `[{"value":"one"}] trailing`} {
		t.Run(output, func(t *testing.T) {
			r := dynamicRecipe(optionsHelper("output", output))
			if _, err := r.Prepare(context.Background(), nil, func(context.Context, Param, any) (any, error) {
				t.Fatal("prompted after invalid output")
				return nil, nil
			}); err == nil {
				t.Fatal("accepted invalid output")
			}
		})
	}
	for _, mode := range []string{"failure", "large"} {
		_, err := resolveOptions(context.Background(), dynamicRecipe(optionsHelper(mode)).Params[0])
		if err == nil {
			t.Fatalf("accepted %s", mode)
		}
		if mode == "failure" && (!strings.Contains(err.Error(), "Docker is unavailable") || !strings.Contains(err.Error(), "exit status 7")) {
			t.Fatal(err)
		}
	}
}

func TestDynamicOptionsMultiselectAndDefaults(t *testing.T) {
	p := Param{Name: "targets", Type: "multiselect", OptionsCommand: optionsHelper("output", `[{"value":"one"},{"value":"two"}]`), Default: []any{"two"}}
	resolved, err := resolveOptions(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateValue(resolved, []string{"one", "two"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateValue(resolved, []string{"missing"}); err == nil {
		t.Fatal("accepted missing option")
	}
	p.Default = []any{"missing"}
	if _, err := resolveOptions(context.Background(), p); err == nil {
		t.Fatal("accepted stale default")
	}
	p.Default = nil
	p.OptionsCommand = optionsHelper("output", `[{"value":"one,two"}]`)
	if _, err := resolveOptions(context.Background(), p); err == nil {
		t.Fatal("accepted comma in multiselect value")
	}
}

func TestDynamicOptionsCancellation(t *testing.T) {
	p := Param{Name: "target", Type: "select", OptionsCommand: optionsHelper("wait")}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := resolveOptions(ctx, p); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v", err)
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("cancellation waited for the provider")
	}
}

func TestDynamicOptionsBashPipelineAndCancellation(t *testing.T) {
	p := Param{Name: "target", Type: "select", OptionsCommand: []string{"bash", "-o", "pipefail", "-c", `printf '%s' "$1" | cat`, "qrr-options", `[{"value":"one"}]`}}
	if _, err := resolveOptions(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	p.OptionsCommand = []string{"bash", "-o", "pipefail", "-c", "exit 9 | cat"}
	if _, err := resolveOptions(context.Background(), p); err == nil {
		t.Fatal("ignored failed pipeline")
	}
	p.OptionsCommand = []string{"bash", "-c", "sleep 30 & wait"}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := resolveOptions(ctx, p); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v", err)
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("cancellation waited for descendants")
	}
}
