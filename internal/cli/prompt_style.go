package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// writeEntryPrompt keeps redirected output plain and decorates terminal output only.
func writeEntryPrompt(out, stderr io.Writer, text string) error {
	reminder := "Copy the following prompt and send it to your AI agent:"
	if _, ok := styledTerminal(out); ok {
		text = renderEntryPrompt(lipgloss.NewRenderer(out), text)
	}
	if _, err := fmt.Fprint(stderr, reminder+"\n\n"); err != nil {
		return err
	}
	_, err := fmt.Fprintln(out, strings.TrimSuffix(text, "\n"))
	return err
}

func styledTerminal(w io.Writer) (int, bool) {
	file, ok := w.(interface{ Fd() uintptr })
	if !ok || os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return 0, false
	}
	fd := int(file.Fd())
	return fd, term.IsTerminal(fd)
}

func renderEntryPrompt(renderer *lipgloss.Renderer, text string) string {
	// Leave wrapping to the terminal so copied prompts retain one logical line.
	return renderer.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#0369A1", Dark: "#67E8F9"}).
		Render(strings.TrimSuffix(text, "\n"))
}
