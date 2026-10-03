# Dependencies


The interactive CLI uses [Huh v2](https://github.com/charmbracelet/huh),
[Bubble Tea v2](https://github.com/charmbracelet/bubbletea),
[Bubbles v2](https://github.com/charmbracelet/bubbles), and
[Lip Gloss v2](https://github.com/charmbracelet/lipgloss). Huh manages selection
and scrolls only enough to keep the active option visible. qrr adds a small
adapter for immediate keyword filtering and empty-result protection.
Dependency versions are pinned in `go.mod`, with checksums recorded in `go.sum`.
The first `require` block lists modules imported directly by qrr. The second lists
indirect dependencies needed by those modules, including relevant upstream tests
and platform-specific packages. The separate blocks improve readability; Go
treats their requirements together. Run `go mod tidy` after changing dependencies
to remove unused requirements and update checksums.

Dependency upgrades target the latest stable release, including maintained major
versions. Keep PTY at `github.com/creack/pty v1.1.24` and YAML at
`go.yaml.in/yaml/v3 v3.0.5`: PTY v2 is an older release line, and YAML v4 is still
an RC. Do not adopt alpha, beta, or RC releases. Modules with no tagged releases
(such as `google/shlex` and Bubble Tea's `ultraviolet` dependency) remain pinned
to pseudo-versions; there is no stable release to select for those modules.
`go list -m -u all` checks existing module paths only. Check upstream release
history separately before switching major versions.
