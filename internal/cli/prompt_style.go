package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"golang.org/x/term"
)

// writeEntryPrompt keeps redirected output plain and decorates terminal output only.
func writeEntryPrompt(out, stderr io.Writer, text string) error {
	reminder := "Copy the following prompt and send it to your AI agent:"
	if _, ok := styledTerminal(out); ok {
		text = renderEntryPrompt(text)
	}
	if isTerminalOutput(out) {
		if _, err := fmt.Fprint(stderr, reminder+"\n\n"); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(out, strings.TrimSuffix(text, "\n"))
	return err
}

func isTerminalOutput(w io.Writer) bool {
	file, ok := w.(interface{ Fd() uintptr })
	return ok && term.IsTerminal(int(file.Fd()))
}

func styledTerminal(w io.Writer) (int, bool) {
	file, ok := w.(interface{ Fd() uintptr })
	if !ok || os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return 0, false
	}
	fd := int(file.Fd())
	return fd, term.IsTerminal(fd)
}

func renderEntryPrompt(text string) string {
	// Leave wrapping to the terminal so copied prompts retain one logical line.
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#38BDF8")).
		Render(strings.TrimSuffix(text, "\n"))
}
