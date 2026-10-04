package recipe

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func preparationRecipe(t *testing.T) *Recipe {
	t.Helper()
	r, err := Parse([]byte(`version: 1
name: demo
type: command
params:
  - {name: text, type: input, required: true}
  - {name: enabled, type: confirm, default: true}
  - name: languages
    type: multiselect
    default: [en]
    options: [{value: en}, {value: ja}]
command: [echo, '{{ .text }}']
optional_args:
  - {when: enabled, args: [--enabled]}
  - {when: languages, args: [--languages, '{{ join .languages "," }}']}
args_tail: [--, '{{ .text }}']
`))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestPrepareExplicitValuesAndDefaults(t *testing.T) {
	r := preparationRecipe(t)
	text := "a b '世界' $(echo unexpected)"
	for _, tc := range []struct {
		name   string
		values map[string]any
		want   []string
	}{
		{"defaults", map[string]any{"text": text}, []string{"echo", text, "--enabled", "--languages", "en", "--", text}},
		{"explicit false and empty", map[string]any{"text": text, "enabled": false, "languages": []string{}}, []string{"echo", text, "--", text}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := r.Prepare(context.Background(), tc.values, nil)
			if err != nil || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %#v, %v; want %#v", got, err, tc.want)
			}
		})
	}
}

func TestPreparePromptsOnlyOmittedValuesInOrder(t *testing.T) {
	r := preparationRecipe(t)
	var asked []string
	ask := func(ctx context.Context, p Param, value any) (any, error) {
		asked = append(asked, p.Name)
		switch p.Name {
		case "text":
			if value != "" {
				t.Fatalf("unexpected text default: %v", value)
			}
			return "answer", nil
		case "languages":
			if !reflect.DeepEqual(value, []string{"en"}) {
				t.Fatalf("unexpected languages default: %v", value)
			}
			return []string{"ja"}, nil
		default:
			t.Fatalf("prompted explicit parameter %s", p.Name)
			return nil, nil
		}
	}
	got, err := r.Prepare(context.Background(), map[string]any{"enabled": false}, ask)
	want := []string{"echo", "answer", "--languages", "ja", "--", "answer"}
	if err != nil || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(asked, []string{"text", "languages"}) {
		t.Fatalf("got %#v, %v; asked %v", got, err, asked)
	}
}

func TestPrepareRejectsInvalidValues(t *testing.T) {
	r := preparationRecipe(t)
	for _, values := range []map[string]any{
		{}, {"text": ""}, {"text": false},
		{"text": "ok", "languages": []string{"missing"}},
		{"text": "ok", "languages": []string{"en", "en"}},
	} {
		if argv, err := r.Prepare(context.Background(), values, nil); err == nil || argv != nil {
			t.Fatalf("accepted %v: argv=%v err=%v", values, argv, err)
		}
	}
	if _, err := r.Prepare(context.Background(), nil, func(context.Context, Param, any) (any, error) { return "", nil }); err == nil {
		t.Fatal("accepted an empty prompted required value")
	}
}

func TestPrepareStopsOnPromptFailureAndCancellation(t *testing.T) {
	r := preparationRecipe(t)
	for _, failure := range []error{context.Canceled, errors.New("prompt failed")} {
		calls := 0
		argv, err := r.Prepare(context.Background(), nil, func(context.Context, Param, any) (any, error) {
			calls++
			return nil, failure
		})
		if !errors.Is(err, failure) || argv != nil || calls != 1 {
			t.Fatalf("got argv=%v err=%v calls=%d", argv, err, calls)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := r.Prepare(ctx, nil, func(context.Context, Param, any) (any, error) {
		t.Fatal("prompted after cancellation")
		return nil, nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	_, err = r.Prepare(ctx, map[string]any{"text": "ok", "enabled": false}, func(context.Context, Param, any) (any, error) {
		cancel()
		return []string{}, nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("rendered after prompt cancellation: %v", err)
	}
}
