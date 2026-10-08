# Recipe schema (version 1)

This is the implemented YAML contract, also embedded in `qrr schema` so it is
available with a standalone binary. Only `version: 1` and `type: command` are
supported.

## File layout and names

Store one recipe per `<name>.yaml` or `<name>.yml` in the configuration directory's
`commands` folder. Names must match filenames and use lowercase letters, digits,
and hyphens, beginning with a letter. For example,
`--config-dir ./recipes` loads `./recipes/commands/greet.yaml`.

CLI command names are reserved: `run`, `list`, `show`, `add`, `edit`, `rename`,
`remove`, `validate`, `completion`, `help`, `prompt`, and `schema`.
Unknown fields, duplicate YAML keys, duplicate parameter names, multiple YAML
documents, and unsupported versions are errors.

## Minimal recipe

Save this as `commands/greet.yaml` inside your configuration directory:

```yaml
version: 1
name: greet
description: Print a greeting
type: command
params:
  - name: person
    type: input
    prompt: Your name
    default: World
command:
  - echo
  - 'Hello, {{ .person }}!'
```

Inspect the result without running `echo`:

```sh
qrr greet --person 'Jane Doe' --non-interactive --dry-run --json
# Expected stdout: ["echo","Hello, Jane Doe!"]
```

## Recipe fields

| Field | Required | Contract |
| --- | --- | --- |
| `version` | Yes | Integer `1` |
| `name` | Yes | Valid, unreserved name matching the filename |
| `description` | No | String shown in the selector, list, and command help |
| `type` | Yes | String `command` |
| `params` | No | Parameter array, processed in declaration order |
| `command` | Yes | Nonempty argv array; first entry must name an executable after rendering |
| `optional_args` | No | Array of `{when, args}` groups appended when the named parameter is nonempty or true |
| `args_tail` | No | Argv array appended after all included optional groups |

Argv entries are strings. `optional_args[].when` must name a declared parameter;
`optional_args[].args` is an argv array. Unknown fields are rejected at every
level. See [Execution and templates](#execution-and-templates) for rendering rules.

## Parameters

| Field | Contract |
| --- | --- |
| `name` | Required; lowercase letters, digits, and hyphens, beginning with a letter |
| `type` | Required; `input`, `select`, `multiselect`, or `confirm` |
| `prompt` | Optional string; defaults to the parameter name when omitted or empty |
| `required` | Optional boolean; defaults to `false` |
| `default` | Optional value with the type specified below |
| `options` | Static option array for `select` or `multiselect` |
| `options_command` | Provider argv for `select` or `multiselect`; mutually exclusive with `options` |

Parameter names must be unique and cannot be `help`, `config-dir`,
`non-interactive`, `dry-run`, or `json`. Recipe command names are a separate
reserved set; parameter names use the same syntax but only the flag restrictions
above apply. Selection types require nonempty static `options` or a nonempty
`options_command`. Neither field is supported for `input` or `confirm`.

| Type | YAML default example | Value when no default is set | CLI form |
| --- | --- | --- | --- |
| `input` | `default: World` | Empty string | `--person 'Jane Doe'` |
| `select` | `default: high` | Empty string | `--quality high` |
| `multiselect` | `default: [en, ja]` | Empty array | `--languages en,ja` or `--languages=` |
| `confirm` | `default: true` | `false` | `--enabled` or `--enabled=false` |

Option entries contain `value` and optional `label`. Values must be nonempty and
unique. Quote numeric-looking values such as `"1440"`. Nonempty static selection
values must occur in the option list. Multiselect values cannot repeat. Defaults must have the
correct types. Without a default, values start as an empty string, empty array, or
false. At execution time, required values cannot be empty; a required confirm
must be true. A required parameter may have no default, allowing the caller to
supply its value later.

Explicit flags override defaults, including `false`, an empty string, and an
empty multiselect. In an interactive terminal, omitted parameters are prompted
with their defaults. With `--non-interactive` or non-terminal stdin/stdout,
omitted parameters use defaults without prompting.

Comma-separated multi-select values do not support commas inside individual option
values. Input values are not trimmed or evaluated as shell expressions.

### Dynamic options

For `select` and `multiselect`, replace `options` with `options_command`, a
nonempty argv array. The executable can be a path or a command found on PATH.
Arguments are passed literally: no qrr template expansion, variable expansion,
or implicit shell evaluation occurs. Paths are resolved from the invocation's
working directory. Providers inherit the environment and receive EOF on stdin.
For pipelines or multiline scripts, invoke the shell explicitly:

```yaml
params:
  - name: container
    type: select
    prompt: Select a running container
    required: true
    options_command:
      - bash
      - -o
      - pipefail
      - -c
      - |
        docker container ls --filter status=running --format json |
          jq -s 'map({label: (.Names + " · " + .Image), value: .ID})'
```

Stdout must contain exactly one JSON array of objects with a nonempty string
`value` and optional string `label`. Unknown fields, duplicate values, malformed
JSON, and comma-containing multiselect values are rejected. An omitted or empty
label displays the value. For example:

```json
[{"label":"web · nginx:alpine","value":"a1b2c3d4"}]
```

The provider runs once immediately before prompting for an omitted parameter.
The returned options validate the interactive choice and any configured default.
An empty array or a default absent from the current options produces an error.
Single-select menus accept direct keyword filtering; multiselect menus use `/`
to filter, Esc to leave filtering, and Space to toggle a choice.

Explicit parameter flags skip both the provider and the prompt. Non-interactive
execution uses supplied values or defaults and never runs providers. These values
are checked for type, requiredness, and multiselect uniqueness, but cannot be
checked against a live option list. Validation, help, schema output, and shell
completion never execute providers; completion offers no dynamic option values.
Interactive `--dry-run` still runs providers to collect choices, then prints the
recipe's argv without executing it. Use explicit flags or `--non-interactive` for
a dry-run without provider execution.

Providers have a 30-second timeout and a 1 MiB stdout limit. Failures include up
to 16 KiB of captured stderr; successful stderr is discarded. Ctrl+C cancels the
query and its process group. Provider scripts should only retrieve options;
diagnostic text belongs on stderr. Programs such as Bash, Docker, and jq are
external dependencies of the recipe, not dependencies of the qrr executable.

## Execution and templates

`command` is a nonempty argv array. Each entry is rendered independently with Go
`text/template`. Use `{{ .person }}` to reference a parameter and
`{{ join .languages "," }}` to join a multi-select value. Missing parameter references
fail validation. Templates have no file-reading or command-execution functions.

### Conditional arguments: complete example

Save this as `commands/fetch.yaml`. It includes all four parameter types and
places the URL after optional flags:

```yaml
version: 1
name: fetch
description: Download a video with optional subtitles
type: command
params:
  - name: url
    type: input
    prompt: Video URL
    required: true
  - name: resolution
    type: select
    prompt: Maximum resolution
    default: "1080"
    options:
      - {label: 1080p, value: "1080"}
      - {label: 720p, value: "720"}
  - name: languages
    type: multiselect
    prompt: Subtitle languages
    default: [en]
    options:
      - {label: English, value: en}
      - {label: Japanese, value: ja}
  - name: playlist
    type: confirm
    prompt: Download the playlist?
    default: false
command:
  - yt-dlp
  - -f
  - 'best[height<={{ .resolution }}]'
optional_args:
  - when: playlist
    args:
      - --yes-playlist
  - when: languages
    args:
      - --write-subs
      - --sub-langs
      - '{{ join .languages "," }}'
args_tail:
  - --
  - '{{ .url }}'
```

`optional_args` appends conditional argv groups in declaration order.
`when` names a parameter and includes the group when its value is nonempty or true.
The final order is `command`, included `optional_args`, then `args_tail`. Template
output is never split into more arguments. A shell is never invoked implicitly.
Use `--` before untrusted positional input when the target program supports it.

Inspect the example without installing or running yt-dlp:

```sh
qrr fetch --url 'https://example.com/video' --languages= --playlist=false --non-interactive --dry-run --json
# Expected stdout: ["yt-dlp","-f","best[height\u003c=1080]","--","https://example.com/video"]
```

The JSON escape `\u003c` represents `<`; it does not change the argument passed
to the program. Both optional groups are omitted because their values are empty
or false.
Without `--languages=`, the default `[en]` includes the subtitle group. A rendered
empty string still occupies one argv element; use `optional_args` to omit a group.
Templates in every group are checked during validation, even when the group
would be omitted at execution time.

## Validation and supported behavior

Only `type: command` is implemented. Handler fields and other workflow extensions
are rejected. Validation is local and does not check installed programs or access
the network. Use `--dry-run --json --non-interactive` to inspect the rendered argv.

Keep shared recipes free of passwords, tokens, cookies, and other credentials.
Before execution, qrr prints the complete rendered command and parameter values
to stderr. Explicitly invoked shell scripts should not concatenate user input
into shell source; qrr does not add shell escaping to template values.
