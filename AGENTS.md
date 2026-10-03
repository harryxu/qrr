# Agent instructions

## Project scope

- qrr (Quick Recipe Runner) is a Go CLI for shortcuts defined in YAML.
- Keep qrr usable as a standalone executable without Node or Python runtimes.
- Use inline terminal prompts. Do not add a full-screen TUI.
- With no command, provide a searchable recipe selector with direct keyword
  input, arrow-key navigation, and Enter to select. Return to the shell after
  execution.
- Support Linux and macOS. External programs invoked by recipes are separate
  dependencies.
- Read `README.md` and `docs/schema.md` before changing behavior. The
  implementation plan records design context; some proposed features remain
  unimplemented.

## Language and tooling

- Use English for code, comments, error messages, CLI output, and new technical
  documentation unless the user explicitly requests otherwise.
- Use the Go version specified in `go.mod` or newer.
- Before running any Node-related command, including npm, yarn, or pnpm, run
  `nvm use` from the project root. The current project does not require Node.
- Keep dependencies pinned in `go.mod` and `go.sum`. Run `go mod tidy` when
  imports or dependencies change.
- Format modified Go files with `gofmt`.

## Project structure

- `cmd/qrr/main.go`: executable entry point, signals, and exit handling.
- `internal/cli`: Cobra commands, flags, management operations, and completion.
- `internal/recipe`: YAML loading, schema validation, defaults, and argv rendering.
- `internal/prompt`: inline forms and the searchable command selector.
- `internal/runner`: subprocess execution and exit-code handling.
- `examples/commands`: shareable recipe examples.
- `docs/schema.md`: the implemented configuration contract.
- Keep the entry point small and implementation packages under `internal`.
  Do not add a public `pkg` directory without a concrete public API requirement.

## Adding or changing recipes

- Prefer adding YAML over changing Go code for ordinary shortcuts.
- Follow `docs/schema.md` and existing examples. Use `version: 1` and
  `type: command`; handlers and workflows are not implemented.
- Match recipe names to filenames. Do not reuse CLI reserved names or global
  flag names for parameters.
- Declare parameter types, prompts, defaults, options, and required values
  explicitly where applicable.
- Keep numeric-looking option values as quoted strings. Multi-select option
  values must not contain commas.
- Execute argv arrays directly. Do not concatenate user input into `sh -c`
  commands or introduce implicit shell evaluation.
- Use `--` before user-controlled positional arguments when the target program
  supports it.
- Do not put passwords, tokens, cookies, or other credentials in shared recipes.
- Only propose a handler or extension when the implemented schema cannot
  reasonably express the requirement.

## Behavior to preserve

- Explicit flags override defaults. Preserve the difference between omitted
  values, explicit false, and explicit empty multi-select values.
- Non-interactive execution must never wait for input. Fail on missing required
  values.
- Configuration validation and shell completion must not run recipes, open
  prompts, or access the network.
- Reload recipes on each invocation so added commands and flags appear in
  completion without regenerating shell scripts.
- Invalid recipes must not prevent valid recipes from being used. Report their
  errors through validation and keep completion output free of warnings.
- Never submit a stale selection when filtering produces no matches.
- Render each template into exactly one argv element. Preserve optional argument
  order and fail on missing parameter references.
- Preserve subprocess streams and exit codes. Ctrl+C must cancel execution and
  restore terminal state.
- Do not claim subtitle failure recovery for the yt-dlp example. Its current
  behavior follows yt-dlp, and live YouTube behavior has not been verified.

## Verification

For Go changes, run the relevant tests and these checks before completing work:

```sh
go test ./...
go vet ./...
go build -o bin/qrr ./cmd/qrr
```

For recipe changes, validate examples and inspect the resulting argv:

```sh
./bin/qrr --config-dir ./examples validate
./bin/qrr --config-dir ./examples hello --name Test --non-interactive --dry-run --json
```

- Supply required flags when inspecting other recipes with non-interactive
  dry-run. Do not invoke real downloads as part of routine automated tests.
- Add regression tests for behavior changes, especially parameter merging,
  schema errors, argv boundaries, completion, empty filter results, cancellation,
  and exit codes.
- For prompt changes, also verify keyboard interaction in a real terminal.
- Keep README and schema documentation consistent with implemented behavior.
- Build artifacts belong in `bin` or `dist` and must remain ignored by Git.
