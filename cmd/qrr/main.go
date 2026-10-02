// Command qrr runs declarative shortcut recipes.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"qrr/internal/cli"
	"qrr/internal/runner"

	"github.com/charmbracelet/huh"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	err := cli.Execute(ctx, os.Args[1:], os.Stdout, os.Stderr)
	cancelled := ctx.Err() != nil || errors.Is(err, huh.ErrUserAborted)
	stop()
	code := runner.ExitCode(err)
	if cancelled {
		code = 130
	}
	if err != nil && !cancelled {
		fmt.Fprintln(os.Stderr, "Error:", err)
	}
	os.Exit(code)
}
