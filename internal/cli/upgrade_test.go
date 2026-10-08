package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/spf13/cobra"
)

func TestUpgradeCommandOnlyRunsExplicitly(t *testing.T) {
	for _, tc := range []struct {
		args    []string
		wantErr bool
	}{
		{[]string{"upgrade", "--help"}, false},
		{[]string{"upgrade", "extra"}, true},
		{[]string{"upgrade", "--dry-run"}, true},
		{[]string{"upgrade", "--json"}, true},
		{[]string{"__complete", "upgrade", ""}, false},
		{[]string{"__complete", "up"}, false},
		{[]string{"validate"}, false},
	} {
		root, err := New([]string{"--config-dir", fixture(t)})
		if err != nil {
			t.Fatal(err)
		}
		for _, cmd := range root.Commands() {
			if cmd.Name() == "upgrade" {
				root.RemoveCommand(cmd)
			}
		}
		called := false
		root.AddCommand(newUpgradeCommand(func(context.Context, string, io.Writer) error { called = true; return nil }))
		root.SetArgs(tc.args)
		root.SetOut(io.Discard)
		root.SetErr(io.Discard)
		err = root.Execute()
		if (err != nil) != tc.wantErr {
			t.Fatalf("%v: unexpected error %v", tc.args, err)
		}
		if called {
			t.Fatalf("upgrade executed for %v", tc.args)
		}
	}
}

func TestUpgradeCommandPassesVersionContextAndStreams(t *testing.T) {
	var output bytes.Buffer
	ctx := context.WithValue(context.Background(), struct{}{}, "upgrade context")
	wantErr := errors.New("upgrade failed")
	cmd := newUpgradeCommand(func(gotCtx context.Context, gotVersion string, out io.Writer) error {
		if gotCtx != ctx || gotVersion != version {
			t.Fatalf("context or version lost: %v %q", gotCtx, gotVersion)
		}
		_, _ = io.WriteString(out, "upgrade output")
		return wantErr
	})
	root := &cobra.Command{Use: "qrr", SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(cmd)
	root.SetOut(&output)
	root.SetErr(io.Discard)
	root.SetArgs([]string{"upgrade"})
	if err := root.ExecuteContext(ctx); !errors.Is(err, wantErr) || output.String() != "upgrade output" {
		t.Fatalf("output=%q err=%v", output.String(), err)
	}
}
