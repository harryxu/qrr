package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestCommandPreviewChild(t *testing.T) {
	if os.Getenv("QRR_PREVIEW_CHILD") != "1" {
		return
	}
	if _, err := os.Stat(os.Getenv("QRR_PREVIEW_MARKER")); err != nil {
		os.Exit(29)
	}
	os.Exit(0)
}

type previewWriter struct {
	bytes.Buffer
	marker string
	fail   bool
}

func (w *previewWriter) Write(p []byte) (int, error) {
	if w.fail {
		return 0, io.ErrClosedPipe
	}
	if err := os.WriteFile(w.marker, p, 0600); err != nil {
		return 0, err
	}
	return w.Buffer.Write(p)
}

func TestCommandPreviewBeforeExecution(t *testing.T) {
	dir := fixture(t)
	argv := []string{os.Args[0], "-test.run=^TestCommandPreviewChild$", "--", "a b '世界' $(echo test)"}
	data, err := yaml.Marshal(map[string]any{"version": 1, "name": "preview", "type": "command", "command": argv})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "commands", "preview.yaml"), data, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("QRR_PREVIEW_CHILD", "1")
	for _, command := range [][]string{{"preview"}, {"run", "preview"}} {
		t.Run(command[0], func(t *testing.T) {
			marker := filepath.Join(t.TempDir(), "preview-written")
			t.Setenv("QRR_PREVIEW_MARKER", marker)
			writer := &previewWriter{marker: marker}
			args := append([]string{"--config-dir", dir, "--non-interactive"}, command...)
			if err := Execute(context.Background(), args, io.Discard, writer); err != nil {
				t.Fatal(err)
			}
			want := fmt.Sprintln("Running:", commandText(argv))
			if writer.String() != want {
				t.Fatalf("got preview %q, want %q", writer.String(), want)
			}
			writer.fail = true
			if err := Execute(context.Background(), args, io.Discard, writer); err != io.ErrClosedPipe {
				t.Fatalf("ignored preview output failure: %v", err)
			}
		})
	}
	out, stderr, err := invoke(dir, "preview", "--dry-run", "--json", "--non-interactive")
	if err != nil || stderr != "" || len(out) == 0 {
		t.Fatalf("dry-run output changed: %q %q %v", out, stderr, err)
	}
}
