// Command qrr runs declarative shortcut recipes.
package main

import (
	"context"
	"fmt"
	"os"

	"qrr/internal/cli"
	"qrr/internal/runner"
)

func main() {
	ctx, stop := runner.InterruptContext(context.Background())
	err := cli.Execute(ctx, os.Args[1:], os.Stdout, os.Stderr)
	cancelled := ctx.Err() != nil || runner.ExitCode(err) == 130
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
