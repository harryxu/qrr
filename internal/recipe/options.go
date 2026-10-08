package recipe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

const optionsTimeout = 30 * time.Second
const optionsOutputLimit = 1024 * 1024

// resolveOptions returns a copy with concrete options, leaving the stored recipe
// unchanged. Provider arguments are literal argv, never recipe templates.
func resolveOptions(ctx context.Context, p Param) (Param, error) {
	if len(p.OptionsCommand) == 0 {
		return p, nil
	}
	ctx, cancel := context.WithTimeout(ctx, optionsTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, p.OptionsCommand[0], p.OptionsCommand[1:]...)
	cleanup := configureOptionsProcess(cmd)
	cmd.WaitDelay = time.Second
	stdout := &limitedOutput{limit: optionsOutputLimit}
	stderr := &limitedOutput{limit: 16 * 1024}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	// Stdin is left nil, so providers receive EOF instead of consuming terminal input.
	err := cmd.Run()
	cleanup()
	if ctx.Err() != nil {
		return p, fmt.Errorf("%s: options_command: %w", p.Name, ctx.Err())
	}
	if err != nil {
		detail := strings.TrimSpace(stderr.String())
		if stderr.overflow {
			detail += " [truncated]"
		}
		if detail != "" {
			return p, fmt.Errorf("%s: options_command: %w: %s", p.Name, err, detail)
		}
		return p, fmt.Errorf("%s: options_command: %w", p.Name, err)
	}
	if stdout.overflow {
		return p, fmt.Errorf("%s: options_command output exceeds 1 MiB", p.Name)
	}
	decoder := json.NewDecoder(bytes.NewReader(stdout.Bytes()))
	decoder.DisallowUnknownFields()
	var options []Option
	if err := decoder.Decode(&options); err != nil {
		return p, fmt.Errorf("%s: options_command must output a JSON option array: %w", p.Name, err)
	}
	if options == nil {
		return p, fmt.Errorf("%s: options_command must output a JSON option array, not null", p.Name)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return p, fmt.Errorf("%s: options_command must output exactly one JSON array", p.Name)
	}
	if len(options) == 0 {
		return p, fmt.Errorf("%s: options_command returned no options", p.Name)
	}
	p.Options, p.OptionsCommand = options, nil
	if err := validateOptions(p); err != nil {
		return p, err
	}
	if p.Default != nil {
		check := p
		check.Required = false
		if err := ValidateValue(check, DefaultValue(p)); err != nil {
			return p, fmt.Errorf("%s: default does not match current options: %w", p.Name, err)
		}
	}
	return p, nil
}

// limitedOutput drains the stream while bounding retained output.
type limitedOutput struct {
	buffer   bytes.Buffer
	limit    int
	overflow bool
}

func (b *limitedOutput) Write(p []byte) (int, error) {
	n := len(p)
	remaining := b.limit - b.buffer.Len()
	if n > remaining {
		b.overflow = true
		p = p[:remaining]
	}
	_, _ = b.buffer.Write(p)
	return n, nil
}

func (b *limitedOutput) String() string { return b.buffer.String() }
func (b *limitedOutput) Bytes() []byte  { return b.buffer.Bytes() }
