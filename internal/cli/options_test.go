package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"qrr/internal/recipe"

	"github.com/charmbracelet/x/ansi"
	"github.com/creack/pty"
	"go.yaml.in/yaml/v3"
)

func writeOptionsRecipe(t *testing.T, dir, kind string, command []string) {
	t.Helper()
	template := "{{ .target }}"
	if kind == "multiselect" {
		template = `{{ join .target "," }}`
	}
	r := recipe.Recipe{Version: 1, Name: "demo", Type: "command", Command: []string{"echo", template},
		Params: []recipe.Param{{Name: "target", Type: kind, Prompt: "Choose a target", Required: true, OptionsCommand: command}}}
	data, err := yaml.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "commands", "demo.yaml"), data, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestOptionsCommandsAreNotRunByDiscoveryOrExplicitFlags(t *testing.T) {
	dir := fixture(t)
	marker := filepath.Join(dir, "provider-ran")
	writeOptionsRecipe(t, dir, "select", []string{"bash", "-c", `touch "$1"; printf '[]'`, "qrr-provider", marker})
	for _, args := range [][]string{
		{"validate"}, {"list"}, {"show", "demo"}, {"demo", "--help"}, {"schema"},
		{"__complete", "demo", "--target", ""}, {"__complete", "d"},
		{"demo", "--target", "explicit value", "--dry-run", "--json", "--non-interactive"},
		{"demo", "--target", "explicit value", "--dry-run", "--json"},
	} {
		if _, stderr, err := invoke(dir, args...); err != nil {
			t.Fatalf("%v: %v; %s", args, err, stderr)
		}
	}
	if _, _, err := invoke(dir, "demo", "--non-interactive", "--dry-run"); err == nil {
		t.Fatal("missing required value accepted")
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("discovery ran provider: %v", err)
	}
}

func TestTerminalDynamicOptionsSearch(t *testing.T) {
	for _, kind := range []string{"select", "multiselect"} {
		t.Run(kind, func(t *testing.T) {
			dir := fixture(t)
			value := "beta value with spaces"
			outputJSON := `[{"label":"Alpha","value":"alpha"},{"label":"Beta","value":"` + value + `"}]`
			writeOptionsRecipe(t, dir, kind, []string{"bash", "-o", "pipefail", "-c", `printf '%s' "$1" | cat`, "qrr-provider", outputJSON})
			cmd := exec.Command(os.Args[0], "-test.run=^TestPromptCancellationChild$")
			cmd.Env = append(terminalTestEnv(), "QRR_PROMPT_CANCEL_CHILD=parameter", "QRR_PROMPT_CANCEL_CONFIG="+dir)
			terminal, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 100})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = cmd.Process.Kill(); _ = terminal.Close() })
			output := make(chan string, 1)
			go func() {
				var text strings.Builder
				buffer := make([]byte, 4096)
				stage := 0
				for {
					n, err := terminal.Read(buffer)
					text.Write(buffer[:n])
					plainText := ansi.Strip(text.String())
					if stage == 0 && strings.Contains(plainText, "Beta") {
						stage = 1
						if kind == "select" {
							_, _ = terminal.Write([]byte("missing\r"))
						} else {
							_, _ = terminal.Write([]byte("/Beta\r \r"))
						}
					}
					if kind == "select" && stage == 1 && strings.Contains(plainText, "No matching options") {
						stage = 2
						_, _ = terminal.Write([]byte(strings.Repeat("\x7f", len("missing")) + "Beta\r"))
					}
					if err != nil {
						output <- plainText
						return
					}
				}
			}()
			finished := make(chan error, 1)
			go func() { finished <- cmd.Wait() }()
			select {
			case err := <-finished:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(10 * time.Second):
				_ = cmd.Process.Kill()
				t.Fatalf("dynamic selection did not finish: %s", <-output)
			}
			select {
			case text := <-output:
				if !strings.Contains(text, "'echo' '"+value+"'") || strings.Contains(text, "Running:") {
					t.Fatalf("wrong dynamic selection or argv boundaries: %s", text)
				}
			case <-time.After(time.Second):
				t.Fatal("terminal did not close")
			}
		})
	}
}
