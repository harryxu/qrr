# Dependencies

qrr compiles to a standalone Go executable. Users do not need Go, Node, or Python
to run the built binary. Programs named by recipes and dynamic option providers
are separate dependencies; for example, the Docker recipe needs Bash, Docker,
and jq on the host, plus the selected `sh` or `bash` inside the container.

## Direct dependency roles

Exact versions are pinned in [go.mod](../go.mod); checksums are recorded in
[go.sum](../go.sum). Use those files as the version reference rather than copying
versions into this table.

| Module | Role in qrr |
| --- | --- |
| `github.com/spf13/cobra` | Commands, flags, help, and shell completion |
| `go.yaml.in/yaml/v3` | Decode and validate recipe YAML; preserve YAML structure during rename |
| `charm.land/huh/v2` | Inline parameter forms and selection menus |
| `charm.land/bubbletea/v2` | Terminal event handling for selector and option adapters |
| `charm.land/bubbles/v2` | Keyboard bindings for the recipe selector |
| `charm.land/lipgloss/v2` | Style the short AI entry prompt |
| `github.com/creack/pty` | Relay terminal stderr with terminal-aware progress and resizing; PTY tests |
| `github.com/google/shlex` | Parse `$EDITOR` into an executable and arguments |
| `golang.org/x/sys` | Unix terminal and signal operations |
| `golang.org/x/term` | Detect terminals and manage terminal state |

Huh manages selection and scrolling. qrr adds adapters for direct keyword
filtering, empty-result protection, and selector rename/edit actions. These
libraries support inline interaction; qrr does not use a full-screen UI.

The first `require` block in `go.mod` lists direct imports. The second lists
indirect dependencies needed by the package graph, including upstream tests and
platform-specific packages. Go treats both blocks together.

## Upgrade policy

Follow [AGENTS.md](../AGENTS.md) when changing dependencies:

- Target stable releases in maintained release lines; do not adopt alpha, beta,
  or RC versions.
- Keep `github.com/creack/pty` at `v1.1.24` and `go.yaml.in/yaml/v3` at `v3.0.5`
  until a newer stable maintained release is available. A higher major number
  alone does not establish that a release line is newer or maintained.
- Preserve pinned pseudo-versions for modules without tagged stable releases,
  and report that limitation when reviewing upgrades. This includes the
  currently pinned `google/shlex` and `ultraviolet` dependencies.
- Check upstream release history before changing a major module path.
  `go list -m -u all` checks existing paths and does not discover new majors.

## Upgrade and verify

From the project root:

1. Inspect `go.mod` and run `go list -m -u all` to identify candidates. This
   command may access module servers; it is a maintainer operation, not recipe
   validation or completion.
2. Check upstream release history and maintenance status for each candidate.
   Select explicit versions before updating requirements.
3. Update the selected dependencies and imports, then run `go mod tidy`.
4. Format modified Go files with `gofmt` and verify:

```sh
go test ./...
go vet ./...
go build -o bin/qrr ./cmd/qrr
./bin/qrr --config-dir ./examples validate
./bin/qrr --config-dir ./examples hello --name Test --non-interactive --dry-run --json
```

For terminal-library changes, also check keyboard interaction and Ctrl+C in a
real terminal. Keep build artifacts in ignored `bin/` or `dist/` directories.
Do not run real downloads as routine verification.
